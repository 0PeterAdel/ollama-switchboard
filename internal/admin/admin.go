package admin

import (
	"encoding/json"
	"net/http"

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
		_ = json.NewEncoder(w).Encode(map[string]any{"stats": a.Tracker.Snapshot(), "upstreams": a.Mgr.Snapshot(), "listen": a.Cfg.ListenAddress, "admin": a.Cfg.AdminAddress, "stream_mode": a.Cfg.Routing.StreamMode})
	})
	mux.HandleFunc("/admin/upstreams", func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(a.Mgr.Snapshot()) })
	return mux
}
