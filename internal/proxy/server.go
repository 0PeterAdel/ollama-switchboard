package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ollama-switchboard/ollama-switchboard/internal/config"
	"github.com/ollama-switchboard/ollama-switchboard/internal/health"
	"github.com/ollama-switchboard/ollama-switchboard/internal/replay"
	"github.com/ollama-switchboard/ollama-switchboard/internal/retry"
	"github.com/ollama-switchboard/ollama-switchboard/internal/router"
	"github.com/ollama-switchboard/ollama-switchboard/internal/storage"
	"github.com/ollama-switchboard/ollama-switchboard/internal/upstream"
)

type Server struct {
	cfg     config.Config
	log     *slog.Logger
	client  *http.Client
	mgr     *upstream.Manager
	tracker *health.Tracker
	secrets map[string]string
}

func New(cfg config.Config, log *slog.Logger, mgr *upstream.Manager, tracker *health.Tracker, secrets map[string]string) *Server {
	return &Server{cfg: cfg, log: log, mgr: mgr, tracker: tracker, secrets: secrets, client: &http.Client{Timeout: cfg.Retry.AttemptTimeout}}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", s.handleProxy)
	return mux
}

func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	s.tracker.IncRequest()
	captured, err := replay.Capture(r)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	decision := router.Decide(s.cfg, r.URL.Path, captured.Body)
	if decision.RewrittenTo != decision.Model {
		captured.Body = router.RewriteModel(captured.Body, decision.RewrittenTo)
	}

	stream := bytes.Contains(captured.Body, []byte(`"stream":true`))
	if stream {
		s.tracker.IncStreaming()
	} else {
		s.tracker.IncNonStreaming()
	}

	if decision.Target == "local" {
		s.tracker.IncLocal()
		if err := s.forwardSingle(w, captured, s.cfg.LocalUpstream, "", stream); err != nil {
			http.Error(w, "local upstream unavailable", http.StatusBadGateway)
		}
		return
	}
	s.tracker.IncCloud()
	if err := s.forwardCloudWithFailover(w, captured, stream); err != nil {
		http.Error(w, "all upstreams unavailable", http.StatusBadGateway)
	}
}

func (s *Server) forwardCloudWithFailover(w http.ResponseWriter, c replay.CapturedRequest, stream bool) error {
	policy := retry.Policy{MaxAttempts: s.cfg.Retry.MaxAttempts, Base: s.cfg.Retry.BackoffBase, Max: s.cfg.Retry.BackoffMax}
	attempted := map[string]bool{}
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		u, err := s.mgr.NextEligible()
		if err != nil {
			return err
		}
		if attempted[u.Config.ID] {
			continue
		}
		attempted[u.Config.ID] = true
		auth := s.secrets[u.Config.SecretRef]
		if auth == "" {
			s.mgr.MarkResult(u.Config.ID, "auth_invalid", s.cfg.Retry.CooldownDuration)
			continue
		}
		err = s.forwardSingle(w, c, u.Config.BaseURL, auth, stream)
		if err == nil {
			s.mgr.MarkResult(u.Config.ID, "", s.cfg.Retry.CooldownDuration)
			return nil
		}
		typ := err.Error()
		s.mgr.MarkResult(u.Config.ID, typ, s.cfg.Retry.CooldownDuration)
		s.tracker.IncFailover()
		time.Sleep(retry.Backoff(policy, attempt))
	}
	return fmt.Errorf("exhausted cloud attempts")
}

func (s *Server) forwardSingle(w http.ResponseWriter, c replay.CapturedRequest, baseURL, apiKey string, stream bool) error {
	req, err := c.NewRequest(baseURL)
	if err != nil {
		return err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	if strings.HasPrefix(c.Path, "/v1") && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.client.Do(req.WithContext(context.Background()))
	typ, retriable := retry.Classify(resp, err)
	if err != nil {
		return fmt.Errorf("%s", typ)
	}
	defer resp.Body.Close()
	if retriable {
		if resp.StatusCode == 429 {
			b, _ := io.ReadAll(resp.Body)
			if strings.Contains(strings.ToLower(string(b)), "quota") {
				return fmt.Errorf("quota_exhausted")
			}
		}
		return fmt.Errorf("%s", typ)
	}
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	if stream && s.cfg.Routing.StreamMode == "safe" {
		buf, _ := io.ReadAll(resp.Body)
		_, _ = w.Write(buf)
		return nil
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

func MarshalJSON(v any) []byte { b, _ := json.Marshal(v); return b }

func LoadSecrets() map[string]string {
	store, err := storage.NewSecretStore()
	if err != nil {
		return map[string]string{}
	}
	m, err := store.ReadAll()
	if err != nil {
		return map[string]string{}
	}
	return m
}
