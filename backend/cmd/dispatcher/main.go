// Command dispatcher runs the messaging engine's background work (Phase 4 Group 2): it hands
// accepted messages to the configured provider, records every submission attempt and its
// classified outcome, receives and applies delivery receipts, and moves overdue messages to
// unknown. Several dispatchers may run against one database.
//
// It connects as the application role (sms_app) and refuses an unsafe role, like the API. It
// never resubmits an uncertain submission. With the simulator provider (the only one available,
// decision log §15) nothing leaves the process and no SMS is sent. It stops gracefully on SIGINT
// or SIGTERM: in-flight submissions finish, then the database pool is closed.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/messaging"
	"github.com/hruizjr/sms/backend/internal/messaging/providers"
	"github.com/hruizjr/sms/backend/internal/observability"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "dispatcher:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		return err
	}
	logger := observability.NewLogger(os.Stdout, cfg.Log, "dispatcher")

	p, err := providers.New(cfg.Provider)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	caps := p.Capabilities()
	logger.Info("starting dispatcher",
		"env", string(cfg.Env),
		"db_host", cfg.Database.Host,
		"db_name", cfg.Database.Name,
		"provider", p.ID(),
		"provider_capabilities_basis", caps.Basis,
		"poll_interval", cfg.Dispatch.PollInterval.String(),
		"concurrency", cfg.Dispatch.Concurrency,
		"submit_timeout", cfg.Dispatch.SubmitTimeout.String(),
		"lease", cfg.Dispatch.Lease.String(),
		"max_attempts", cfg.Dispatch.MaxAttempts,
		"report_window", cfg.Provider.ReportWindow().String(),
	)

	db, err := database.Open(ctx, cfg.Database, "sms-dispatcher", database.RequireApplicationRole())
	if err != nil {
		return err
	}
	defer func() {
		db.Close()
		logger.Info("database pool closed")
	}()
	pingCtx, cancel := context.WithTimeout(ctx, cfg.Database.ConnectTimeout)
	err = db.Ping(pingCtx)
	cancel()
	if errors.Is(err, database.ErrUnsafeRole) {
		return err
	}
	if err != nil {
		logger.Warn("database not reachable at startup; retrying every cycle", "error", err.Error())
	}

	d, err := messaging.NewDispatcher(db, p, messaging.DispatcherConfig{
		PollInterval:   cfg.Dispatch.PollInterval,
		BatchSize:      cfg.Dispatch.BatchSize,
		Concurrency:    cfg.Dispatch.Concurrency,
		SubmitTimeout:  cfg.Dispatch.SubmitTimeout,
		Lease:          cfg.Dispatch.Lease,
		MaxAttempts:    cfg.Dispatch.MaxAttempts,
		RetryBaseDelay: cfg.Dispatch.RetryBaseDelay,
		RetryMaxDelay:  cfg.Dispatch.RetryMaxDelay,
		MaxMessageAge:  cfg.Dispatch.MaxMessageAge,
		ReportWindow:   cfg.Provider.ReportWindow(),
	}, logger)
	if err != nil {
		return err
	}
	err = d.Run(ctx)
	logger.Info("dispatcher stopped")
	return err
}
