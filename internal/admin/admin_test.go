package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollama-switchboard/ollama-switchboard/internal/config"
	"github.com/ollama-switchboard/ollama-switchboard/internal/health"
	"github.com/ollama-switchboard/ollama-switchboard/internal/upstream"
)

func TestAdminTokenProtection(t *testing.T) {
	cfg := config.Default()
	cfg.Security.AdminTokenRequired = true
	cfg.Security.AdminToken = "secret"
	api := (&API{Cfg: cfg, Tracker: health.NewTracker(), Mgr: upstream.NewManager(cfg)}).Handler()
	r := httptest.NewRequest(http.MethodGet, "/admin/status", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	r = httptest.NewRequest(http.MethodGet, "/admin/status", nil)
	r.Header.Set("X-OSB-Admin-Token", "secret")
	w = httptest.NewRecorder()
	api.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
