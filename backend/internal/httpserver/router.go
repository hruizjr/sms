package httpserver

import (
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hruizjr/sms/backend/internal/audit"
	"github.com/hruizjr/sms/backend/internal/middleware"
)

// MaxRequestBodyBytes caps request bodies for every route (AGENTS.md §9). Routes may set a lower
// limit (POST /v1/messages does).
const MaxRequestBodyBytes = 1 << 20 // 1 MiB

// RouterDeps are the dependencies of the application router.
type RouterDeps struct {
	Logger    *slog.Logger
	Metrics   middleware.RequestObserver
	Readiness *Readiness
	// Database is pinged by /readyz. It is required.
	Database Pinger
	// ReadinessTimeout bounds the database ping made by /readyz; zero uses
	// DefaultReadinessTimeout.
	ReadinessTimeout time.Duration
	// NewRequestID generates request IDs; nil uses middleware.NewRandomID.
	NewRequestID middleware.IDGenerator
	// TrustedProxies are the proxies whose X-Forwarded-For header names the client address; empty
	// means the direct peer address is always used (middleware.ClientIP).
	TrustedProxies []netip.Prefix
	// PublicAPI, when set, mounts the public API v1 (POST /v1/messages, GET /v1/messages and
	// GET /v1/messages/{message_id}) behind API key authentication. Without it, /v1 answers 404.
	PublicAPI *PublicAPIDeps
	// Portal, when set, mounts the client portal API (/portal/v1, session-authenticated;
	// contracts/openapi/portal-api-v1.yaml). Without it, /portal answers 404.
	Portal *PortalDeps
}

// NewRouter returns the application HTTP handler with its middleware chain and routes.
//
// Middleware order (outermost first): request ID → client IP → audit source → security headers →
// access log → metrics → panic recovery → request body limit → route handler. Recovery sits inside
// logging and metrics so that a recovered panic is logged and counted as a 500. The client IP and
// the audit source are stored in the request context only (the access log never records them).
func NewRouter(deps RouterDeps) http.Handler {
	return newRouter(deps, nil)
}

// newRouter lets tests mount extra routes behind the production middleware chain.
func newRouter(deps RouterDeps, extra func(chi.Router)) http.Handler {
	if deps.Database == nil {
		panic("httpserver: RouterDeps.Database is required") // programmer error at startup
	}
	timeout := deps.ReadinessTimeout
	if timeout <= 0 {
		timeout = DefaultReadinessTimeout
	}

	r := chi.NewRouter()
	r.Use(
		middleware.RequestID(deps.NewRequestID),
		middleware.ClientIP(deps.TrustedProxies),
		auditSource,
		middleware.SecurityHeaders,
		middleware.AccessLog(deps.Logger),
		middleware.Metrics(deps.Metrics),
		middleware.Recover(deps.Logger, http.HandlerFunc(internalError)),
		middleware.LimitRequestBody(MaxRequestBodyBytes),
	)
	r.NotFound(notFound)
	r.MethodNotAllowed(methodNotAllowed)

	r.Get("/healthz", healthz)
	r.Method(http.MethodGet, "/readyz", &readinessCheck{
		readiness: deps.Readiness,
		database:  deps.Database,
		timeout:   timeout,
		logger:    deps.Logger,
	})

	if deps.PublicAPI != nil {
		mountMessagesAPI(r, *deps.PublicAPI, deps.Logger)
	}
	if deps.Portal != nil {
		mountPortalAPI(r, *deps.Portal, deps.Logger)
	}
	if extra != nil {
		extra(r)
	}
	return r
}

// auditSource stores the request source of security audit events (request ID, client IP address,
// sanitized user agent) in the request context (audit.WithSource).
func auditSource(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := audit.WithSource(r.Context(), audit.Source{
			RequestID: middleware.RequestIDFromContext(r.Context()),
			IP:        middleware.ClientIPFromContext(r.Context()),
			UserAgent: r.UserAgent(),
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
