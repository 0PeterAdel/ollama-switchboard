package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ollama-switchboard/ollama-switchboard/internal/admin"
	"github.com/ollama-switchboard/ollama-switchboard/internal/config"
	"github.com/ollama-switchboard/ollama-switchboard/internal/health"
	"github.com/ollama-switchboard/ollama-switchboard/internal/proxy"
	"github.com/ollama-switchboard/ollama-switchboard/internal/ui"
	"github.com/ollama-switchboard/ollama-switchboard/internal/upstream"
)

type Runtime struct {
	cfg     config.Config
	logger  *slog.Logger
	tracker *health.Tracker
	mgr     *upstream.Manager
}

func New(cfg config.Config, logger *slog.Logger) *Runtime {
	return &Runtime{cfg: cfg, logger: logger, tracker: health.NewTracker(), mgr: upstream.NewManager(cfg)}
}

func (r *Runtime) Run(ctx context.Context) error {
	p := proxy.New(r.cfg, r.logger, r.mgr, r.tracker, proxy.LoadSecrets())
	adminAPI := (&admin.API{Cfg: r.cfg, Tracker: r.tracker, Mgr: r.mgr}).Handler()

	proxySrv := &http.Server{Addr: r.cfg.ListenAddress, Handler: p.Handler()}
	adminMux := http.NewServeMux()
	adminMux.Handle("/", adminAPI)
	adminSrv := &http.Server{Addr: r.cfg.AdminAddress, Handler: adminMux}

	uiSrv := &http.Server{Addr: r.cfg.UIAddress, Handler: ui.Handler("http://" + r.cfg.ListenAddress)}

	errCh := make(chan error, 3)
	go func() { errCh <- proxySrv.ListenAndServe() }()
	go func() { errCh <- adminSrv.ListenAndServe() }()
	go func() { errCh <- uiSrv.ListenAndServe() }()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ctx.Done():
	case <-sigCh:
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server failed: %w", err)
		}
	}
	_ = proxySrv.Shutdown(context.Background())
	_ = adminSrv.Shutdown(context.Background())
	_ = uiSrv.Shutdown(context.Background())
	return nil
}
