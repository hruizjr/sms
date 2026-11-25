// Package middleware contains the HTTP middleware shared by the API: request IDs, structured
// access logging, metrics, panic recovery, security headers, and request body limits.
package middleware

import "net/http"

// statusRecorder wraps a ResponseWriter to remember the status code that was sent.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

// recordStatus wraps w, reusing an existing recorder so that stacked middleware share one wrapper.
func recordStatus(w http.ResponseWriter) *statusRecorder {
	if rec, ok := w.(*statusRecorder); ok {
		return rec
	}
	return &statusRecorder{ResponseWriter: w}
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.status = code
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}

// Status returns the status code sent, or 200 when the handler wrote nothing (net/http's default).
func (r *statusRecorder) Status() int {
	if !r.wroteHeader {
		return http.StatusOK
	}
	return r.status
}

// Unwrap lets http.ResponseController reach the underlying writer (for Flush, deadlines, etc.).
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
