package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type SecretStore struct{ path string }

func NewSecretStore() (*SecretStore, error) {
	h, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	var dir string
	switch runtime.GOOS {
	case "windows":
		dir = filepath.Join(h, "AppData", "Local", "OllamaSwitchboard")
	case "darwin":
		dir = filepath.Join(h, "Library", "Application Support", "OllamaSwitchboard")
	default:
		dir = filepath.Join(h, ".config", "ollama-switchboard")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &SecretStore{path: filepath.Join(dir, "secrets.env")}, nil
}

func (s *SecretStore) Put(ref, value string) error {
	m, _ := s.ReadAll()
	m[ref] = value
	var b strings.Builder
	for k, v := range m {
		b.WriteString(fmt.Sprintf("%s=%s\n", k, v))
	}
	return os.WriteFile(s.path, []byte(b.String()), 0o600)
}

func (s *SecretStore) Delete(ref string) error {
	m, _ := s.ReadAll()
	delete(m, ref)
	var b strings.Builder
	for k, v := range m {
		b.WriteString(fmt.Sprintf("%s=%s\n", k, v))
	}
	return os.WriteFile(s.path, []byte(b.String()), 0o600)
}

func (s *SecretStore) ReadAll() (map[string]string, error) {
	out := map[string]string{}
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	for _, line := range strings.Split(string(b), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			out[parts[0]] = parts[1]
		}
	}
	return out, nil
}

func Fingerprint(secret string) string {
	h := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(h[:])[:12]
}
