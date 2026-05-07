package router

import (
	"testing"

	"github.com/0PeterAdel/ollama-switchboard/internal/config"
)

func TestDecideCloudSuffix(t *testing.T) {
	cfg := config.Default()
	d := Decide(cfg, "/api/chat", []byte(`{"model":"llama3:cloud"}`))
	if d.Target != "cloud" {
		t.Fatalf("expected cloud, got %s", d.Target)
	}
}
