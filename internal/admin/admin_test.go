package admin

import (
	"bytes"
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

func TestAdminUpstreamsReplace(t *testing.T) {
	cfg := config.Default()
	api := (&API{Cfg: cfg, Tracker: health.NewTracker(), Mgr: upstream.NewManager(cfg)}).Handler()
	body := bytes.NewBufferString(`{"upstreams":[{"id":"cloud","name":"Cloud","base_url":"https://example.com","secret_ref":"ref","enabled":true}]}`)
	r := httptest.NewRequest(http.MethodPost, "/admin/upstreams", body)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	r = httptest.NewRequest(http.MethodGet, "/admin/upstreams", nil)
	w = httptest.NewRecorder()
	api.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"id":"cloud"`)) {
		t.Fatalf("expected updated upstreams, got %s", w.Body.String())
	}
}
