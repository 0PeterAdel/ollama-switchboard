package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Config struct {
	Version       int               `json:"version"`
	ListenAddress string            `json:"listen_address"`
	AdminAddress  string            `json:"admin_address"`
	UIAddress     string            `json:"ui_address"`
	LocalUpstream string            `json:"local_upstream"`
	Routing       RoutingConfig     `json:"routing"`
	Retry         RetryConfig       `json:"retry"`
	Logging       LoggingConfig     `json:"logging"`
	Security      SecurityConfig    `json:"security"`
	Upstreams     []UpstreamConfig  `json:"upstreams"`
	ModelMap      map[string]string `json:"model_map"`
}

type RoutingConfig struct {
	Policy      string   `json:"policy"`
	StreamMode  string   `json:"stream_mode"`
	CloudSuffix []string `json:"cloud_suffix"`
	LocalRegex  []string `json:"local_regex"`
	CloudRegex  []string `json:"cloud_regex"`
}

type RetryConfig struct {
	MaxAttempts      int           `json:"max_attempts"`
	AttemptTimeout   time.Duration `json:"attempt_timeout"`
	BackoffBase      time.Duration `json:"backoff_base"`
	BackoffMax       time.Duration `json:"backoff_max"`
	CooldownDuration time.Duration `json:"cooldown_duration"`
}

type LoggingConfig struct {
	Level, Format, File   string
	MaxSizeMB, MaxBackups int
}
type SecurityConfig struct {
	AdminTokenRequired bool `json:"admin_token_required"`
}

type UpstreamConfig struct {
	ID, Name, Type, BaseURL, SecretRef string
	Enabled                            bool
	Priority                           int
	Tags                               []string
	ModelRewrite                       map[string]string
}

func Default() Config {
	return Config{Version: 1, ListenAddress: "127.0.0.1:11434", AdminAddress: "127.0.0.1:11439", UIAddress: "127.0.0.1:11440", LocalUpstream: "http://127.0.0.1:11435", Routing: RoutingConfig{Policy: "auto", StreamMode: "safe", CloudSuffix: []string{":cloud"}, LocalRegex: []string{"^(llama|qwen|mistral|gemma)"}}, Retry: RetryConfig{MaxAttempts: 3, AttemptTimeout: 60 * time.Second, BackoffBase: 300 * time.Millisecond, BackoffMax: 3 * time.Second, CooldownDuration: 45 * time.Second}, Logging: LoggingConfig{Level: "info", Format: "json", MaxSizeMB: 20, MaxBackups: 5}, Security: SecurityConfig{}, Upstreams: []UpstreamConfig{}, ModelMap: map[string]string{}}
}

func ConfigDir() (string, error) {
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(h, "AppData", "Local", "OllamaSwitchboard"), nil
	case "darwin":
		return filepath.Join(h, "Library", "Application Support", "OllamaSwitchboard"), nil
	default:
		if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
			return filepath.Join(x, "ollama-switchboard"), nil
		}
		return filepath.Join(h, ".config", "ollama-switchboard"), nil
	}
}
func DefaultPath() (string, error) {
	d, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.json"), nil
}
func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	cfg := Default()
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, Validate(cfg)
}
func Save(path string, cfg Config) error {
	if err := Validate(cfg); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
func Validate(c Config) error {
	if c.ListenAddress == "" || c.AdminAddress == "" || c.LocalUpstream == "" {
		return errors.New("listen/admin/local_upstream are required")
	}
	p := strings.ToLower(c.Routing.Policy)
	allowed := map[string]bool{"auto": true, "local-only": true, "cloud-only": true, "prefer-local": true, "prefer-cloud": true}
	if !allowed[p] {
		return fmt.Errorf("invalid routing policy %q", c.Routing.Policy)
	}
	if c.Retry.MaxAttempts < 1 || c.Retry.MaxAttempts > 10 {
		return fmt.Errorf("retry.max_attempts out of range")
	}
	return nil
}
