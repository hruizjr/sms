package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// UnmatchedRoute is the route label used when no route pattern matched (for example a 404).
const UnmatchedRoute = "unmatched"

// otherMethod is the method label used for any non-standard HTTP method, keeping cardinality bounded.
const otherMethod = "OTHER"

// Metrics owns a dedicated Prometheus registry and the HTTP server metrics.
//
// Labels are deliberately bounded: the HTTP method (normalised to a fixed set), the matched route
// pattern (never the raw URL), and the status code. Identifiers such as phone numbers, message IDs,
// tenant IDs, or API keys must never be used as labels.
type Metrics struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

// NewMetrics creates a registry with Go runtime, process, and HTTP request metrics.
func NewMetrics() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "sms",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total HTTP requests handled by the API, by method, matched route pattern, and status code.",
		}, []string{"method", "route", "code"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "sms",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds, by method and matched route pattern.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"method", "route"}),
	}
	m.registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.requests,
		m.duration,
	)
	return m
}

// ObserveHTTPRequest records one completed HTTP request.
func (m *Metrics) ObserveHTTPRequest(method, route string, status int, duration time.Duration) {
	method = normaliseMethod(method)
	if route == "" {
		route = UnmatchedRoute
	}
	m.requests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
	m.duration.WithLabelValues(method, route).Observe(duration.Seconds())
}

// Handler serves the registry in the Prometheus exposition format. It must only be mounted on
// the separate, non-public metrics listener.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Registry exposes the underlying registry for tests and future collectors.
func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

func normaliseMethod(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodOptions:
		return method
	default:
		return otherMethod
	}
}
