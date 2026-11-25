// Command api runs the SMS platform HTTP API (Phase 3 walking skeleton).
//
// It serves /healthz and /readyz on the application listener and Prometheus metrics on a
// separate, loopback-only listener. /readyz checks PostgreSQL connectivity. The API never runs
// database migrations; use the migrate command. It shuts down gracefully on SIGINT or SIGTERM,
// closing the database pool after in-flight requests have finished.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/database"
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

	logger := observability.NewLogger(os.Stdout, cfg.Log, "api")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Log only an explicit, non-sensitive summary of the configuration — never the environment,
	// the database password, or a connection string.
	logger.Info("starting api",
		"env", string(cfg.Env),
		"log_level", cfg.Log.Level.String(),
		"log_format", string(cfg.Log.Format),
		"db_host", cfg.Database.Host,
		"db_port", cfg.Database.Port,
		"db_name", cfg.Database.Name,
		"db_sslmode", string(cfg.Database.SSLMode),
		"db_max_conns", cfg.Database.MaxConns,
	)

	db, err := database.Open(ctx, cfg.Database, "sms-api")
	if err != nil {
		return err
	}
	// Closed after server.Run returns, i.e. after in-flight requests have finished.
	defer func() {
		db.Close()
		logger.Info("database pool closed")
	}()

	// A database outage at startup is not fatal: the API starts, /readyz reports 503, and the
	// pool connects once PostgreSQL is reachable.
	pingCtx, cancel := context.WithTimeout(ctx, cfg.Database.ConnectTimeout)
	if err := db.Ping(pingCtx); err != nil {
		logger.Warn("database not reachable at startup; /readyz will report not ready", "error", err.Error())
	}
	cancel()

	metrics := observability.NewMetrics()
	readiness := &httpserver.Readiness{}

	router := httpserver.NewRouter(httpserver.RouterDeps{
		Logger:    logger,
		Metrics:   metrics,
		Readiness: readiness,
		Database:  db,
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

	return server.Run(ctx)
}
