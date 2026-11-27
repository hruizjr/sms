package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

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
	// PublicAPI, when set, mounts the public API v1 (POST /v1/messages, GET /v1/messages and
	// GET /v1/messages/{message_id}) behind API key authentication. Without it, /v1 answers 404.
	PublicAPI *PublicAPIDeps
	// Portal, when set, mounts the client portal API (/portal/v1, session-authenticated;
	// contracts/openapi/portal-api-v1.yaml). Without it, /portal answers 404.
	Portal *PortalDeps
}

// NewRouter returns the application HTTP handler with its middleware chain and routes.
//
// Middleware order (outermost first): request ID → security headers → access log → metrics →
// panic recovery → request body limit → route handler. Recovery sits inside logging and metrics
// so that a recovered panic is logged and counted as a 500.
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
