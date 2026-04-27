package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultValid(t *testing.T) {
	if err := Validate(Default()); err != nil {
		t.Fatalf("default config invalid: %v", err)
	}
}

func TestLoadYAMLConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := []byte(`
version: 1
listen_address: 127.0.0.1:19034
admin_address: 127.0.0.1:19039
ui_address: 127.0.0.1:19040
local_upstream: http://127.0.0.1:11435
routing:
  policy: prefer-cloud
  stream_mode: live
  cloud_suffix: [":cloud"]
  local_regex: ["^qwen"]
  cloud_regex: ["^gpt"]
retry:
  max_attempts: 2
  attempt_timeout: 60s
  backoff_base: 300ms
  backoff_max: 3s
  cooldown_duration: 45s
upstreams: []
`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ListenAddress != "127.0.0.1:19034" {
		t.Fatalf("unexpected listen address %q", cfg.ListenAddress)
	}
	if cfg.Retry.AttemptTimeout.Std().Seconds() != 60 {
		t.Fatalf("unexpected timeout %s", cfg.Retry.AttemptTimeout)
	}
}

func TestValidateRejectsInvalidRegex(t *testing.T) {
	cfg := Default()
	cfg.Routing.CloudRegex = []string{"["}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected invalid regex error")
	}
}
