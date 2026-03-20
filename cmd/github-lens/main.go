package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/b42labs/github-lens/internal/api"
	"github.com/b42labs/github-lens/internal/config"
	"github.com/b42labs/github-lens/internal/github"
	"github.com/b42labs/github-lens/internal/store"
	syncpkg "github.com/b42labs/github-lens/internal/sync"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

//go:embed all:frontend
var frontendFS embed.FS

func main() {
	var (
		configPath  string
		dbPath      string
		showVersion bool
		syncOnce    bool
	)

	flag.StringVar(&configPath, "config", "", "path to config.yaml")
	flag.StringVar(&dbPath, "db", "github-lens.db", "path to SQLite database")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.BoolVar(&syncOnce, "sync-once", false, "run a single sync and exit")
	flag.Parse()

	if showVersion {
		fmt.Printf("github-lens %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Open store
	s, err := store.New(dbPath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer func() { _ = s.Close() }()

	// Create GitHub client
	if cfg.GitHubToken == "" {
		slog.Warn("no GitHub token configured — unauthenticated API requests are limited to 60/hour")
	}
	client := github.NewClient(cfg.GitHubToken)

	// Create sync service
	syncSvc := syncpkg.NewService(cfg, client, s)

	// Sync-once mode
	if syncOnce {
		if err := syncSvc.SyncAndWait(context.Background()); err != nil {
			slog.Error("sync failed", "error", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Setup frontend FS
	frontend, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		slog.Error("failed to setup frontend filesystem", "error", err)
		os.Exit(1)
	}

	// Create handler and router
	handler := api.NewHandler(s, syncSvc, cfg, frontend)
	router := handler.Router()

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	// Start background sync
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	syncSvc.Start(ctx)

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		slog.Info("shutting down...")
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
	}()

	slog.Info("starting server", "port", cfg.Server.Port, "version", version)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
