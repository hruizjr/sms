// Package database owns the PostgreSQL connection pool shared by the api and migrate executables.
//
// It configures a pgxpool.Pool from config.DatabaseConfig, provides an explicit Ping for readiness
// checks, transaction helpers (InTx, and InTenantTx for row-level security), checks of the
// connected database role (roleguard.go), and Close for graceful shutdown. It contains no SQL for
// business data: queries are written as SQL and compiled with sqlc (ADR-0014) into the sqlcgen
// package. Opening the pool performs no network I/O; connections are established on first use, so
// the API can start, report not ready, and recover once PostgreSQL becomes reachable.
package database

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hruizjr/sms/backend/internal/config"
)

// healthCheckPeriod is how often the pool checks idle connections (pgxpool's default).
const healthCheckPeriod = time.Minute

// DB is a PostgreSQL connection pool. It is safe for concurrent use.
type DB struct {
	pool *pgxpool.Pool
}

// Option changes how Open configures the pool.
type Option func(*pgxpool.Config)

// Open creates a connection pool from cfg. applicationName is reported to PostgreSQL
// (pg_stat_activity.application_name) so that connections can be attributed to a process.
//
// Open does not connect; use Ping to check connectivity. Errors never contain the password.
func Open(ctx context.Context, cfg config.DatabaseConfig, applicationName string, opts ...Option) (*DB, error) {
	poolCfg, err := poolConfig(cfg, applicationName)
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt(poolCfg)
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	return &DB{pool: pool}, nil
}

// poolConfig translates cfg into a pgxpool configuration.
//
// The settings are passed to pgx as a keyword/value connection string so that pgx derives the TLS
// configuration for the SSL mode exactly as libpq does. The string exists only in memory and is
// never logged; a parse error is reported without it.
func poolConfig(cfg config.DatabaseConfig, applicationName string) (*pgxpool.Config, error) {
	settings := []struct{ key, value string }{
		{"host", cfg.Host},
		{"port", strconv.Itoa(cfg.Port)},
		{"dbname", cfg.Name},
		{"user", cfg.User},
		{"password", cfg.Password.Reveal()},
		{"sslmode", string(cfg.SSLMode)},
		{"application_name", applicationName},
	}
	var b strings.Builder
	for _, s := range settings {
		b.WriteString(s.key)
		b.WriteString("=")
		b.WriteString(quote(s.value))
		b.WriteString(" ")
	}

	poolCfg, err := pgxpool.ParseConfig(b.String())
	if err != nil {
		// pgx redacts the password from parse errors, but the message still echoes the connection
		// string; report a fixed message instead. Configuration validation makes this unlikely.
		return nil, errors.New("invalid database connection settings")
	}
	poolCfg.MaxConns = int32(cfg.MaxConns) //nolint:gosec // G115: bounded to 1..config.MaxDBConns by validation.
	poolCfg.MinConns = int32(cfg.MinConns) //nolint:gosec // G115: bounded to 0..MaxConns by validation.
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.HealthCheckPeriod = healthCheckPeriod
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout
	return poolCfg, nil
}

// quote returns value as a single-quoted libpq connection-string value, escaping backslashes and
// single quotes, so that any value (including a password with spaces or quotes) is passed intact.
func quote(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `'`, `\'`)
	return "'" + value + "'"
}

// Ping checks that a connection to PostgreSQL can be acquired and is alive. Callers bound it with a
// context deadline; the per-connection ConnectTimeout also applies.
func (db *DB) Ping(ctx context.Context) error {
	if err := db.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}

// InTx runs fn in a transaction. It commits if fn returns nil and rolls back if fn returns an error
// or panics (the panic is then re-raised). Keep transactions short and never call external
// services from fn (AGENTS.md §7.8).
func (db *DB) InTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, db.pool, fn)
}

// Pool returns the underlying pool, for sqlc-generated query code and for the migration runner.
func (db *DB) Pool() *pgxpool.Pool { return db.pool }

// Close closes all connections. It waits for acquired connections to be released, so call it only
// after the HTTP server has finished shutting down.
func (db *DB) Close() { db.pool.Close() }
