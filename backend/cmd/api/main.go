// Command api runs the SMS platform HTTP API (Phase 3 walking skeleton).
//
// It serves /healthz and /readyz on the application listener and Prometheus metrics on a
// separate listener that binds to the loopback interface by default. It shuts down gracefully
// on SIGINT or SIGTERM.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/httpserver"
	"github.com/hruizjr/sms/backend/internal/observability"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "api:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		return err
	}

	logger := observability.NewLogger(os.Stdout, cfg.Log)
	metrics := observability.NewMetrics()
	readiness := &httpserver.Readiness{}

	router := httpserver.NewRouter(httpserver.RouterDeps{
		Logger:    logger,
		Metrics:   metrics,
		Readiness: readiness,
	})
	server := httpserver.NewServer(httpserver.ServerConfig{
		Addr:              cfg.HTTP.Addr,
		MetricsAddr:       cfg.Metrics.Addr,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
		ShutdownTimeout:   cfg.ShutdownTimeout,
	}, logger, router, metrics.Handler(), readiness)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Log only an explicit, non-sensitive summary of the configuration — never the environment.
	logger.Info("starting api",
		"env", string(cfg.Env),
		"log_level", cfg.Log.Level.String(),
		"log_format", string(cfg.Log.Format),
	)
	return server.Run(ctx)
}
