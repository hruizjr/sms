//go:build integration

package database_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/dbtest"
)

const testTimeout = 30 * time.Second

func open(t *testing.T, cfg config.DatabaseConfig) *database.DB {
	t.Helper()
	db, err := database.Open(context.Background(), cfg, "sms-integration-test")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	t.Cleanup(db.Close)
	return db
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)
	return ctx
}

func TestIntegrationOpenAndPing(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	cfg := dbtest.NewDatabase(t)
	db := open(t, cfg)

	if err := db.Ping(ctx); err != nil {
		t.Fatalf("Ping() error: %v", err)
	}

	var name, app string
	var versionNum int
	err := db.Pool().QueryRow(ctx,
		"SELECT current_database(), current_setting('application_name'), current_setting('server_version_num')::int",
	).Scan(&name, &app, &versionNum)
	if err != nil {
		t.Fatalf("query session settings: %v", err)
	}
	if name != cfg.Name {
		t.Errorf("connected to database %q, want the isolated test database %q", name, cfg.Name)
	}
	if app != "sms-integration-test" {
		t.Errorf("application_name = %q, want sms-integration-test", app)
	}
	if versionNum < 180000 || versionNum >= 190000 {
		t.Errorf("server_version_num = %d, want PostgreSQL 18.x (ADR-0005)", versionNum)
	}
}

func TestIntegrationWrongPasswordFailsWithoutLeakingIt(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	cfg := dbtest.ServerConfig(t)
	realPassword := cfg.Password.Reveal()
	const wrong = "synthetic-wrong-password-for-test"
	cfg.Password = config.Secret(wrong)

	err := open(t, cfg).Ping(ctx)
	if err == nil {
		t.Fatal("Ping() with a wrong password succeeded")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "28P01" {
		t.Errorf("error = %v, want SQLSTATE 28P01 (invalid_password)", err)
	}
	for _, secret := range []string{wrong, realPassword} {
		if strings.Contains(err.Error(), secret) {
			t.Error("connection error reveals a password")
		}
	}
}

func TestIntegrationMissingDatabaseFails(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	cfg := dbtest.ServerConfig(t)
	cfg.Name = dbtest.NewName(t)

	err := open(t, cfg).Ping(ctx)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "3D000" {
		t.Errorf("Ping() error = %v, want SQLSTATE 3D000 (invalid_catalog_name)", err)
	}
}

func TestIntegrationPoolLifecycle(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	cfg := dbtest.NewDatabase(t)
	cfg.MaxConns = 3
	db, err := database.Open(ctx, cfg, "sms-pool-lifecycle")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}

	// The pool never opens more than MaxConns connections.
	conns := make([]*pgxpool.Conn, 0, cfg.MaxConns)
	for range cfg.MaxConns {
		c, err := db.Pool().Acquire(ctx)
		if err != nil {
			t.Fatalf("Acquire() error: %v", err)
		}
		conns = append(conns, c)
	}
	waitCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	if extra, err := db.Pool().Acquire(waitCtx); err == nil {
		extra.Release()
		t.Error("acquired more connections than MaxConns")
	}
	cancel()
	if got := connectionsNamed(t, ctx, cfg, "sms-pool-lifecycle"); got != cfg.MaxConns {
		t.Errorf("server sees %d pool connections, want %d", got, cfg.MaxConns)
	}

	for _, c := range conns {
		c.Release()
	}
	db.Close()

	if err := db.Ping(ctx); err == nil {
		t.Error("Ping() succeeded after Close()")
	}
	// Close terminates every connection the pool opened.
	deadline := time.Now().Add(5 * time.Second)
	for connectionsNamed(t, ctx, cfg, "sms-pool-lifecycle") != 0 {
		if time.Now().After(deadline) {
			t.Fatal("connections remain open after Close()")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// connectionsNamed counts server connections to cfg.Name with the given application_name, using a
// separate connection.
func connectionsNamed(t *testing.T, ctx context.Context, cfg config.DatabaseConfig, app string) int {
	t.Helper()
	observer := open(t, cfg)
	var n int
	err := observer.Pool().QueryRow(ctx,
		"SELECT count(*) FROM pg_stat_activity WHERE datname = $1 AND application_name = $2",
		cfg.Name, app).Scan(&n)
	if err != nil {
		t.Fatalf("count connections: %v", err)
	}
	observer.Close()
	return n
}

func TestIntegrationInTx(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	db := open(t, dbtest.NewDatabase(t))

	// A test-only table in the throwaway database; not a migration and not a business table.
	if _, err := db.Pool().Exec(ctx, "CREATE TABLE tx_probes (id int PRIMARY KEY)"); err != nil {
		t.Fatalf("create probe table: %v", err)
	}
	insert := func(id int) func(pgx.Tx) error {
		return func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, "INSERT INTO tx_probes (id) VALUES ($1)", id)
			return err
		}
	}
	exists := func(id int) bool {
		var ok bool
		if err := db.Pool().QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM tx_probes WHERE id = $1)", id).Scan(&ok); err != nil {
			t.Fatalf("query probe: %v", err)
		}
		return ok
	}

	t.Run("commit", func(t *testing.T) {
		if err := db.InTx(ctx, insert(1)); err != nil {
			t.Fatalf("InTx() error: %v", err)
		}
		if !exists(1) {
			t.Error("committed row is missing")
		}
	})

	t.Run("rollback on error", func(t *testing.T) {
		errAbort := errors.New("synthetic failure")
		err := db.InTx(ctx, func(tx pgx.Tx) error {
			if err := insert(2)(tx); err != nil {
				return err
			}
			return errAbort
		})
		if !errors.Is(err, errAbort) {
			t.Errorf("InTx() error = %v, want the function's error", err)
		}
		if exists(2) {
			t.Error("row from a failed transaction was committed")
		}
	})

	t.Run("rollback on panic", func(t *testing.T) {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("InTx() swallowed the panic")
				}
			}()
			_ = db.InTx(ctx, func(tx pgx.Tx) error {
				if err := insert(3)(tx); err != nil {
					return err
				}
				panic("synthetic panic")
			})
		}()
		if exists(3) {
			t.Error("row from a panicking transaction was committed")
		}
	})

	t.Run("rollback on constraint violation", func(t *testing.T) {
		err := db.InTx(ctx, func(tx pgx.Tx) error {
			if err := insert(4)(tx); err != nil {
				return err
			}
			return insert(1)(tx) // duplicate primary key
		})
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
			t.Errorf("InTx() error = %v, want SQLSTATE 23505 (unique_violation)", err)
		}
		if exists(4) {
			t.Error("row from a failed transaction was committed")
		}
	})
}

func TestIntegrationConcurrentUse(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	cfg := dbtest.NewDatabase(t)
	cfg.MaxConns = 4
	db := open(t, cfg)

	// More concurrent queries than connections: callers wait for a free connection.
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for range 20 {
		wg.Go(func() {
			var one int
			errs <- db.Pool().QueryRow(ctx, "SELECT 1 FROM pg_sleep(0.02)").Scan(&one)
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("concurrent query error: %v", err)
		}
	}
	if got := db.Pool().Stat().MaxConns(); got != 4 {
		t.Errorf("pool MaxConns = %d, want 4", got)
	}
}
