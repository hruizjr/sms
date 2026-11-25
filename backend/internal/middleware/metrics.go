package middleware

import (
	"net/http"
	"time"
)

// RequestObserver records completed HTTP requests. observability.Metrics satisfies it.
type RequestObserver interface {
	ObserveHTTPRequest(method, route string, status int, duration time.Duration)
}

// Metrics reports every completed request to obs, labelled by the matched route pattern rather
// than the raw URL so that metric cardinality stays bounded.
func Metrics(obs RequestObserver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := recordStatus(w)

			next.ServeHTTP(rec, r)

			obs.ObserveHTTPRequest(r.Method, RoutePattern(r), rec.Status(), time.Since(start))
		})
	}
}
