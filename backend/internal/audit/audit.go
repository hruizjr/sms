// Package audit records security events in the durable, append-only security audit log (owner
// decision G4-8; security baseline §9, S-8; Phase 4 Group 6) and lists an organization's events for
// its owners.
//
// # Two tables
//
// An event that belongs to an organization (a sign-in of one of its users, an API key change, a
// refused request of a signed-in user) is stored in security_audit_events, a tenant-owned table under
// forced row-level security: it is written and read only inside that tenant's transaction-local
// context, and only the organization's owners can list it through the portal API. An event that
// belongs to no identifiable organization (a sign-in for an address that is not registered, a
// rate-limited sign-in) is stored in platform_security_events, which has no tenant column; the
// application can insert into it but cannot read it, so no portal user can see platform-wide records.
//
// # Append-only
//
// The application role may insert (and, for its own tenant, read) audit events but has no UPDATE,
// DELETE or TRUNCATE privilege; triggers additionally reject those statements by any role, including
// the table owner. Database superusers, and the table owner after disabling the triggers, can still
// alter rows: the log is append-only under the application's permissions, not tamper-proof.
//
// # Same transaction
//
// Insert writes an event through the caller's transaction, so a security-sensitive change (an API key
// created or revoked, a session started) and its audit event are committed together or not at all:
// if the event cannot be written, the change is rolled back and the caller reports an error.
//
// # What is never recorded
//
// Passwords, session tokens, API keys and their hashes, authorization headers and message text are
// never part of an event: the request context contributes only the request ID, the client IP address
// and a sanitized user agent, and Details has a fixed set of non-secret fields.
package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hruizjr/sms/backend/internal/database/sqlcgen"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

// Action is what happened. Its values are stored and returned by the portal API.
type Action string

// Actions of tenant events (security_audit_events).
//
//nolint:gosec // G101 false positive: these are audit action names, not credentials.
const (
	// ActionSignIn: a portal sign-in (success: a session started; failure: refused credentials).
	ActionSignIn Action = "portal.sign_in"
	// ActionSignOut: a portal user ended their session.
	ActionSignOut Action = "portal.sign_out"
	// ActionAccountLocked: repeated failed sign-ins or re-authentications locked the account.
	ActionAccountLocked Action = "portal.account_locked"
	// ActionAPIKeyList: listing the API keys (recorded only when refused).
	ActionAPIKeyList Action = "api_key.list"
	// ActionAPIKeyCreate: creating an API key (success, failed re-authentication, or refused).
	ActionAPIKeyCreate Action = "api_key.create"
	// ActionAPIKeyRevoke: revoking an API key (success or refused).
	ActionAPIKeyRevoke Action = "api_key.revoke"
	// ActionAuditLogList: reading the audit log (recorded only when refused).
	ActionAuditLogList Action = "audit_log.list"
)

// Actions of platform events (platform_security_events): no organization can be identified.
const (
	// ActionSignInRateLimited: sign-in attempts from one source exceeded the per-source limit.
	ActionSignInRateLimited Action = "portal.sign_in_rate_limited"
)

// TenantActions are the actions of tenant events, in display order.
var TenantActions = []Action{
	ActionSignIn, ActionSignOut, ActionAccountLocked,
	ActionAPIKeyCreate, ActionAPIKeyRevoke, ActionAPIKeyList, ActionAuditLogList,
}

// platformActions are the actions of platform events.
var platformActions = []Action{ActionSignIn, ActionSignInRateLimited}

// Outcome is the result of the action.
type Outcome string

// Outcomes.
const (
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure" // the actor could not prove who they are (credentials)
	OutcomeDenied  Outcome = "denied"  // refused by policy: missing permission, rate limit
)

// Valid reports whether o is a known outcome.
func (o Outcome) Valid() bool {
	return o == OutcomeSuccess || o == OutcomeFailure || o == OutcomeDenied
}

// ActorType is who performed the action.
type ActorType string

// Actor types.
const (
	ActorPortalUser ActorType = "portal_user" // a signed-in portal user (ID: the user)
	ActorAnonymous  ActorType = "anonymous"   // an unauthenticated caller
	ActorSystem     ActorType = "system"      // the platform itself, or the local administration tool
)

// Actor is who performed the action.
type Actor struct {
	Type ActorType
	ID   [16]byte // the portal user's ID; zero for anonymous and system actors
}

// PortalUser is the actor of a signed-in portal user.
func PortalUser(id [16]byte) Actor { return Actor{Type: ActorPortalUser, ID: id} }

// Anonymous is the actor of an unauthenticated request.
var Anonymous = Actor{Type: ActorAnonymous}

// System is the platform itself or the local administration tool (smsctl).
var System = Actor{Type: ActorSystem}

// TargetType is the kind of resource an action concerned.
type TargetType string

// Target types.
const (
	TargetPortalUser    TargetType = "portal_user"
	TargetPortalSession TargetType = "portal_session"
	TargetAPIKey        TargetType = "api_key"
)

// Target is the resource an action concerned.
type Target struct {
	Type TargetType
	ID   [16]byte
}

// Reason explains a failure, a refusal or a lock. Its values are stored and returned.
type Reason string

// Reasons.
const (
	ReasonInvalidPassword       Reason = "invalid_password"
	ReasonAccountLocked         Reason = "account_locked"
	ReasonUnknownAccount        Reason = "unknown_account" // platform events only
	ReasonInvalidInput          Reason = "invalid_input"   // platform events only: not an acceptable email or password
	ReasonOrganizationNotActive Reason = "organization_not_active"
	ReasonInsufficientRole      Reason = "insufficient_role"
	ReasonTooManyAttempts       Reason = "too_many_attempts"
)

// Details are the event's structured metadata. The fields are fixed and none holds a secret: an API
// key's name and permissions are not secret, its value and hash are never included.
type Details struct {
	Reason Reason `json:"reason,omitempty"`
	// APIKeyName and APIKeyScopes describe the key that was created or revoked.
	APIKeyName   string   `json:"api_key_name,omitempty"`
	APIKeyScopes []string `json:"api_key_scopes,omitempty"`
	// Role and RequiredPermission describe a refused request.
	Role               string `json:"role,omitempty"`
	RequiredPermission string `json:"required_permission,omitempty"`
	// LockedUntil is the end of an account lock.
	LockedUntil *time.Time `json:"locked_until,omitempty"`
	// Via names the local administration tool for actions it performed ("smsctl").
	Via string `json:"via,omitempty"`
	// MaxAttempts and WindowSeconds are the rate limit that was exceeded.
	MaxAttempts   int `json:"max_attempts,omitempty"`
	WindowSeconds int `json:"window_seconds,omitempty"`
}

// Event is a tenant security event.
type Event struct {
	OccurredAt time.Time
	Action     Action
	Outcome    Outcome
	Actor      Actor
	Target     *Target // nil when the action concerns no single resource
	Details    Details
}

// PlatformEvent is a security event that belongs to no identifiable organization.
type PlatformEvent struct {
	OccurredAt time.Time
	Action     Action
	Outcome    Outcome
	Details    Details
}

// Source is the request an event came from. Every field is optional (the local administration tool
// has none).
type Source struct {
	RequestID string
	IP        netip.Addr
	UserAgent string
}

type sourceKey struct{}

// WithSource returns ctx carrying the request source of the events recorded while handling it.
func WithSource(ctx context.Context, s Source) context.Context {
	return context.WithValue(ctx, sourceKey{}, s)
}

// SourceFrom returns the request source stored by WithSource (zero outside an HTTP request).
func SourceFrom(ctx context.Context) Source {
	s, _ := ctx.Value(sourceKey{}).(Source)
	return s
}

// MaxUserAgentCharacters bounds the stored user agent.
const MaxUserAgentCharacters = 256

// NormalizeUserAgent returns ua without control characters, trimmed and cut to
// MaxUserAgentCharacters characters ("" for an empty or invalid value).
func NormalizeUserAgent(ua string) string {
	if !utf8.ValidString(ua) {
		return ""
	}
	ua = strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, ua))
	if utf8.RuneCountInString(ua) > MaxUserAgentCharacters {
		ua = strings.TrimSpace(string([]rune(ua)[:MaxUserAgentCharacters]))
	}
	return ua
}

// ErrInvalidEvent reports an event the code should never build (a programmer error).
var ErrInvalidEvent = errors.New("invalid audit event")

// Insert writes ev for tenantID through tx. tx MUST be a transaction of database.DB.InTenantTx for
// tenantID, so that the event commits or rolls back with the change it records. The request source is
// taken from ctx (WithSource).
func Insert(ctx context.Context, tx pgx.Tx, tenantID tenancy.ID, ev Event) error {
	if err := ev.validate(); err != nil {
		return err
	}
	details, err := json.Marshal(ev.Details)
	if err != nil {
		return fmt.Errorf("%w: details: %w", ErrInvalidEvent, err)
	}
	src := SourceFrom(ctx)
	params := sqlcgen.InsertSecurityAuditEventParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		OccurredAt: ev.OccurredAt.UTC(),
		Action:     string(ev.Action),
		Outcome:    string(ev.Outcome),
		ActorType:  string(ev.Actor.Type),
		RequestID:  optional(src.RequestID),
		SourceIp:   optionalAddr(src.IP),
		UserAgent:  optional(NormalizeUserAgent(src.UserAgent)),
		Details:    details,
	}
	if ev.Actor.Type == ActorPortalUser {
		params.ActorID = pgtype.UUID{Bytes: ev.Actor.ID, Valid: true}
	}
	if ev.Target != nil {
		t := string(ev.Target.Type)
		params.TargetType = &t
		params.TargetID = pgtype.UUID{Bytes: ev.Target.ID, Valid: true}
	}
	if err := sqlcgen.New(tx).InsertSecurityAuditEvent(ctx, params); err != nil {
		return fmt.Errorf("record audit event %s: %w", ev.Action, err)
	}
	return nil
}

// InsertPlatform writes a platform event through db (a pool, a connection or a transaction; no tenant
// context is needed or used). The request source is taken from ctx (WithSource).
func InsertPlatform(ctx context.Context, db sqlcgen.DBTX, ev PlatformEvent) error {
	if !slices.Contains(platformActions, ev.Action) || !ev.Outcome.Valid() || ev.OccurredAt.IsZero() {
		return fmt.Errorf("%w: platform event %q (%q)", ErrInvalidEvent, ev.Action, ev.Outcome)
	}
	details, err := json.Marshal(ev.Details)
	if err != nil {
		return fmt.Errorf("%w: details: %w", ErrInvalidEvent, err)
	}
	src := SourceFrom(ctx)
	err = sqlcgen.New(db).InsertPlatformSecurityEvent(ctx, sqlcgen.InsertPlatformSecurityEventParams{
		OccurredAt: ev.OccurredAt.UTC(),
		Action:     string(ev.Action),
		Outcome:    string(ev.Outcome),
		RequestID:  optional(src.RequestID),
		SourceIp:   optionalAddr(src.IP),
		UserAgent:  optional(NormalizeUserAgent(src.UserAgent)),
		Details:    details,
	})
	if err != nil {
		return fmt.Errorf("record platform security event %s: %w", ev.Action, err)
	}
	return nil
}

func (ev Event) validate() error {
	switch {
	case !slices.Contains(TenantActions, ev.Action):
		return fmt.Errorf("%w: unknown action %q", ErrInvalidEvent, ev.Action)
	case !ev.Outcome.Valid():
		return fmt.Errorf("%w: unknown outcome %q", ErrInvalidEvent, ev.Outcome)
	case ev.OccurredAt.IsZero():
		return fmt.Errorf("%w: no time", ErrInvalidEvent)
	}
	switch ev.Actor.Type {
	case ActorPortalUser:
		if ev.Actor.ID == ([16]byte{}) {
			return fmt.Errorf("%w: portal user actor without an ID", ErrInvalidEvent)
		}
	case ActorAnonymous, ActorSystem:
		if ev.Actor.ID != ([16]byte{}) {
			return fmt.Errorf("%w: %s actor with an ID", ErrInvalidEvent, ev.Actor.Type)
		}
	default:
		return fmt.Errorf("%w: unknown actor type %q", ErrInvalidEvent, ev.Actor.Type)
	}
	if t := ev.Target; t != nil {
		if (t.Type != TargetPortalUser && t.Type != TargetPortalSession && t.Type != TargetAPIKey) || t.ID == ([16]byte{}) {
			return fmt.Errorf("%w: target %q", ErrInvalidEvent, t.Type)
		}
	}
	return nil
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func optionalAddr(ip netip.Addr) *netip.Addr {
	if !ip.IsValid() {
		return nil
	}
	ip = ip.Unmap().WithZone("")
	return &ip
}
