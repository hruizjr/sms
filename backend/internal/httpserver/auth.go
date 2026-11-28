package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/hruizjr/sms/backend/internal/apikey"
	"github.com/hruizjr/sms/backend/internal/middleware"
)

// Authenticator verifies API keys and records their use (implemented by apikey.Service).
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (apikey.Principal, error)
	// RecordUse records that the key was used for a request it is authorized for (throttled).
	RecordUse(ctx context.Context, p apikey.Principal) error
}

type principalKey struct{}

// principalFrom returns the authenticated API key of the request. Handlers behind requireAPIKey
// always have one.
func principalFrom(ctx context.Context) (apikey.Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(apikey.Principal)
	return p, ok
}

// requireAPIKey authenticates every /v1 request with a Bearer API key (contract security scheme
// api_key, D-143) and checks the permission the operation requires (x-required-permission):
// messages:send for POST, messages:read for GET. The tenant is taken from the key only; no request
// field can select it. It runs before parameter binding and body parsing. Once the key is
// authenticated and authorized, its last-used time is recorded (throttled, owner decision G4-6); a
// failure to record it is logged and does not fail the request, because the time is informational.
func requireAPIKey(auth Authenticator, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				unauthenticated(w, r)
				return
			}
			p, err := auth.Authenticate(r.Context(), token)
			switch {
			case errors.Is(err, apikey.ErrUnauthenticated):
				unauthenticated(w, r)
				return
			case errors.Is(err, apikey.ErrTenantNotActive):
				writePublicProblem(w, r, http.StatusForbidden, "forbidden", "Permission denied",
					"The account is not allowed to use the API.", nil)
				return
			case err != nil:
				logger.ErrorContext(r.Context(), "api key authentication failed",
					slog.String("request_id", middleware.RequestIDFromContext(r.Context())), slog.String("error", err.Error()))
				internalError(w, r)
				return
			}
			scope := apikey.ScopeMessagesRead
			if r.Method == http.MethodPost {
				scope = apikey.ScopeMessagesSend
			}
			if !p.Has(scope) {
				writePublicProblem(w, r, http.StatusForbidden, "forbidden", "Permission denied",
					"This API key does not have the "+string(scope)+" permission.", nil)
				return
			}
			if err := auth.RecordUse(r.Context(), p); err != nil {
				logger.WarnContext(r.Context(), "api key last-used time not recorded",
					slog.String("request_id", middleware.RequestIDFromContext(r.Context())),
					slog.String("tenant_id", p.TenantID.String()), slog.String("error", err.Error()))
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, p)))
		})
	}
}

// bearerToken returns the token of a single "Authorization: Bearer <token>" header.
func bearerToken(r *http.Request) (string, bool) {
	values := r.Header.Values("Authorization")
	if len(values) != 1 {
		return "", false
	}
	scheme, token, found := strings.Cut(values[0], " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != "" && !strings.ContainsAny(token, " \t")
}

func unauthenticated(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="api"`)
	writePublicProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required",
		"Provide a valid API key in the Authorization header.", nil)
}
