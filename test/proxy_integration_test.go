package test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ollama-switchboard/ollama-switchboard/internal/config"
	"github.com/ollama-switchboard/ollama-switchboard/internal/health"
	"github.com/ollama-switchboard/ollama-switchboard/internal/proxy"
	"github.com/ollama-switchboard/ollama-switchboard/internal/upstream"
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
