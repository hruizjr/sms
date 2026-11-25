// Command migrate applies, reports, and (outside production) rolls back database migrations.
//
// Migrations are embedded in the binary (backend/migrations) and run only when this command is
// invoked explicitly; the API never migrates on startup. It reads the same DB_* configuration as
// the API and never prints credentials.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/jackc/pgx/v5/stdlib"

	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/migration"
	"github.com/hruizjr/sms/backend/internal/observability"
	"github.com/hruizjr/sms/backend/migrations"
)

const usage = `Usage: migrate <command>

Commands:
  up        Apply all pending migrations.
  status    List every migration and whether it is applied.
  version   Print the version of the most recently applied migration (0 if none).
  down      Roll back the most recently applied migration.
            Refused when APP_ENV=production: production migrations are forward-only.

Configuration is read from the same environment variables as the API (DB_HOST, DB_PORT,
DB_NAME, DB_USER, DB_PASSWORD, DB_SSLMODE, ...). See backend/README.md.
`

// Exit codes.
const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.LookupEnv, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

// run executes one migrate command and returns the process exit code. Results go to stdout;
// structured logs and errors go to stderr.
func run(ctx context.Context, args []string, lookup config.LookupFunc, stdout, stderr io.Writer) int {
	// fail reports a problem on stderr. If stderr itself cannot be written, nothing more can be
	// reported, so that write error is deliberately ignored; the exit code still signals failure.
	fail := func(format string, a ...any) {
		_, _ = fmt.Fprintf(stderr, format, a...)
	}

	if len(args) != 1 {
		fail("%s", usage)
		return exitUsage
	}
	command := args[0]
	switch command {
	case "up", "status", "version", "down":
	case "help", "-h", "-help", "--help":
		if _, err := fmt.Fprint(stdout, usage); err != nil {
			return exitError
		}
		return exitOK
	default:
		fail("migrate: unknown command %q\n\n%s", command, usage)
		return exitUsage
	}

	cfg, err := config.Load(lookup)
	if err != nil {
		fail("migrate: %v\n", err)
		return exitError
	}
	if command == "down" && cfg.Env == config.EnvironmentProduction {
		fail("migrate: down is disabled when APP_ENV=production; production migrations are " +
			"forward-only (AGENTS.md §10.1). Fix a mistake with a new migration.\n")
		return exitError
	}

	logger := observability.NewLogger(stderr, cfg.Log, "migrate").With("command", command)
	// Log only explicit, non-sensitive settings — never the password or a connection string.
	logger.Info("migrate starting",
		"env", string(cfg.Env),
		"db_host", cfg.Database.Host,
		"db_port", cfg.Database.Port,
		"db_name", cfg.Database.Name,
	)

	out := &output{w: stdout}
	err = execute(ctx, command, cfg.Database, out)
	if err == nil {
		err = out.err
	}
	if err != nil {
		logger.Error("migrate failed", "error", err)
		fail("migrate: %v\n", err)
		return exitError
	}
	return exitOK
}

// output writes command results and remembers the first write error, so that a failed write (for
// example to a closed pipe) makes the command fail instead of silently losing its output.
type output struct {
	w   io.Writer
	err error
}

func (o *output) Write(p []byte) (int, error) {
	if o.err != nil {
		return 0, o.err
	}
	n, err := o.w.Write(p)
	o.err = err
	return n, err
}

func (o *output) println(a ...any) {
	_, _ = fmt.Fprintln(o, a...) // the error is kept in o.err and reported by run
}

func execute(ctx context.Context, command string, cfg config.DatabaseConfig, stdout *output) error {
	db, err := database.Open(ctx, cfg, "sms-migrate")
	if err != nil {
		return err
	}
	defer db.Close()

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	err = db.Ping(pingCtx)
	cancel()
	if err != nil {
		return err
	}

	sqlDB := stdlib.OpenDBFromPool(db.Pool())
	defer func() {
		// Closing the database/sql wrapper releases nothing the pool still needs; an error here
		// cannot affect the migration outcome, which has already been reported.
		_ = sqlDB.Close()
	}()

	runner, err := migration.New(sqlDB, migrations.FS)
	if err != nil {
		return err
	}

	switch command {
	case "up":
		return up(ctx, runner, stdout)
	case "status":
		return status(ctx, runner, stdout)
	case "version":
		v, err := runner.Version(ctx)
		if err != nil {
			return err
		}
		stdout.println(v)
		return nil
	case "down":
		return down(ctx, runner, stdout)
	}
	return fmt.Errorf("unknown command %q", command)
}

func up(ctx context.Context, runner *migration.Runner, stdout *output) error {
	results, err := runner.Up(ctx)
	for _, r := range results {
		stdout.println(r)
	}
	if err != nil {
		return err
	}
	if len(results) == 0 {
		stdout.println("no pending migrations")
	}
	v, err := runner.Version(ctx)
	if err != nil {
		return err
	}
	stdout.println("database version:", v)
	return nil
}

func down(ctx context.Context, runner *migration.Runner, stdout *output) error {
	result, err := runner.Down(ctx)
	if errors.Is(err, migration.ErrNothingToRollBack) {
		return err
	}
	if result != nil {
		stdout.println(result)
	}
	if err != nil {
		return err
	}
	v, err := runner.Version(ctx)
	if err != nil {
		return err
	}
	stdout.println("database version:", v)
	return nil
}

func status(ctx context.Context, runner *migration.Runner, stdout *output) error {
	statuses, err := runner.Status(ctx)
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "VERSION\tSTATE\tAPPLIED AT (UTC)\tFILE") // buffered; errors surface from Flush
	for _, s := range statuses {
		appliedAt := "-"
		if !s.AppliedAt.IsZero() {
			appliedAt = s.AppliedAt.UTC().Format(time.RFC3339)
		}
		_, _ = fmt.Fprintf(tw, "%d\t%s\t%s\t%s\n", s.Source.Version, s.State, appliedAt, s.Source.Path)
	}
	return tw.Flush()
}
