package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Duration time.Duration

func (d Duration) Std() time.Duration {
	return time.Duration(d)
}

func (d Duration) String() string {
	return time.Duration(d).String()
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		parsed, err := parseDuration(s)
		if err != nil {
			return err
		}
		*d = parsed
		return nil
	}
	var n int64
	if err := json.Unmarshal(b, &n); err == nil {
		*d = Duration(time.Duration(n))
		return nil
	}
	return errors.New("duration must be a string or integer nanoseconds")
}

func (d Duration) MarshalYAML() (any, error) {
	return d.String(), nil
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	parsed, err := parseDuration(value.Value)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

type Config struct {
	Version       int               `json:"version" yaml:"version"`
	ListenAddress string            `json:"listen_address" yaml:"listen_address"`
	AdminAddress  string            `json:"admin_address" yaml:"admin_address"`
	UIAddress     string            `json:"ui_address" yaml:"ui_address"`
	LocalUpstream string            `json:"local_upstream" yaml:"local_upstream"`
	Routing       RoutingConfig     `json:"routing" yaml:"routing"`
	Retry         RetryConfig       `json:"retry" yaml:"retry"`
	Logging       LoggingConfig     `json:"logging" yaml:"logging"`
	Security      SecurityConfig    `json:"security" yaml:"security"`
	Upstreams     []UpstreamConfig  `json:"upstreams" yaml:"upstreams"`
	ModelMap      map[string]string `json:"model_map" yaml:"model_map"`
}

type RoutingConfig struct {
	Policy      string   `json:"policy" yaml:"policy"`
	StreamMode  string   `json:"stream_mode" yaml:"stream_mode"`
	CloudSuffix []string `json:"cloud_suffix" yaml:"cloud_suffix"`
	LocalRegex  []string `json:"local_regex" yaml:"local_regex"`
	CloudRegex  []string `json:"cloud_regex" yaml:"cloud_regex"`
}

type RetryConfig struct {
	MaxAttempts      int      `json:"max_attempts" yaml:"max_attempts"`
	AttemptTimeout   Duration `json:"attempt_timeout" yaml:"attempt_timeout"`
	BackoffBase      Duration `json:"backoff_base" yaml:"backoff_base"`
	BackoffMax       Duration `json:"backoff_max" yaml:"backoff_max"`
	CooldownDuration Duration `json:"cooldown_duration" yaml:"cooldown_duration"`
}

type LoggingConfig struct {
	Level      string `json:"level" yaml:"level"`
	Format     string `json:"format" yaml:"format"`
	File       string `json:"file,omitempty" yaml:"file,omitempty"`
	MaxSizeMB  int    `json:"max_size_mb" yaml:"max_size_mb"`
	MaxBackups int    `json:"max_backups" yaml:"max_backups"`
}

type SecurityConfig struct {
	AdminTokenRequired bool   `json:"admin_token_required" yaml:"admin_token_required"`
	AdminToken         string `json:"admin_token,omitempty" yaml:"admin_token,omitempty"`
}

type UpstreamConfig struct {
	ID           string            `json:"id" yaml:"id"`
	Name         string            `json:"name" yaml:"name"`
	Type         string            `json:"type" yaml:"type"`
	BaseURL      string            `json:"base_url" yaml:"base_url"`
	SecretRef    string            `json:"secret_ref" yaml:"secret_ref"`
	Enabled      bool              `json:"enabled" yaml:"enabled"`
	Priority     int               `json:"priority" yaml:"priority"`
	Tags         []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
	ModelRewrite map[string]string `json:"model_rewrite,omitempty" yaml:"model_rewrite,omitempty"`
}

func Default() Config {
	return Config{
		Version:       1,
		ListenAddress: "127.0.0.1:11434",
		AdminAddress:  "127.0.0.1:11439",
		UIAddress:     "127.0.0.1:11440",
		LocalUpstream: "http://127.0.0.1:11435",
		Routing: RoutingConfig{
			Policy:      "auto",
			StreamMode:  "safe",
			CloudSuffix: []string{":cloud"},
			LocalRegex:  []string{"^(llama|qwen|mistral|gemma)"},
			CloudRegex:  []string{},
		},
		Retry: RetryConfig{
			MaxAttempts:      3,
			AttemptTimeout:   Duration(60 * time.Second),
			BackoffBase:      Duration(300 * time.Millisecond),
			BackoffMax:       Duration(3 * time.Second),
			CooldownDuration: Duration(45 * time.Second),
		},
		Logging:   LoggingConfig{Level: "info", Format: "json", MaxSizeMB: 20, MaxBackups: 5},
		Security:  SecurityConfig{},
		Upstreams: []UpstreamConfig{},
		ModelMap:  map[string]string{},
	}
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
	if err := decode(path, b, &cfg); err != nil {
		return Config{}, err
	}
	Normalize(&cfg)
	return cfg, Validate(cfg)
}

func Save(path string, cfg Config) error {
	Normalize(&cfg)
	if err := Validate(cfg); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	var (
		b   []byte
		err error
	)
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		b, err = yaml.Marshal(cfg)
	default:
		b, err = json.MarshalIndent(cfg, "", "  ")
	}
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

func Normalize(c *Config) {
	c.Routing.Policy = strings.ToLower(strings.TrimSpace(c.Routing.Policy))
	c.Routing.StreamMode = strings.ToLower(strings.TrimSpace(c.Routing.StreamMode))
	if c.Routing.CloudSuffix == nil {
		c.Routing.CloudSuffix = []string{}
	}
	if c.Routing.LocalRegex == nil {
		c.Routing.LocalRegex = []string{}
	}
	if c.Routing.CloudRegex == nil {
		c.Routing.CloudRegex = []string{}
	}
	if c.Upstreams == nil {
		c.Upstreams = []UpstreamConfig{}
	}
	if c.ModelMap == nil {
		c.ModelMap = map[string]string{}
	}
}

func Validate(c Config) error {
	if c.ListenAddress == "" || c.AdminAddress == "" || c.UIAddress == "" || c.LocalUpstream == "" {
		return errors.New("listen_address, admin_address, ui_address, and local_upstream are required")
	}
	policy := strings.ToLower(c.Routing.Policy)
	allowedPolicies := map[string]bool{"auto": true, "local-only": true, "cloud-only": true, "prefer-local": true, "prefer-cloud": true}
	if !allowedPolicies[policy] {
		return fmt.Errorf("invalid routing policy %q", c.Routing.Policy)
	}
	streamMode := strings.ToLower(c.Routing.StreamMode)
	if streamMode != "safe" && streamMode != "live" {
		return fmt.Errorf("invalid routing stream_mode %q", c.Routing.StreamMode)
	}
	if c.Retry.MaxAttempts < 1 || c.Retry.MaxAttempts > 10 {
		return errors.New("retry.max_attempts out of range")
	}
	if c.Retry.AttemptTimeout.Std() <= 0 {
		return errors.New("retry.attempt_timeout must be positive")
	}
	if c.Retry.BackoffBase.Std() <= 0 || c.Retry.BackoffMax.Std() <= 0 {
		return errors.New("retry backoff durations must be positive")
	}
	if c.Retry.CooldownDuration.Std() < 0 {
		return errors.New("retry.cooldown_duration cannot be negative")
	}
	if c.Retry.BackoffMax.Std() > 0 && c.Retry.BackoffBase.Std() > c.Retry.BackoffMax.Std() {
		return errors.New("retry.backoff_base cannot exceed retry.backoff_max")
	}
	for _, pattern := range append(append([]string{}, c.Routing.LocalRegex...), c.Routing.CloudRegex...) {
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("invalid regex %q: %w", pattern, err)
		}
	}
	if c.Security.AdminTokenRequired && strings.TrimSpace(c.Security.AdminToken) == "" && strings.TrimSpace(os.Getenv("OSB_ADMIN_TOKEN")) == "" {
		return errors.New("admin_token_required is true but no admin token was provided")
	}
	return nil
}

func decode(path string, b []byte, cfg *Config) error {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return yaml.Unmarshal(b, cfg)
	case ".json", "":
		return json.Unmarshal(b, cfg)
	default:
		jsonCfg := *cfg
		if err := json.Unmarshal(b, &jsonCfg); err == nil {
			*cfg = jsonCfg
			return nil
		}
		return yaml.Unmarshal(b, cfg)
	}
}

func parseDuration(raw string) (Duration, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, errors.New("duration cannot be empty")
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return Duration(time.Duration(n)), nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	return Duration(d), nil
}
