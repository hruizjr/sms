package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

// DefaultReadinessTimeout bounds the database check made by each /readyz request.
const DefaultReadinessTimeout = 2 * time.Second

// Readiness records whether the process's listeners are serving. It is set by the server: ready
// once both listeners are serving, and not ready as soon as graceful shutdown begins.
type Readiness struct {
	ready atomic.Bool
}

// SetReady sets the readiness state.
func (r *Readiness) SetReady(ready bool) { r.ready.Store(ready) }

// Ready reports the readiness state.
func (r *Readiness) Ready() bool { return r.ready.Load() }

// Pinger checks a dependency's connectivity. *database.DB satisfies it.
type Pinger interface {
	Ping(ctx context.Context) error
}

type statusBody struct {
	Status string `json:"status"`
}

// notReadyDetail is the only explanation given to clients. Why the service is not ready (which
// dependency failed and how) is logged server-side and never returned.
const notReadyDetail = "The service is not ready to accept requests."

// healthz reports liveness: the process is running and serving HTTP. It checks no dependencies
// (not even the database) and reveals no configuration, so a database outage never causes the
// process to be restarted.
func healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, statusBody{Status: "ok"})
}

// readinessCheck implements /readyz: the listeners must be serving and the database must answer a
// ping within the timeout.
type readinessCheck struct {
	readiness *Readiness
	database  Pinger
	timeout   time.Duration
	logger    *slog.Logger
	// dbFailing remembers the last database result so that failures and recoveries are logged
	// once per transition instead of on every probe.
	dbFailing atomic.Bool
}

func (c *readinessCheck) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !c.readiness.Ready() {
		writeProblem(w, r, http.StatusServiceUnavailable, notReadyDetail)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), c.timeout)
	err := c.database.Ping(ctx)
	cancel()

	if err != nil {
		if !c.dbFailing.Swap(true) {
			c.logger.LogAttrs(r.Context(), slog.LevelWarn, "readiness check failed",
				slog.String("dependency", "database"),
				slog.String("error", err.Error()),
			)
		}
		writeProblem(w, r, http.StatusServiceUnavailable, notReadyDetail)
		return
	}
	if c.dbFailing.Swap(false) {
		c.logger.LogAttrs(r.Context(), slog.LevelInfo, "readiness check recovered",
			slog.String("dependency", "database"),
		)
	}
	writeJSON(w, http.StatusOK, statusBody{Status: "ready"})
}
