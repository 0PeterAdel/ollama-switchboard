package router

import (
	"testing"

	"github.com/ollama-switchboard/ollama-switchboard/internal/config"
)

func TestDecideLocalRegex(t *testing.T) {
	cfg := config.Default()
	cfg.Routing.CloudRegex = []string{"^gpt"}
	cfg.Routing.LocalRegex = []string{"^qwen"}
	d := Decide(cfg, "/api/chat", []byte(`{"model":"qwen3"}`))
	if d.Target != "local" {
		t.Fatalf("expected local, got %s", d.Target)
	}
}
