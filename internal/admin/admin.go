package admin

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/ollama-switchboard/ollama-switchboard/internal/config"
	"github.com/ollama-switchboard/ollama-switchboard/internal/health"
	"github.com/ollama-switchboard/ollama-switchboard/internal/upstream"
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
	mux.HandleFunc("/admin/status", func(w http.ResponseWriter, r *http.Request) {
		if !a.authorized(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"stats": a.Tracker.Snapshot(), "upstreams": a.Mgr.Snapshot(), "listen": a.Cfg.ListenAddress, "admin": a.Cfg.AdminAddress, "stream_mode": a.Cfg.Routing.StreamMode})
	})
	mux.HandleFunc("/admin/upstreams", func(w http.ResponseWriter, r *http.Request) {
		if !a.authorized(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(a.Mgr.Snapshot())
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
	})
	return mux
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
	provided := r.Header.Get("X-OSB-Admin-Token")
	if provided == "" {
		provided = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	return provided == token
}
