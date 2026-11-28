package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/netip"
	"slices"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hruizjr/sms/backend/internal/apikey"
	"github.com/hruizjr/sms/backend/internal/audit"
	"github.com/hruizjr/sms/backend/internal/messaging"
	"github.com/hruizjr/sms/backend/internal/middleware"
	"github.com/hruizjr/sms/backend/internal/portal"
	"github.com/hruizjr/sms/backend/internal/portalapi"
	"github.com/hruizjr/sms/backend/internal/signinlimit"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

// PortalSessionCookie is the name of the portal session cookie. The __Host- prefix makes browsers
// accept it only with Secure, Path=/ and no Domain, so no other host or path can set or read it.
const PortalSessionCookie = "__Host-sms_session"

// maxSignInRequestBytes bounds the sign-in body (an email address and a password of at most 1 KiB).
const maxSignInRequestBytes = 4 << 10

// PortalAuthenticator signs portal users in and out (implemented by portal.Service).
type PortalAuthenticator interface {
	Login(ctx context.Context, email, password string) (portal.Session, error)
	Authenticate(ctx context.Context, token string) (portal.Principal, error)
	Logout(ctx context.Context, p portal.Principal) error
	// Reauthenticate verifies the signed-in user's current password before a sensitive action, and
	// records a failure in the audit log as a failure of action.
	Reauthenticate(ctx context.Context, p portal.Principal, password string, action audit.Action) error
}

// PortalMessageService is what the portal needs from the messaging engine (implemented by
// messaging.Service; the same engine as the public API).
type PortalMessageService interface {
	MessageService
	GetWithHistory(ctx context.Context, tenantID tenancy.ID, id messaging.MessageID) (messaging.Message, []messaging.StatusChange, error)
	ListSenderIDs(ctx context.Context, tenantID tenancy.ID) ([]string, error)
}

// PortalAPIKeyService is what the portal needs to manage API keys (implemented by apikey.Service;
// the same service that verifies keys on the public API).
type PortalAPIKeyService interface {
	List(ctx context.Context, tenantID tenancy.ID) ([]apikey.Key, error)
	// Create and Revoke write their audit event in the same transaction as the change.
	Create(ctx context.Context, tenantID tenancy.ID, label string, scopes []apikey.Scope, actor audit.Actor, via string) (apikey.Key, string, error)
	Revoke(ctx context.Context, tenantID tenancy.ID, keyID [16]byte, actor audit.Actor, via string) (apikey.Key, error)
}

// PortalAuditService records security events that change nothing else (refusals) and lists an
// organization's audit log (implemented by audit.Service).
type PortalAuditService interface {
	Record(ctx context.Context, tenantID tenancy.ID, ev audit.Event) error
	RecordPlatform(ctx context.Context, ev audit.PlatformEvent) error
	List(ctx context.Context, tenantID tenancy.ID, f audit.Filter) ([]audit.Entry, string, error)
}

// SignInLimiter applies the layered per-source sign-in limits (implemented by signinlimit.Limiter).
type SignInLimiter interface {
	// Admit takes a token of the source's all-attempt bucket and reserves its failed-attempt slots.
	Admit(ctx context.Context, ip netip.Addr) (signinlimit.Admission, error)
	// Refund returns the slots of an attempt that succeeded or whose password was never checked.
	Refund(ctx context.Context, res signinlimit.Reservation) error
	Config() signinlimit.Config
}

// PortalDeps are the dependencies of the client portal API. When they are set, NewRouter mounts
// /portal/v1. All are required.
type PortalDeps struct {
	Auth          PortalAuthenticator
	Messages      PortalMessageService
	APIKeys       PortalAPIKeyService
	Audit         PortalAuditService
	SignInLimiter SignInLimiter
	// Now is the clock of the deduplication intervals of refusals recorded by the HTTP layer; nil
	// uses time.Now. (Audit event times are assigned by the database.)
	Now func() time.Time
}

type portalPrincipalKey struct{}

func portalPrincipalFrom(ctx context.Context) portal.Principal {
	p, _ := ctx.Value(portalPrincipalKey{}).(portal.Principal)
	return p
}

// mountPortalAPI registers the client portal API (contracts/openapi/portal-api-v1.yaml). Every
// route except sign-in requires a valid session; every state-changing route also requires the
// session's CSRF token; each handler checks the permission its operation requires. The tenant is
// taken from the session only.
func mountPortalAPI(r chi.Router, deps PortalDeps, logger *slog.Logger) {
	if deps.Auth == nil || deps.Messages == nil || deps.APIKeys == nil || deps.Audit == nil || deps.SignInLimiter == nil {
		panic("httpserver: every PortalDeps field is required") // programmer error at startup
	}
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	api := &portalAPI{
		auth: deps.Auth, svc: deps.Messages, keys: deps.APIKeys, audit: deps.Audit, limiter: deps.SignInLimiter,
		now: now, logger: logger,
	}
	r.Group(func(r chi.Router) {
		r.Use(api.requireSession)
		portalapi.HandlerWithOptions(api, portalapi.ChiServerOptions{
			BaseRouter:       r,
			ErrorHandlerFunc: portalParameterErrorHandler,
		})
	})
}

type portalAPI struct {
	auth    PortalAuthenticator
	svc     PortalMessageService
	keys    PortalAPIKeyService
	audit   PortalAuditService
	limiter SignInLimiter
	now     func() time.Time
	logger  *slog.Logger
}

var _ portalapi.ServerInterface = (*portalAPI)(nil)

// requireSession authenticates every portal request except sign-in, and checks the CSRF token of
// every state-changing request, before any parameter binding or body parsing.
func (a *portalAPI) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/portal/v1/session" {
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie(PortalSessionCookie)
		if err != nil || cookie.Value == "" {
			portalUnauthenticated(w, r)
			return
		}
		p, err := a.auth.Authenticate(r.Context(), cookie.Value)
		switch {
		case errors.Is(err, portal.ErrUnauthenticated):
			clearSessionCookie(w)
			portalUnauthenticated(w, r)
			return
		case errors.Is(err, portal.ErrTenantNotActive):
			clearSessionCookie(w)
			accountNotActive(w, r)
			return
		case err != nil:
			a.serverError(w, r, "portal session authentication", err)
			return
		}
		if !isSafeMethod(r.Method) {
			values := r.Header.Values("X-CSRF-Token")
			if len(values) != 1 || !p.CheckCSRF(values[0]) {
				writePortalProblem(w, r, http.StatusForbidden, "csrf-token-invalid", "CSRF token missing or invalid",
					"Reload the page and try again.", nil)
				return
			}
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), portalPrincipalKey{}, p)))
	})
}

// userIDString formats a user ID for logs (the standard UUID text form).
func userIDString(id [16]byte) string { return pgtype.UUID{Bytes: id, Valid: true}.String() }

func isSafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

// require writes 403 and returns false unless the session's role has permission.
func (a *portalAPI) require(w http.ResponseWriter, r *http.Request, permission portal.Permission) (portal.Principal, bool) {
	p := portalPrincipalFrom(r.Context())
	if !p.Has(permission) {
		forbidden(w, r, permission)
		return p, false
	}
	return p, true
}

// requireAudited is require for the protected operations whose refusals are security events (API key
// management and the audit log): a refusal is recorded as a denied action, at most once per user,
// action and audit.DeniedInterval (five minutes), so repeating a refused request cannot multiply
// audit writes. The request is refused whether or not the event could be written; a failure to write
// it is logged as an error.
func (a *portalAPI) requireAudited(w http.ResponseWriter, r *http.Request, permission portal.Permission, action audit.Action) (portal.Principal, bool) {
	p := portalPrincipalFrom(r.Context())
	if p.Has(permission) {
		return p, true
	}
	actor := audit.PortalUser(p.UserID)
	err := a.audit.Record(r.Context(), p.TenantID, audit.Event{
		Action: action, Outcome: audit.OutcomeDenied, Actor: actor,
		Details: audit.Details{
			Reason: audit.ReasonInsufficientRole, Role: string(p.Role), RequiredPermission: string(permission),
			DedupIntervalSeconds: int(audit.DeniedInterval / time.Second),
		},
		DedupKey: audit.DeniedKey(actor, action, a.now()),
	})
	if err != nil {
		a.logger.ErrorContext(r.Context(), "audit event of a refused request not recorded",
			slog.String("request_id", middleware.RequestIDFromContext(r.Context())),
			slog.String("tenant_id", p.TenantID.String()), slog.String("action", string(action)), slog.String("error", err.Error()))
	}
	forbidden(w, r, permission)
	return p, false
}

func forbidden(w http.ResponseWriter, r *http.Request, permission portal.Permission) {
	writePortalProblem(w, r, http.StatusForbidden, "forbidden", "Permission denied",
		"Your role does not allow this action ("+string(permission)+").", nil)
}

// SignIn implements POST /portal/v1/session.
func (a *portalAPI) SignIn(w http.ResponseWriter, r *http.Request, _ portalapi.SignInParams) {
	// The per-source limits apply to every request, before the body is read and before any
	// credential is checked, so a refusal reveals nothing about the account and never touches its
	// lockout state.
	admission, ok := a.admitSignIn(w, r)
	if !ok {
		return
	}
	raw, ok := readJSONBody(w, r, maxSignInRequestBytes)
	if !ok {
		a.refundSignIn(r, admission.Reservation) // no password was checked
		return
	}
	fields, errs, malformed := parseStringObject(raw, []string{"email", "password"}, []string{"email", "password"})
	if malformed {
		a.refundSignIn(r, admission.Reservation)
		malformedJSON(w, r)
		return
	}
	if len(errs) > 0 {
		a.refundSignIn(r, admission.Reservation)
		writePortalProblem(w, r, http.StatusBadRequest, problemValidationError, "The request is invalid", "One or more fields are invalid.", errs)
		return
	}
	session, err := a.auth.Login(r.Context(), fields["email"], fields["password"])
	switch {
	case errors.Is(err, portal.ErrInvalidCredentials):
		// The failure keeps its reserved slots.
		a.logger.InfoContext(r.Context(), "portal sign-in failed",
			slog.String("request_id", middleware.RequestIDFromContext(r.Context())))
		writePortalProblem(w, r, http.StatusUnauthorized, "invalid-credentials", "Sign-in failed",
			"The email address or password is incorrect, or the account is temporarily locked.", nil)
		return
	case errors.Is(err, portal.ErrTenantNotActive):
		// A correct password, but no session: counted like a failure (no refund).
		accountNotActive(w, r)
		return
	case errors.Is(err, portal.ErrBusy):
		// The password was never checked: the attempt returns its slots.
		a.refundSignIn(r, admission.Reservation)
		serviceBusy(w, r)
		return
	case err != nil:
		a.serverError(w, r, "portal sign-in", err)
		return
	}
	// A successful sign-in returns exactly its own slots; earlier failures stay counted.
	a.refundSignIn(r, admission.Reservation)
	p := session.Principal
	a.logger.InfoContext(r.Context(), "portal sign-in",
		slog.String("request_id", middleware.RequestIDFromContext(r.Context())),
		slog.String("tenant_id", p.TenantID.String()), slog.String("user_id", userIDString(p.UserID)))
	http.SetCookie(w, &http.Cookie{
		Name:     PortalSessionCookie,
		Value:    session.Token,
		Path:     "/",
		MaxAge:   max(int(time.Until(p.ExpiresAt).Seconds()), 1),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, http.StatusOK, toPortalSession(p))
}

// refundTimeout bounds a refund, which runs even if the client has gone away.
const refundTimeout = 5 * time.Second

// refundSignIn returns an attempt's failed-attempt slots. A refund that fails leaves the slots
// counted (the safe direction) and is logged.
func (a *portalAPI) refundSignIn(r *http.Request, res signinlimit.Reservation) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), refundTimeout)
	defer cancel()
	if err := a.limiter.Refund(ctx, res); err != nil {
		a.logger.ErrorContext(r.Context(), "sign-in failure slot not refunded",
			slog.String("request_id", middleware.RequestIDFromContext(r.Context())), slog.String("error", err.Error()))
	}
}

// admitSignIn applies the per-source sign-in limits to the request's source. When a limit refuses
// the attempt it writes 429 with Retry-After and returns false. The first refusal of a failure
// budget's window, or of a token bucket's report interval, is recorded as a platform audit event (no
// organization is identified); later refusals are not recorded one by one, so a flood cannot multiply
// audit writes.
func (a *portalAPI) admitSignIn(w http.ResponseWriter, r *http.Request) (signinlimit.Admission, bool) {
	ip := middleware.ClientIPFromContext(r.Context())
	admission, err := a.limiter.Admit(r.Context(), ip)
	if err != nil {
		a.serverError(w, r, "portal sign-in rate limit", err)
		return admission, false
	}
	if admission.Allowed {
		return admission, true
	}
	refusal := admission.Refusal
	if refusal.Report {
		cfg := a.limiter.Config()
		details := audit.Details{Limit: refusal.Limit, SourcePrefixLength: refusal.Source.Bits()}
		if refusal.Layer == signinlimit.LayerRequestRate {
			details.Reason, details.RatePerSecond = audit.ReasonTooManyRequests, cfg.RequestsPerSecond
		} else {
			details.Reason, details.WindowSeconds = audit.ReasonTooManyFailures, int(cfg.FailureWindow/time.Second)
		}
		err := a.audit.RecordPlatform(r.Context(), audit.PlatformEvent{
			Action: audit.ActionSignInRateLimited, Outcome: audit.OutcomeDenied, Details: details,
		})
		if err != nil {
			a.logger.ErrorContext(r.Context(), "audit event of a rate-limited sign-in not recorded",
				slog.String("request_id", middleware.RequestIDFromContext(r.Context())), slog.String("error", err.Error()))
		}
	}
	a.logger.InfoContext(r.Context(), "portal sign-in rate limited",
		slog.String("request_id", middleware.RequestIDFromContext(r.Context())), slog.String("limit", string(refusal.Layer)))
	w.Header().Set("Retry-After", strconv.FormatInt(int64(refusal.RetryAfter/time.Second), 10))
	detail := "Too many failed sign-in attempts from your network. Wait before trying again."
	if refusal.Layer == signinlimit.LayerRequestRate {
		detail = "Too many sign-in requests from your network. Wait before trying again."
	}
	writePortalProblem(w, r, http.StatusTooManyRequests, "too-many-sign-in-attempts", "Too many sign-in attempts", detail, nil)
	return admission, false
}

// serviceBusyRetryAfter is the Retry-After of a 503 caused by exhausted password-hashing capacity.
const serviceBusyRetryAfter = "2"

// serviceBusy writes 503 (service-busy): the password-hashing capacity is exhausted; nothing was checked.
func serviceBusy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Retry-After", serviceBusyRetryAfter)
	writePortalProblem(w, r, http.StatusServiceUnavailable, "service-busy", "Service busy",
		"The service is busy. Try again in a moment.", nil)
}

// GetSession implements GET /portal/v1/session.
func (a *portalAPI) GetSession(w http.ResponseWriter, r *http.Request, _ portalapi.GetSessionParams) {
	writeJSON(w, http.StatusOK, toPortalSession(portalPrincipalFrom(r.Context())))
}

// SignOut implements DELETE /portal/v1/session.
func (a *portalAPI) SignOut(w http.ResponseWriter, r *http.Request, _ portalapi.SignOutParams) {
	p := portalPrincipalFrom(r.Context())
	if err := a.auth.Logout(r.Context(), p); err != nil {
		a.serverError(w, r, "portal sign-out", err)
		return
	}
	a.logger.InfoContext(r.Context(), "portal sign-out",
		slog.String("request_id", middleware.RequestIDFromContext(r.Context())),
		slog.String("tenant_id", p.TenantID.String()), slog.String("user_id", userIDString(p.UserID)))
	clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// ListSenderIds implements GET /portal/v1/sender-ids.
func (a *portalAPI) ListSenderIds(w http.ResponseWriter, r *http.Request, _ portalapi.ListSenderIdsParams) {
	p, ok := a.require(w, r, portal.PermissionMessagesRead)
	if !ok {
		return
	}
	ids, err := a.svc.ListSenderIDs(r.Context(), p.TenantID)
	if err != nil {
		a.serverError(w, r, "list sender IDs", err)
		return
	}
	writeJSON(w, http.StatusOK, portalapi.SenderIdList{Data: ids})
}

// EstimateMessage implements POST /portal/v1/message-estimates with the one shared segmentation
// function (messaging.Segment).
func (a *portalAPI) EstimateMessage(w http.ResponseWriter, r *http.Request, _ portalapi.EstimateMessageParams) {
	if _, ok := a.require(w, r, portal.PermissionMessagesSend); !ok {
		return
	}
	raw, ok := readJSONBody(w, r, maxMessageRequestBytes)
	if !ok {
		return
	}
	fields, errs, malformed := parseStringObject(raw, []string{"body"}, []string{"body"})
	if malformed {
		malformedJSON(w, r)
		return
	}
	body := fields["body"]
	if len(errs) == 0 && utf8.RuneCountInString(body) > messaging.MaxBodyCharacters {
		errs = append(errs, portalBodyError("/body", "too_long", "Must be at most 1600 characters."))
	}
	if len(errs) > 0 {
		writePortalProblem(w, r, http.StatusBadRequest, problemValidationError, "The request is invalid", "One or more fields are invalid.", errs)
		return
	}
	est := portalapi.MessageEstimate{
		Encoding:      portalapi.Encoding(messaging.EncodingGSM7),
		Characters:    utf8.RuneCountInString(body),
		MaxCharacters: messaging.MaxBodyCharacters,
	}
	if body != "" {
		seg := messaging.Segment(body)
		est.Encoding, est.SegmentCount = portalapi.Encoding(seg.Encoding), seg.Segments
	}
	writeJSON(w, http.StatusOK, est)
}

// CreateMessage implements POST /portal/v1/messages through the same messaging engine, with the
// same idempotency handling, as POST /v1/messages.
func (a *portalAPI) CreateMessage(w http.ResponseWriter, r *http.Request, params portalapi.CreateMessageParams) {
	p, ok := a.require(w, r, portal.PermissionMessagesSend)
	if !ok {
		return
	}
	key, ok := normalizeIdempotencyKey(params.IdempotencyKey)
	if !ok {
		writePortalProblem(w, r, http.StatusBadRequest, "idempotency-key-invalid", "Idempotency-Key header is invalid",
			"The Idempotency-Key must be 1 to 255 visible ASCII characters, optionally in double quotes.", nil)
		return
	}
	raw, ok := readJSONBody(w, r, maxMessageRequestBytes)
	if !ok {
		return
	}
	// The public API's validation, without client_reference (not part of the portal request).
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields) // a non-object is reported by parseMessageRequest below
	req, publicErrs, malformed := parseMessageRequest(raw)
	if malformed {
		malformedJSON(w, r)
		return
	}
	errs := make([]portalapi.FieldError, 0, len(publicErrs)+1)
	for _, e := range publicErrs {
		errs = append(errs, portalapi.FieldError{Pointer: e.Pointer, Parameter: e.Parameter, Code: e.Code, Detail: e.Detail})
	}
	if _, present := fields["client_reference"]; present {
		errs = append(errs, portalBodyError("/client_reference", "unknown_field", "This field is not allowed."))
	}
	if len(errs) > 0 {
		writePortalProblem(w, r, http.StatusBadRequest, problemValidationError, "The request is invalid", "One or more fields are invalid.", errs)
		return
	}
	req.ClientReference = nil
	req.IdempotencyKey = key

	res, err := a.svc.Accept(r.Context(), p.TenantID, req)
	switch {
	case errors.Is(err, messaging.ErrIdempotencyKeyReused):
		writePortalProblem(w, r, http.StatusUnprocessableEntity, "idempotency-key-reused", "Idempotency-Key reused with a different request",
			"This Idempotency-Key was already used for a different message.", nil)
	case errors.Is(err, messaging.ErrIdempotencyInProgress):
		w.Header().Set("Retry-After", "1")
		writePortalProblem(w, r, http.StatusConflict, "idempotency-request-in-progress", "A request with this Idempotency-Key is in progress",
			"Retry after the indicated delay with the same key and body.", nil)
	case errors.Is(err, messaging.ErrSenderIDNotApproved):
		writePortalProblem(w, r, http.StatusUnprocessableEntity, "sender-id-not-approved", "Sender ID not approved",
			"The sender ID is not an approved sender ID for this account.", nil)
	case errors.Is(err, messaging.ErrDestinationNotSupported):
		writePortalProblem(w, r, http.StatusUnprocessableEntity, "destination-not-supported", "Destination not supported",
			"Only Philippine mobile numbers (+639...) are supported.", nil)
	case errors.Is(err, messaging.ErrMessageTooLong):
		writePortalProblem(w, r, http.StatusUnprocessableEntity, "message-too-long", "Message too long",
			"The message needs more segments than are allowed.", nil)
	case err != nil:
		a.serverError(w, r, "accept portal message", err)
	default:
		if res.Replayed {
			w.Header().Set("Idempotent-Replayed", "true")
		}
		writeJSON(w, http.StatusAccepted, toPortalMessage(res.Message))
	}
}

// ListMessages implements GET /portal/v1/messages.
func (a *portalAPI) ListMessages(w http.ResponseWriter, r *http.Request, params portalapi.ListMessagesParams) {
	p, ok := a.require(w, r, portal.PermissionMessagesRead)
	if !ok {
		return
	}
	f := messaging.ListFilter{Limit: 25}
	var errs []portalapi.FieldError
	if params.Limit != nil {
		if *params.Limit < 1 || *params.Limit > 100 {
			errs = append(errs, portalParamError("limit", "out_of_range", "Must be from 1 to 100."))
		}
		f.Limit = *params.Limit
	}
	if params.Status != nil {
		s := messaging.Status(*params.Status)
		if s.States() == nil {
			errs = append(errs, portalParamError("status", "invalid_value", "Must be one of the documented message statuses."))
		}
		f.Status = &s
	}
	if params.Cursor != nil {
		if n := len(*params.Cursor); n < 1 || n > 512 {
			errs = append(errs, portalParamError("cursor", "invalid_cursor", "The cursor is invalid or expired."))
		}
		f.Cursor = *params.Cursor
	}
	if len(errs) > 0 {
		writePortalProblem(w, r, http.StatusBadRequest, problemValidationError, "The request is invalid", "One or more parameters are invalid.", errs)
		return
	}
	messages, next, err := a.svc.List(r.Context(), p.TenantID, f)
	switch {
	case errors.Is(err, messaging.ErrInvalidCursor):
		writePortalProblem(w, r, http.StatusBadRequest, problemValidationError, "The request is invalid", "One or more parameters are invalid.",
			[]portalapi.FieldError{portalParamError("cursor", "invalid_cursor", "The cursor is invalid or expired.")})
	case err != nil:
		a.serverError(w, r, "list portal messages", err)
	default:
		page := portalapi.MessageList{Data: make([]portalapi.Message, len(messages))}
		for i, m := range messages {
			page.Data[i] = toPortalMessage(m)
		}
		if next != "" {
			page.NextCursor = &next
		}
		writeJSON(w, http.StatusOK, page)
	}
}

// GetMessage implements GET /portal/v1/messages/{message_id}.
func (a *portalAPI) GetMessage(w http.ResponseWriter, r *http.Request, messageID portalapi.MessageId, _ portalapi.GetMessageParams) {
	p, ok := a.require(w, r, portal.PermissionMessagesRead)
	if !ok {
		return
	}
	if !messageIDPattern.MatchString(messageID) {
		writePortalProblem(w, r, http.StatusBadRequest, problemValidationError, "The request is invalid", "One or more parameters are invalid.",
			[]portalapi.FieldError{portalParamError("message_id", "invalid_format", "Must be a message id returned at acceptance.")})
		return
	}
	id, err := messaging.ParseMessageID(messageID)
	if err != nil {
		notFound(w, r)
		return
	}
	m, history, err := a.svc.GetWithHistory(r.Context(), p.TenantID, id)
	switch {
	case errors.Is(err, messaging.ErrNotFound):
		notFound(w, r) // another tenant's message is indistinguishable from a missing one
	case err != nil:
		a.serverError(w, r, "get portal message", err)
	default:
		base := toPortalMessage(m)
		detail := portalapi.MessageDetail{
			Id: base.Id, To: base.To, SenderId: base.SenderId, Status: base.Status, StatusCategory: base.StatusCategory,
			StatusReason: base.StatusReason, Encoding: base.Encoding, SegmentCount: base.SegmentCount,
			AcceptedAt: base.AcceptedAt, StatusUpdatedAt: base.StatusUpdatedAt,
			StatusHistory: make([]portalapi.StatusChange, len(history)),
		}
		for i, h := range history {
			detail.StatusHistory[i] = portalapi.StatusChange{
				Status: portalapi.MessageStatus(h.Status), StatusCategory: portalapi.StatusCategory(h.Status.Category()),
				StatusReason: portalReason(h.Reason), OccurredAt: h.OccurredAt.UTC(),
			}
		}
		writeJSON(w, http.StatusOK, detail)
	}
}

func (a *portalAPI) serverError(w http.ResponseWriter, r *http.Request, op string, err error) {
	a.logger.ErrorContext(r.Context(), op+" failed",
		slog.String("request_id", middleware.RequestIDFromContext(r.Context())), slog.String("error", err.Error()))
	internalError(w, r)
}

func toPortalSession(p portal.Principal) portalapi.Session {
	s := portalapi.Session{CsrfToken: p.CSRFToken, ExpiresAt: p.ExpiresAt.UTC(), IdleExpiresAt: p.IdleExpiresAt.UTC()}
	s.User.Email = p.Email
	s.User.Role = portalapi.Role(p.Role)
	s.Organization.Name = p.TenantName
	for _, perm := range p.Role.Permissions() {
		s.Permissions = append(s.Permissions, portalapi.Permission(perm))
	}
	if s.Permissions == nil {
		s.Permissions = []portalapi.Permission{}
	}
	return s
}

func toPortalMessage(m messaging.Message) portalapi.Message {
	status := m.State.Status()
	return portalapi.Message{
		Id:              m.ID.String(),
		To:              m.Destination,
		SenderId:        m.SenderID,
		Status:          portalapi.MessageStatus(status),
		StatusCategory:  portalapi.StatusCategory(status.Category()),
		StatusReason:    portalReason(m.Reason),
		Encoding:        portalapi.Encoding(m.Encoding),
		SegmentCount:    m.SegmentCount,
		AcceptedAt:      m.AcceptedAt.UTC(),
		StatusUpdatedAt: m.StatusUpdatedAt.UTC(),
	}
}

func portalReason(r *messaging.Reason) *portalapi.StatusReason {
	if r == nil {
		return nil
	}
	return &portalapi.StatusReason{Code: string(*r), Description: r.Description()}
}

// readJSONBody checks the media type and reads at most limit bytes. On failure it writes the
// problem (415, 413 or 400) and returns false.
func readJSONBody(w http.ResponseWriter, r *http.Request, limit int64) ([]byte, bool) {
	if mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mediaType != "application/json" {
		writePortalProblem(w, r, http.StatusUnsupportedMediaType, "unsupported-media-type", "Unsupported media type",
			"Send the request body as application/json.", nil)
		return nil, false
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limit))
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		writePortalProblem(w, r, http.StatusRequestEntityTooLarge, "payload-too-large", "Request body too large",
			"The request body exceeds the allowed size.", nil)
		return nil, false
	case err != nil:
		malformedJSON(w, r)
		return nil, false
	}
	return raw, true
}

// parseStringObject parses a JSON object whose properties are all strings: allowed lists the
// permitted properties (additionalProperties: false) and required those that must be present.
func parseStringObject(raw []byte, allowed, required []string) (map[string]string, []portalapi.FieldError, bool) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	var top any
	if err := dec.Decode(&top); err != nil {
		return nil, nil, true
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, nil, true
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return nil, []portalapi.FieldError{portalBodyError("", "invalid_type", "The body must be a JSON object.")}, false
	}
	out := make(map[string]string, len(fields))
	var errs []portalapi.FieldError
	for _, name := range required {
		if _, ok := fields[name]; !ok {
			errs = append(errs, portalBodyError("/"+name, "required", "This field is required."))
		}
	}
	for _, name := range allowed {
		v, ok := fields[name]
		if !ok {
			continue
		}
		var s string
		if bytes.Equal(bytes.TrimSpace(v), []byte("null")) || json.Unmarshal(v, &s) != nil {
			errs = append(errs, portalBodyError("/"+name, "invalid_type", "Must be a string."))
			continue
		}
		out[name] = s
	}
	for name := range fields {
		if _, ok := out[name]; !ok && !slices.Contains(allowed, name) {
			errs = append(errs, portalBodyError("/"+jsonPointerEscape(name), "unknown_field", "This field is not allowed."))
		}
	}
	return out, errs, false
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: PortalSessionCookie, Value: "", Path: "/", MaxAge: -1,
		Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode,
	})
}

func portalUnauthenticated(w http.ResponseWriter, r *http.Request) {
	writePortalProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required", "Sign in to continue.", nil)
}

func accountNotActive(w http.ResponseWriter, r *http.Request) {
	writePortalProblem(w, r, http.StatusForbidden, "account-not-active", "Account not active",
		"Your organization's account is not active. Contact support.", nil)
}

func malformedJSON(w http.ResponseWriter, r *http.Request) {
	writePortalProblem(w, r, http.StatusBadRequest, problemMalformedRequest,
		"The request body is not valid JSON", "The request body could not be parsed as JSON.", nil)
}

func portalBodyError(pointer, code, detail string) portalapi.FieldError {
	return portalapi.FieldError{Pointer: &pointer, Code: code, Detail: detail}
}

func portalParamError(name, code, detail string) portalapi.FieldError {
	return portalapi.FieldError{Parameter: &name, Code: code, Detail: detail}
}

// writePortalProblem writes an RFC 9457 problem document of the portal API. detail must be a
// fixed, client-safe message.
func writePortalProblem(w http.ResponseWriter, r *http.Request, status int, slug, title, detail string, fieldErrors []portalapi.FieldError) {
	body := portalapi.Problem{Type: problemTypeBase + slug, Title: title, Status: status, Detail: &detail}
	if requestID := middleware.RequestIDFromContext(r.Context()); requestID != "" {
		body.RequestId = &requestID
	}
	if len(fieldErrors) > 0 {
		body.Errors = &fieldErrors
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body) // see writeProblem for why the error is not actionable
}

// portalParameterErrorHandler turns a generated parameter-binding error into a 400 problem.
func portalParameterErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	var missingHeader *portalapi.RequiredHeaderError
	if errors.As(err, &missingHeader) {
		switch missingHeader.ParamName {
		case "Idempotency-Key":
			writePortalProblem(w, r, http.StatusBadRequest, problemIdempotencyKeyMissing, "Idempotency-Key header is required",
				"Send an Idempotency-Key header that is unique for this operation.", nil)
			return
		case "X-CSRF-Token": // checked earlier by requireSession; kept for completeness
			writePortalProblem(w, r, http.StatusForbidden, "csrf-token-invalid", "CSRF token missing or invalid",
				"Reload the page and try again.", nil)
			return
		}
	}
	name := ""
	var (
		invalidFormat *portalapi.InvalidParamFormatError
		tooMany       *portalapi.TooManyValuesForParamError
		requiredParam *portalapi.RequiredParamError
	)
	switch {
	case errors.As(err, &invalidFormat):
		name = invalidFormat.ParamName
	case errors.As(err, &tooMany):
		name = tooMany.ParamName
	case errors.As(err, &requiredParam):
		name = requiredParam.ParamName
	}
	var fieldErrors []portalapi.FieldError
	if name != "" {
		fieldErrors = []portalapi.FieldError{portalParamError(name, "invalid_value", "The value could not be parsed as the declared type.")}
	}
	writePortalProblem(w, r, http.StatusBadRequest, problemValidationError, "The request is invalid", "One or more parameters are invalid.", fieldErrors)
}
