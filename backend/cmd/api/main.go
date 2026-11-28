// Command api runs the SMS platform HTTP API.
//
// It serves the public API v1 (POST /v1/messages, GET /v1/messages, GET /v1/messages/{message_id};
// Bearer API keys; Phase 4 Group 2), /healthz and /readyz on the application listener, and
// Prometheus metrics on a separate, loopback-only listener. Accepted messages are handed to the
// provider by the dispatcher command, not by the API. /readyz checks PostgreSQL connectivity. The API never runs
// database migrations; use the migrate command. It connects as the application role (sms_app) and
// refuses to start, or to use any connection, if that role is unsafe (database.RequireApplicationRole). It shuts down gracefully on SIGINT or SIGTERM,
// closing the database pool after in-flight requests have finished.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/hruizjr/sms/backend/internal/apikey"
	"github.com/hruizjr/sms/backend/internal/audit"
	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/httpserver"
	"github.com/hruizjr/sms/backend/internal/messaging"
	"github.com/hruizjr/sms/backend/internal/messaging/provider"
	"github.com/hruizjr/sms/backend/internal/messaging/providers"
	"github.com/hruizjr/sms/backend/internal/observability"
	"github.com/hruizjr/sms/backend/internal/portal"
	"github.com/hruizjr/sms/backend/internal/signinlimit"
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
	// In production TRUSTED_PROXIES must be set explicitly (addresses, or "none").
	if err := cfg.ValidateAPI(); err != nil {
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
		"sms_provider", cfg.Provider.Name,
		"trusted_proxies", len(cfg.HTTP.Proxies.Prefixes),
		"trusted_proxies_set", cfg.HTTP.Proxies.Set,
		"signin_failure_limit_ipv4", cfg.SignInLimit.FailuresPerIPv4,
		"signin_failure_limit_ipv6_64", cfg.SignInLimit.FailuresPerIPv6Subnet,
		"signin_failure_limit_ipv6_48", cfg.SignInLimit.FailuresPerIPv6Aggregate,
		"signin_failure_window", cfg.SignInLimit.FailureWindow.String(),
		"signin_request_rate", cfg.SignInLimit.RequestsPerSecond,
		"signin_request_burst", cfg.SignInLimit.RequestBurst,
		"password_hash_concurrency", cfg.PasswordHash.Concurrency,
		"password_hash_queue", cfg.PasswordHash.Queue,
		"password_hash_max_wait", cfg.PasswordHash.MaxWait.String(),
	)
	if len(cfg.HTTP.Proxies.Broad) > 0 {
		broad := make([]string, len(cfg.HTTP.Proxies.Broad))
		for i, p := range cfg.HTTP.Proxies.Broad {
			broad[i] = p.String()
		}
		logger.Warn("SECURITY WARNING: TRUSTED_PROXIES contains ranges broader than an IPv4 /8 or an IPv6 /32 "+
			"(accepted because TRUSTED_PROXIES_ALLOW_BROAD_RANGES=true). Any client inside these ranges can "+
			"choose the client address the API records and rate-limits. List the proxies' actual addresses instead.",
			"broad_ranges", strings.Join(broad, ","))
	}
	if !cfg.HTTP.Proxies.Set {
		logger.Info("TRUSTED_PROXIES is not set: the direct peer address is the client address (allowed outside production)")
	}

	// The API only needs the configured provider's declared capabilities (for example a segment
	// limit); it never calls the provider. Without a provider every capability is unknown.
	var caps provider.Capabilities
	if cfg.Provider.Name != "" {
		p, err := providers.New(cfg.Provider)
		if err != nil {
			return err
		}
		caps = p.Capabilities()
	}

	// Every new connection is checked: the API never uses a superuser, a role with BYPASSRLS, a
	// table owner, or a role that can create objects or truncate tables (decision log §18).
	db, err := database.Open(ctx, cfg.Database, "sms-api", database.RequireApplicationRole())
	if err != nil {
		return err
	}
	// Closed after server.Run returns, i.e. after in-flight requests have finished.
	defer func() {
		db.Close()
		logger.Info("database pool closed")
	}()

	// A database outage at startup is not fatal: the API starts, /readyz reports 503, and the
	// pool connects once PostgreSQL is reachable. An unsafe database role is fatal: the API
	// refuses to start. (If the database is unreachable at startup, the same check runs on every
	// later connection, so an unsafe role can never serve a query; /readyz then stays 503.)
	pingCtx, cancel := context.WithTimeout(ctx, cfg.Database.ConnectTimeout)
	err = db.Ping(pingCtx)
	cancel()
	if errors.Is(err, database.ErrUnsafeRole) {
		return err
	}
	if err != nil {
		logger.Warn("database not reachable at startup; /readyz will report not ready", "error", err.Error())
	}

	metrics := observability.NewMetrics()
	readiness := &httpserver.Readiness{}

	// The public API (API keys) and the client portal (sessions) share one messaging engine, so a
	// portal message gets exactly the same idempotency, dispatch and status handling.
	// One API key service for both: the portal creates and revokes the keys the public API verifies.
	messages := messaging.NewService(db, caps)
	keys := apikey.NewService(db)
	portalAccounts, err := portal.NewService(db, portal.WithHashLimits(portal.HashLimits{
		Concurrency: cfg.PasswordHash.Concurrency, Queue: cfg.PasswordHash.Queue, MaxWait: cfg.PasswordHash.MaxWait,
	}))
	if err != nil {
		return err
	}
	// The security audit log and the per-source sign-in limit (Phase 4 Group 6) are stored in
	// PostgreSQL, so every API instance shares them and a restart resets nothing.
	auditLog := audit.NewService(db)
	signIns, err := signinlimit.New(db, signinlimit.Config{
		FailuresPerIPv4: cfg.SignInLimit.FailuresPerIPv4, FailuresPerIPv6Subnet: cfg.SignInLimit.FailuresPerIPv6Subnet,
		FailuresPerIPv6Aggregate: cfg.SignInLimit.FailuresPerIPv6Aggregate, FailureWindow: cfg.SignInLimit.FailureWindow,
		RequestsPerSecond: cfg.SignInLimit.RequestsPerSecond, RequestBurst: cfg.SignInLimit.RequestBurst,
	})
	if err != nil {
		return err
	}
	// Deletes ended rate-limit windows until shutdown; waited for before the pool is closed.
	cleanupCtx, stopCleanup := context.WithCancel(ctx)
	cleanupDone := make(chan struct{})
	go func() {
		defer close(cleanupDone)
		signIns.Run(cleanupCtx, logger)
	}()
	defer func() {
		stopCleanup()
		<-cleanupDone
	}()
	router := httpserver.NewRouter(httpserver.RouterDeps{
		Logger:         logger,
		Metrics:        metrics,
		Readiness:      readiness,
		Database:       db,
		TrustedProxies: cfg.HTTP.Proxies.Prefixes,
		PublicAPI: &httpserver.PublicAPIDeps{
			Messages: messages,
			Auth:     keys,
		},
		Portal: &httpserver.PortalDeps{
			Messages:      messages,
			Auth:          portalAccounts,
			APIKeys:       keys,
			Audit:         auditLog,
			SignInLimiter: signIns,
		},
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
