package test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0PeterAdel/ollama-switchboard/internal/config"
	"github.com/0PeterAdel/ollama-switchboard/internal/health"
	"github.com/0PeterAdel/ollama-switchboard/internal/proxy"
	"github.com/0PeterAdel/ollama-switchboard/internal/upstream"
)

func TestProxyForwardsLocal(t *testing.T) {
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"ok":true}`)) }))
	defer local.Close()
	cfg := config.Default()
	cfg.LocalUpstream = local.URL
	p := proxy.New(cfg, slog.Default(), upstream.NewManager(cfg), health.NewTracker(), map[string]string{})
	s := httptest.NewServer(p.Handler())
	defer s.Close()
	resp, err := http.Post(s.URL+"/api/chat", "application/json", strings.NewReader(`{"model":"llama3","stream":false,"messages":[{"role":"user","content":"hi"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if string(b) != `{"ok":true}` {
		t.Fatalf("unexpected: %s", string(b))
	}
}

func TestProxyPreferLocalFallsBackToCloud(t *testing.T) {
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer local.Close()
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"cloud":true}`))
	}))
	defer cloud.Close()

	cfg := config.Default()
	cfg.LocalUpstream = local.URL
	cfg.Routing.Policy = "prefer-local"
	cfg.Upstreams = []config.UpstreamConfig{{ID: "cloud", Name: "cloud", BaseURL: cloud.URL, SecretRef: "cloud", Enabled: true}}
	p := proxy.New(cfg, slog.Default(), upstream.NewManager(cfg), health.NewTracker(), map[string]string{"cloud": "token"})
	s := httptest.NewServer(p.Handler())
	defer s.Close()

	resp, err := http.Post(s.URL+"/api/chat", "application/json", strings.NewReader(`{"model":"unmatched","stream":false}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if string(b) != `{"cloud":true}` {
		t.Fatalf("unexpected: %s", string(b))
	}
}
