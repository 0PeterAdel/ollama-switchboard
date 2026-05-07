package admin

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/0PeterAdel/ollama-switchboard/internal/config"
	"github.com/0PeterAdel/ollama-switchboard/internal/health"
	"github.com/0PeterAdel/ollama-switchboard/internal/upstream"
)

type API struct {
	Cfg     config.Config
	Tracker *health.Tracker
	Mgr     *upstream.Manager
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	mux.HandleFunc("/admin/status", a.requireAdmin(a.handleStatus))
	mux.HandleFunc("/admin/upstreams", a.requireAdmin(a.handleUpstreams))
	return mux
}

func (a *API) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]any{
		"stats":       a.Tracker.Snapshot(),
		"upstreams":   a.Mgr.Snapshot(),
		"listen":      a.Cfg.ListenAddress,
		"admin":       a.Cfg.AdminAddress,
		"stream_mode": a.Cfg.Routing.StreamMode,
	})
}

func (a *API) handleUpstreams(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, a.Mgr.Snapshot())
	case http.MethodPost:
		var payload struct {
			Upstreams []config.UpstreamConfig `json:"upstreams"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		a.Mgr.Replace(payload.Upstreams)
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *API) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.authorized(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (a *API) authorized(r *http.Request) bool {
	if !a.Cfg.Security.AdminTokenRequired {
		return true
	}
	token := strings.TrimSpace(a.Cfg.Security.AdminToken)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("OSB_ADMIN_TOKEN"))
	}
	if token == "" {
		return false
	}
	provided := strings.TrimSpace(r.Header.Get("X-OSB-Admin-Token"))
	if provided == "" {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			provided = strings.TrimSpace(auth[len("Bearer "):])
		}
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(token)) == 1
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
