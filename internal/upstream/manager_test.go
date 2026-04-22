package upstream

import (
	"testing"

	"github.com/ollama-switchboard/ollama-switchboard/internal/config"
)

func TestRoundRobin(t *testing.T) {
	cfg := config.Default()
	cfg.Upstreams = []config.UpstreamConfig{{ID: "a", Name: "a", Enabled: true}, {ID: "b", Name: "b", Enabled: true}}
	m := NewManager(cfg)
	u1, _ := m.NextEligible()
	u2, _ := m.NextEligible()
	if u1.Config.ID == u2.Config.ID {
		t.Fatalf("expected rotation")
	}
}
