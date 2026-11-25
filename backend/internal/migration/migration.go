// Package migration runs versioned SQL migrations with goose (ADR-0014).
//
// Migrations run only through the migrate executable, never during API startup. Each SQL migration
// runs in its own transaction unless the file opts out with "-- +goose NO TRANSACTION" (needed, for
// example, for CREATE INDEX CONCURRENTLY). A PostgreSQL session-level advisory lock ensures that two
// migrate processes never apply migrations concurrently. Out-of-order migrations are rejected.
package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// ErrNothingToRollBack is returned by Down when no migration has been applied.
var ErrNothingToRollBack = errors.New("no applied migration to roll back")

// Runner applies, rolls back, and reports migrations from a file system of goose SQL files.
type Runner struct {
	provider *goose.Provider
}

// New returns a Runner for the migrations in fsys, executed through db.
//
// Only the SQL files in fsys are used: Go migrations registered globally are ignored, so the set
// of migrations is exactly what the file system contains.
func New(db *sql.DB, fsys fs.FS) (*Runner, error) {
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, fmt.Errorf("create migration lock: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db, fsys,
		goose.WithSessionLocker(locker),
		goose.WithDisableGlobalRegistry(true),
		goose.WithAllowOutofOrder(false),
	)
	if err != nil {
		return nil, fmt.Errorf("load migrations: %w", err)
	}
	return &Runner{provider: provider}, nil
}

// Up applies every pending migration in version order and returns the results. If a migration
// fails, the error is a *goose.PartialError listing what was applied before the failure; the failed
// migration's transaction is rolled back.
func (r *Runner) Up(ctx context.Context) ([]*goose.MigrationResult, error) {
	results, err := r.provider.Up(ctx)
	if err != nil {
		return results, fmt.Errorf("apply migrations: %w", err)
	}
	return results, nil
}

// Down rolls back the most recently applied migration.
func (r *Runner) Down(ctx context.Context) (*goose.MigrationResult, error) {
	result, err := r.provider.Down(ctx)
	if errors.Is(err, goose.ErrNoNextVersion) {
		return nil, ErrNothingToRollBack
	}
	if err != nil {
		return result, fmt.Errorf("roll back migration: %w", err)
	}
	return result, nil
}

// Status reports every known migration and whether it is applied, in version order.
func (r *Runner) Status(ctx context.Context) ([]*goose.MigrationStatus, error) {
	statuses, err := r.provider.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("read migration status: %w", err)
	}
	return statuses, nil
}

// Version returns the version of the most recently applied migration, or 0 if none is applied.
func (r *Runner) Version(ctx context.Context) (int64, error) {
	v, err := r.provider.GetDBVersion(ctx)
	if err != nil {
		return 0, fmt.Errorf("read migration version: %w", err)
	}
	return v, nil
}
