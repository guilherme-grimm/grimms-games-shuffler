// Command ggs is the GGS server: composition root only — config, wiring,
// serve, shutdown.
package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/guilherme-grimm/ggs/internal/adapter/sqlite"
	handler "github.com/guilherme-grimm/ggs/internal/handler/http"
	"github.com/guilherme-grimm/ggs/web"
)

type config struct {
	port          string
	dataDir       string
	baseURL       string
	steamAPIKey   string
	sessionSecret string
	anthropicKey  string
	aiModel       string
}

func loadConfig() config {
	return config{
		port:          envOr("PORT", "8080"),
		dataDir:       envOr("DATA_DIR", "/data"),
		baseURL:       os.Getenv("BASE_URL"),
		steamAPIKey:   os.Getenv("STEAM_API_KEY"),
		sessionSecret: os.Getenv("SESSION_SECRET"),
		anthropicKey:  os.Getenv("ANTHROPIC_API_KEY"),
		aiModel:       envOr("GGS_AI_MODEL", "claude-sonnet-5"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)
	cfg := loadConfig()

	// Auth needs these; the skeleton runs without them so a fresh deploy
	// comes up before secrets are configured.
	for _, v := range []struct{ name, val string }{
		{"STEAM_API_KEY", cfg.steamAPIKey},
		{"SESSION_SECRET", cfg.sessionSecret},
		{"BASE_URL", cfg.baseURL},
	} {
		if v.val == "" {
			log.Warn("env var not set; auth disabled until configured", "var", v.name)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := os.MkdirAll(cfg.dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	db, err := sqlite.Open(ctx, filepath.Join(cfg.dataDir, "ggs.db"))
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	dist, err := fs.Sub(web.DistFS, "dist")
	if err != nil {
		return fmt.Errorf("sub dist fs: %w", err)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.port,
		Handler:           handler.NewServer(log, db, dist).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	log.Info("ggs listening", "port", cfg.port)

	select {
	case err := <-errCh:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("shutdown: %w", err)
	}
	log.Info("ggs stopped")
	return nil
}
