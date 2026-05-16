package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/config"
	"github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/invoke"
	"github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/server"
	"github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/version"
	"github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/workflow"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	log := setupLogger(cfg.LogLevel)

	log.Info("invoke-openai-proxy starting",
		"version", version.Version,
		"commit", version.Commit,
		"listen", cfg.Addr(),
		"invoke_url", cfg.InvokeURL,
		"data_dir", cfg.DataDir,
	)

	// Ensure data dir exists
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Error("create data dir", "error", err)
		os.Exit(1)
	}

	// Initialize registry
	registry, err := workflow.NewRegistry(cfg.DataDir)
	if err != nil {
		log.Error("load registry", "error", err)
		os.Exit(1)
	}
	log.Info("registry loaded", "models", len(registry.List()))

	// Initialize InvokeAI client
	invokeClient := invoke.NewClient(cfg.InvokeURL, cfg.Timeout, log)

	srv := server.New(cfg, log, invokeClient, registry)

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", "error", err)
	}
}

func setupLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))
}
