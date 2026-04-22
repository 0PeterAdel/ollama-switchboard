package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ollama-switchboard/ollama-switchboard/internal/config"
)

type Options struct {
	DryRun bool
	Yes    bool
}

type Result struct {
	ConfigPath string
	BackupPath string
	Notes      []string
}

func Run(path string, opts Options) (Result, error) {
	cfg := config.Default()
	res := Result{ConfigPath: path}
	if _, err := os.Stat(path); err == nil {
		backup := filepath.Join(filepath.Dir(path), fmt.Sprintf("config.backup.%d.json", time.Now().Unix()))
		b, _ := os.ReadFile(path)
		if !opts.DryRun {
			if err := os.WriteFile(backup, b, 0o600); err != nil {
				return res, err
			}
		}
		res.BackupPath = backup
		res.Notes = append(res.Notes, "existing config backup created")
	}
	if opts.DryRun {
		res.Notes = append(res.Notes, "dry-run: no files modified")
		return res, nil
	}
	if err := config.Save(path, cfg); err != nil {
		return res, err
	}
	res.Notes = append(res.Notes, "wrote default switchboard config")
	res.Notes = append(res.Notes, "set local Ollama upstream target to http://127.0.0.1:11435")
	res.Notes = append(res.Notes, "next: update Ollama OLLAMA_HOST to 127.0.0.1:11435")
	return res, nil
}
