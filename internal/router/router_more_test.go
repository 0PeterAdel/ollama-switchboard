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

func TestPreferCloudFallsBackToLocal(t *testing.T) {
	cfg := config.Default()
	cfg.Routing.Policy = "prefer-cloud"
	d := Decide(cfg, "/api/chat", []byte(`{"model":"unmatched"}`))
	if d.Target != TargetCloud {
		t.Fatalf("expected cloud target, got %s", d.Target)
	}
	if d.FallbackTarget != TargetLocal {
		t.Fatalf("expected local fallback, got %s", d.FallbackTarget)
	}
}

func TestInvalidRegexDoesNotPanic(t *testing.T) {
	cfg := config.Default()
	cfg.Routing.CloudRegex = []string{"["}
	d := Decide(cfg, "/api/chat", []byte(`{"model":"gpt-oss"}`))
	if d.Target != TargetLocal {
		t.Fatalf("expected default local target, got %s", d.Target)
	}
}
