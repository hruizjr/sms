package httpserver

import (
	"net/http"
	"sync/atomic"
)

// Readiness reports whether the process is ready to accept traffic.
//
// In Group 1 the API has no external dependencies, so it is ready once both listeners are serving
// and becomes not ready as soon as graceful shutdown begins. Group 2 adds a database check to
// readiness; until then /readyz makes no claim about a database.
type Readiness struct {
	ready atomic.Bool
}

// SetReady sets the readiness state.
func (r *Readiness) SetReady(ready bool) { r.ready.Store(ready) }

// Ready reports the readiness state.
func (r *Readiness) Ready() bool { return r.ready.Load() }

type statusBody struct {
	Status string `json:"status"`
}

// healthz reports liveness: the process is running and serving HTTP. It checks no dependencies
// and reveals no configuration.
func healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, statusBody{Status: "ok"})
}

// readyz reports readiness. It returns 503 while starting up or shutting down.
func readyz(readiness *Readiness) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !readiness.Ready() {
			writeProblem(w, r, http.StatusServiceUnavailable, "The service is not ready to accept requests.")
			return
		}
		writeJSON(w, http.StatusOK, statusBody{Status: "ready"})
	}
}
