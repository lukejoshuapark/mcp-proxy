package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/lukejoshuapark/mcp-proxy/config"
	"github.com/lukejoshuapark/mcp-proxy/handler"
)

func main() {
	if os.Getenv("MCP_PROXY_PRETTY") != "" {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	} else {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	}

	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
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

	slog.Info("listening", "addr", cfg.ListenAddr)
	return httpServer.ListenAndServe()
}
