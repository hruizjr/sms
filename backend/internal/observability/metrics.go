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

	// Sign-in limit cleanup (internal/signinlimit): one observation per cleanup cycle.
	cleanupRuns        *prometheus.CounterVec
	cleanupIncomplete  prometheus.Counter
	cleanupDuration    prometheus.Histogram
	cleanupDeleted     *prometheus.CounterVec
	cleanupLastSuccess prometheus.Gauge
}

// Label values of the sign-in limit cleanup metrics (fixed; no source address, user or tenant).
const (
	cleanupResultSuccess = "success"
	cleanupResultFailure = "failure"
	cleanupTableBudgets  = "failure_budgets"
	cleanupTableBuckets  = "request_buckets"
)

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
	m.cleanupRuns = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "sms",
		Subsystem: "signin_limit",
		Name:      "cleanup_runs_total",
		Help:      "Sign-in limit cleanup cycles, by result (success or failure); their sum is the number of attempts.",
	}, []string{"result"})
	m.cleanupIncomplete = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "sms",
		Subsystem: "signin_limit",
		Name:      "cleanup_incomplete_runs_total",
		Help:      "Sign-in limit cleanup cycles that stopped at their work bound or failed with rows possibly left to delete.",
	})
	m.cleanupDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "sms",
		Subsystem: "signin_limit",
		Name:      "cleanup_duration_seconds",
		Help:      "Duration of sign-in limit cleanup cycles in seconds, successful or not.",
		Buckets:   []float64{0.005, 0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
	})
	m.cleanupDeleted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "sms",
		Subsystem: "signin_limit",
		Name:      "cleanup_deleted_rows_total",
		Help:      "Rows deleted by sign-in limit cleanup, by table (failure_budgets or request_buckets).",
	}, []string{"table"})
	m.cleanupLastSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "sms",
		Subsystem: "signin_limit",
		Name:      "cleanup_last_success_timestamp_seconds",
		Help:      "Unix time of the last successful sign-in limit cleanup cycle of this process; 0 until one succeeds.",
	})
	// Every series exists from the start, so a process whose cleanup never succeeds shows 0 successes
	// and a last-success time of 0 rather than no data.
	m.cleanupRuns.WithLabelValues(cleanupResultSuccess)
	m.cleanupRuns.WithLabelValues(cleanupResultFailure)
	m.cleanupDeleted.WithLabelValues(cleanupTableBudgets)
	m.cleanupDeleted.WithLabelValues(cleanupTableBuckets)
	m.registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.requests,
		m.duration,
		m.cleanupRuns,
		m.cleanupIncomplete,
		m.cleanupDuration,
		m.cleanupDeleted,
		m.cleanupLastSuccess,
	)
	return m
}

// ObserveSignInLimitCleanup records one sign-in limit cleanup cycle (signinlimit.CleanupMetrics).
// A failed cycle never updates the last-success time.
func (m *Metrics) ObserveSignInLimitCleanup(succeeded, complete bool, duration time.Duration, failureBudgetsDeleted, requestBucketsDeleted int64) {
	result := cleanupResultFailure
	if succeeded {
		result = cleanupResultSuccess
		m.cleanupLastSuccess.SetToCurrentTime()
	}
	m.cleanupRuns.WithLabelValues(result).Inc()
	if !complete {
		m.cleanupIncomplete.Inc()
	}
	m.cleanupDuration.Observe(duration.Seconds())
	m.cleanupDeleted.WithLabelValues(cleanupTableBudgets).Add(float64(max(failureBudgetsDeleted, 0)))
	m.cleanupDeleted.WithLabelValues(cleanupTableBuckets).Add(float64(max(requestBucketsDeleted, 0)))
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
