// Package httpserver wires the HTTP router, the health endpoints, and the process's HTTP
// listeners (the public application listener and the separate, non-public metrics listener).
package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/hruizjr/sms/backend/internal/middleware"
)

// problem is an RFC 9457 Problem Details document.
//
// The walking skeleton uses "about:blank" as the problem type, for which RFC 9457 specifies that
// the title is the HTTP status phrase. Application-specific problem type URIs are defined with the
// OpenAPI contract in Phase 3 Group 3.
type problem struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// writeProblem writes a problem document. detail must be a fixed, client-safe message: never an
// error string, stack trace, SQL, or any internal detail.
func writeProblem(w http.ResponseWriter, r *http.Request, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	// The status line is already sent; an encoding or write failure here (for example a client
	// that disconnected) cannot be reported to the client, and the access log records the status.
	_ = json.NewEncoder(w).Encode(problem{
		Type:      "about:blank",
		Title:     http.StatusText(status),
		Status:    status,
		Detail:    detail,
		RequestID: middleware.RequestIDFromContext(r.Context()),
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body) // see writeProblem for why the error is not actionable
}

func notFound(w http.ResponseWriter, r *http.Request) {
	writeProblem(w, r, http.StatusNotFound, "The requested resource does not exist.")
}

func methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	writeProblem(w, r, http.StatusMethodNotAllowed, "The method is not allowed for this resource.")
}

func internalError(w http.ResponseWriter, r *http.Request) {
	writeProblem(w, r, http.StatusInternalServerError, "An unexpected error occurred.")
}
