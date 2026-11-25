package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// unmatchedRoute is logged when the router matched no pattern (for example a 404).
const unmatchedRoute = "unmatched"

// AccessLog writes one structured log entry per request after it completes.
//
// It records only the request ID, method, matched route pattern, status, and duration. It never
// records the raw URL or query string, headers (including Authorization), request or response
// bodies, or client IP addresses, because any of these can carry secrets or personal data.
// 5xx responses are logged at ERROR level; everything else at INFO.
func AccessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := recordStatus(w)

			next.ServeHTTP(rec, r)

			status := rec.Status()
			level := slog.LevelInfo
			if status >= http.StatusInternalServerError {
				level = slog.LevelError
			}
			logger.LogAttrs(r.Context(), level, "http request",
				slog.String("request_id", RequestIDFromContext(r.Context())),
				slog.String("method", r.Method),
				slog.String("route", RoutePattern(r)),
				slog.Int("status", status),
				slog.Float64("duration_ms", float64(time.Since(start).Microseconds())/1000),
			)
		})
	}
}

// RoutePattern returns the chi route pattern that matched the request, or "unmatched".
// It is only meaningful after the router has handled the request.
func RoutePattern(r *http.Request) string {
	if rc := chi.RouteContext(r.Context()); rc != nil {
		if pattern := rc.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	return unmatchedRoute
}
