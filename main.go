package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lukejoshuapark/mcp-proxy/config"
	"github.com/lukejoshuapark/mcp-proxy/handler"
)

func main() {
	var logHandler slog.Handler
	if os.Getenv("MCP_PROXY_PRETTY") != "" {
		logHandler = slog.NewTextHandler(os.Stderr, nil)
	} else {
		logHandler = slog.NewJSONHandler(os.Stderr, nil)
	}
	slog.SetDefault(slog.New(logHandler))

	if err := run(logHandler); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(logHandler slog.Handler) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	if cfg.AppInsightsConnectionString != "" {
		aiHandler := newAppInsightsHandler(cfg.AppInsightsConnectionString, logHandler)
		defer aiHandler.Close()
		slog.SetDefault(slog.New(aiHandler))
		slog.Info("application insights enabled")
	}

	sessions, codes, err := initializeStores(cfg)
	if err != nil {
		return err
	}

	srv, err := handler.NewServer(cfg, sessions, codes)
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}

	httpServer := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      srv.Routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen error", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("listening", "addr", cfg.ListenAddr)
	<-ctx.Done()
	slog.Info("shutting down gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	slog.Info("shutdown complete")
	return nil
}
