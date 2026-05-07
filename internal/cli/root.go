package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/0PeterAdel/ollama-switchboard/internal/config"
	"github.com/0PeterAdel/ollama-switchboard/internal/logging"
	"github.com/0PeterAdel/ollama-switchboard/internal/platform"
	"github.com/0PeterAdel/ollama-switchboard/internal/service"
	"github.com/0PeterAdel/ollama-switchboard/internal/setup"
	"github.com/0PeterAdel/ollama-switchboard/internal/storage"
	"github.com/0PeterAdel/ollama-switchboard/internal/version"
)

func Run(args []string) error {
	cfgPath, _ := config.DefaultPath()
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		printHelp()
		return nil
	}
	switch args[0] {
	case "setup":
		return runSetup(cfgPath, args[1:])
	case "serve":
		return runServe(cfgPath)
	case "status":
		return runStatus(cfgPath, args[1:])
	case "add":
		return runAdd(cfgPath, args[1:])
	case "remove":
		return runRemove(cfgPath, args[1:])
	case "list":
		return runList(cfgPath)
	case "doctor":
		return runDoctor(cfgPath)
	case "chat":
		return runChat(cfgPath, args[1:])
	case "logs":
		fmt.Println("Use your OS service logs for now.")
		return nil
	case "version":
		fmt.Printf("osb %s commit=%s date=%s\n", version.Version, version.CommitSHA, version.BuildDate)
		return nil
	case "start":
		return execDetached(os.Args[0], "serve")
	case "stop":
		fmt.Println("Stop via service manager or terminate osb process.")
		return nil
	case "restart":
		fmt.Println("Restart not wired in v0.1; use stop/start.")
		return nil
	case "reload":
		return runReload(cfgPath)
	case "config":
		b, err := os.ReadFile(cfgPath)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	case "enable", "disable":
		fmt.Println("Use config edit for enable/disable in v0.1.")
		return nil
	case "ui":
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}
		fmt.Printf("Open http://%s\n", cfg.UIAddress)
		return nil
	case "uninstall":
		return runUninstall(cfgPath, args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func printHelp() {
	fmt.Println(`osb commands: help, setup, serve, status, add, remove, list, doctor, chat, logs, version, start, stop, restart, reload, config, enable, disable, ui, uninstall`)
}

func has(args []string, k string) bool {
	for _, a := range args {
		if a == k {
			return true
		}
	}
	return false
}

func val(args []string, k string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == k {
			return args[i+1]
		}
	}
	return ""
}

func runSetup(path string, args []string) error {
	res, err := setup.Run(path, setup.Options{DryRun: has(args, "--dry-run"), Yes: has(args, "--yes")})
	if err != nil {
		return err
	}
	fmt.Println("Setup complete", res.ConfigPath)
	for _, n := range res.Notes {
		fmt.Println("-", n)
	}
	return nil
}

func runServe(path string) error {
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	return service.New(cfg, logging.New(cfg.Logging.Level, cfg.Logging.Format)).Run(context.Background())
}

func runStatus(path string, args []string) error {
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodGet, "http://"+cfg.AdminAddress+"/admin/status", nil)
	if err != nil {
		return err
	}
	setAdminAuth(req, cfg)
	resp, err := (&http.Client{Timeout: 2 * time.Second}).Do(req)
	if err != nil {
		fmt.Println("daemon: not running")
		return nil
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if has(args, "--json") {
		fmt.Println(string(b))
		return nil
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("admin API unauthorized")
	}
	fmt.Println("daemon: running")
	fmt.Println(string(b))
	return nil
}

func runList(path string) error {
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	for _, u := range cfg.Upstreams {
		fmt.Printf("%s\t%s\t%s\n", u.ID, u.Name, u.BaseURL)
	}
	return nil
}

func runAdd(path string, args []string) error {
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	name := val(args, "--name")
	key := val(args, "--api-key")
	keyEnv := val(args, "--api-key-env")
	if key == "" && keyEnv != "" {
		key = os.Getenv(keyEnv)
	}
	if name == "" || key == "" {
		return errors.New("usage: osb add --name <n> (--api-key <k>|--api-key-env <ENV>)")
	}
	id := strings.ToLower(strings.ReplaceAll(name, " ", "-")) + "-" + fmt.Sprintf("%d", time.Now().Unix()%100000)
	ref := "upstream_" + id
	store, err := storage.NewSecretStore()
	if err != nil {
		return err
	}
	if err := store.Put(ref, key); err != nil {
		return err
	}
	cfg.Upstreams = append(cfg.Upstreams, config.UpstreamConfig{ID: id, Name: name, Type: "ollama_cloud", BaseURL: "https://ollama.com", SecretRef: ref, Enabled: true})
	if err := config.Save(path, cfg); err != nil {
		return err
	}
	reportDaemonSync(syncUpstreamsToDaemon(cfg))
	fmt.Println("added", name, "fingerprint", storage.Fingerprint(key))
	return nil
}

func runRemove(path string, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: osb remove <id-or-name> [--yes]")
	}
	target := args[0]
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	out := make([]config.UpstreamConfig, 0, len(cfg.Upstreams))
	store, err := storage.NewSecretStore()
	if err != nil {
		return err
	}
	for _, u := range cfg.Upstreams {
		if u.ID == target || u.Name == target {
			_ = store.Delete(u.SecretRef)
			continue
		}
		out = append(out, u)
	}
	cfg.Upstreams = out
	if err := config.Save(path, cfg); err != nil {
		return err
	}
	reportDaemonSync(syncUpstreamsToDaemon(cfg))
	return nil
}

func runReload(path string) error {
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	if err := syncUpstreamsToDaemon(cfg); err != nil {
		return err
	}
	fmt.Println("Reloaded upstreams")
	return nil
}

func runDoctor(path string) error {
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	fmt.Println("os:", platform.Name())
	if _, err := exec.LookPath("ollama"); err != nil {
		fmt.Println("ollama: not in PATH")
	} else {
		fmt.Println("ollama: found")
	}
	c := http.Client{Timeout: 2 * time.Second}
	if _, err := c.Get("http://" + cfg.ListenAddress + "/health"); err != nil {
		fmt.Println(cfg.ListenAddress + ": unavailable")
	} else {
		fmt.Println(cfg.ListenAddress + ": reachable")
	}
	return nil
}

func runChat(path string, args []string) error {
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	model := "llama3"
	if m := val(args, "--model"); m != "" {
		model = m
	}
	msg := "hello"
	if len(args) > 0 && !strings.HasPrefix(args[len(args)-1], "--") {
		msg = args[len(args)-1]
	}
	payload := map[string]any{"model": model, "stream": false, "messages": []map[string]string{{"role": "user", "content": msg}}}
	b, _ := json.Marshal(payload)
	resp, err := http.Post("http://"+cfg.ListenAddress+"/api/chat", "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	fmt.Println(string(rb))
	return nil
}

func runUninstall(path string, args []string) error {
	if has(args, "--purge-config") {
		_ = os.Remove(path)
		fmt.Println("removed", path)
	}
	fmt.Println("restore OLLAMA_HOST to 127.0.0.1:11434 manually or from backup")
	return nil
}

func syncUpstreamsToDaemon(cfg config.Config) error {
	payload := map[string]any{"upstreams": cfg.Upstreams}
	b, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, "http://"+cfg.AdminAddress+"/admin/upstreams", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	setAdminAuth(req, cfg)
	resp, err := (&http.Client{Timeout: 2 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("daemon sync failed: %s", resp.Status)
	}
	return nil
}

func setAdminAuth(req *http.Request, cfg config.Config) {
	if !cfg.Security.AdminTokenRequired {
		return
	}
	token := strings.TrimSpace(cfg.Security.AdminToken)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("OSB_ADMIN_TOKEN"))
	}
	if token != "" {
		req.Header.Set("X-OSB-Admin-Token", token)
	}
}

func reportDaemonSync(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "daemon sync skipped:", err)
	}
}

func execDetached(bin string, args ...string) error {
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd.Start()
}
