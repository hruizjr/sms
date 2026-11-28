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
// the table owner. The application cannot choose an event's ID or time either: PostgreSQL assigns
// both when the row is inserted, so an event can be neither back-dated nor made to collide with an
// existing one. Database superusers, and the table owner after disabling the triggers, can still alter
// rows: the log is append-only under the application's permissions, not tamper-proof, and evidence
// that holds against a superuser needs an external immutable export (security baseline S-8).
//
// # Deduplication
//
// Some events would otherwise be written once per request, so that a caller could multiply audit
// writes at will: the refusals of a signed-in user's requests and the attempts on a locked account.
// They carry a DedupKey built from server-side values only (DeniedKey, LockedAttemptKey); the database
// writes the first event of a key and silently skips every later one (INSERT ... ON CONFLICT DO
// NOTHING on a unique index). No existing event is modified and no auxiliary record is created.
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
	// ActionPortalUserCreate: the local administration tool created a portal user.
	ActionPortalUserCreate Action = "portal_user.create"
	// ActionPortalUserUnlock: the local administration tool cleared a portal user's lockout.
	ActionPortalUserUnlock Action = "portal_user.unlock"
	// ActionPortalUserSessionsRevoke: the local administration tool ended a portal user's sessions.
	ActionPortalUserSessionsRevoke Action = "portal_user.sessions_revoke"
	// ActionSenderIDApprove: the local administration tool approved a sender ID for the organization.
	ActionSenderIDApprove Action = "sender_id.approve"
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
	ActionPortalUserCreate, ActionPortalUserUnlock, ActionPortalUserSessionsRevoke, ActionSenderIDApprove,
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
	// OutcomeEnforced: the platform applied a security control (an account lock). It is the outcome
	// of ActionAccountLocked, and of nothing else.
	OutcomeEnforced Outcome = "enforced"
)

// Valid reports whether o is a known outcome.
func (o Outcome) Valid() bool {
	return o == OutcomeSuccess || o == OutcomeFailure || o == OutcomeDenied || o == OutcomeEnforced
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
	ReasonTooManyRequests       Reason = "too_many_requests" // platform events only: the all-attempt rate
	ReasonTooManyFailures       Reason = "too_many_failures" // platform events only: a failure budget
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
	// SenderID is the sender ID that was approved (ActionSenderIDApprove; a sender name, not secret).
	SenderID string `json:"sender_id,omitempty"`
	// DedupIntervalSeconds is set on a refusal recorded once per actor, action and interval of this
	// many seconds: repeated refusals in the same interval are not recorded again.
	DedupIntervalSeconds int `json:"dedup_interval_seconds,omitempty"`

	// Platform events only: the sign-in limit that refused (Limit is a failure budget's limit or the
	// token bucket's burst), its window or rate, and the prefix length of the limited source.
	Limit              int     `json:"limit,omitempty"`
	WindowSeconds      int     `json:"window_seconds,omitempty"`
	RatePerSecond      float64 `json:"rate_per_second,omitempty"`
	SourcePrefixLength int     `json:"source_prefix_length,omitempty"`
}

// Event is a tenant security event. Its ID and time are assigned by the database.
type Event struct {
	Action  Action
	Outcome Outcome
	Actor   Actor
	Target  *Target // nil when the action concerns no single resource
	Details Details
	// DedupKey, when set, records the event only if the organization has no event with this key yet
	// (DeniedKey, LockedAttemptKey). Empty: always recorded.
	DedupKey string
}

// PlatformEvent is a security event that belongs to no identifiable organization. Its ID and time are
// assigned by the database.
type PlatformEvent struct {
	Action  Action
	Outcome Outcome
	Details Details
}

// DeniedInterval is the interval within which the refusals of one actor's requests for one action
// are recorded once (owner decision, decision log §23).
const DeniedInterval = 5 * time.Minute

// DeniedKey is the deduplication key of a refused request of actor for action at now: one event per
// actor, action and DeniedInterval (intervals are aligned to the Unix epoch).
func DeniedKey(actor Actor, action Action, now time.Time) string {
	interval := now.UTC().Truncate(DeniedInterval).Unix()
	return fmt.Sprintf("denied:%s:%x:%d", action, actor.ID, interval)
}

// LockedAttemptKey is the deduplication key of an attempt (action) on the locked account userID:
// one event per account, action and lock (identified by the time the lock ends).
func LockedAttemptKey(action Action, userID [16]byte, lockedUntil time.Time) string {
	return fmt.Sprintf("locked:%s:%x:%d", action, userID, lockedUntil.UTC().UnixMicro())
}

// dedupKeyAllowed is the form of a deduplication key (the database checks the same).
func dedupKeyAllowed(k string) bool {
	if len(k) < 1 || len(k) > 160 {
		return false
	}
	return strings.Trim(k, "abcdefghijklmnopqrstuvwxyz0123456789_.:-") == ""
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
// taken from ctx (WithSource). An event whose DedupKey the organization already has is not written,
// and that is not an error.
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
		TenantID:  pgtype.UUID{Bytes: tenantID, Valid: true},
		Action:    string(ev.Action),
		Outcome:   string(ev.Outcome),
		ActorType: string(ev.Actor.Type),
		RequestID: optional(src.RequestID),
		SourceIp:  optionalAddr(src.IP),
		UserAgent: optional(NormalizeUserAgent(src.UserAgent)),
		Details:   details,
		DedupKey:  optional(ev.DedupKey),
	}
	if ev.Actor.Type == ActorPortalUser {
		params.ActorID = pgtype.UUID{Bytes: ev.Actor.ID, Valid: true}
	}
	if ev.Target != nil {
		t := string(ev.Target.Type)
		params.TargetType = &t
		params.TargetID = pgtype.UUID{Bytes: ev.Target.ID, Valid: true}
	}
	if _, err := sqlcgen.New(tx).InsertSecurityAuditEvent(ctx, params); err != nil {
		return fmt.Errorf("record audit event %s: %w", ev.Action, err)
	}
	return nil
}

// InsertPlatform writes a platform event through db (a pool, a connection or a transaction; no tenant
// context is needed or used). The request source is taken from ctx (WithSource).
func InsertPlatform(ctx context.Context, db sqlcgen.DBTX, ev PlatformEvent) error {
	if !slices.Contains(platformActions, ev.Action) || !ev.Outcome.Valid() || ev.Outcome == OutcomeEnforced {
		return fmt.Errorf("%w: platform event %q (%q)", ErrInvalidEvent, ev.Action, ev.Outcome)
	}
	details, err := json.Marshal(ev.Details)
	if err != nil {
		return fmt.Errorf("%w: details: %w", ErrInvalidEvent, err)
	}
	src := SourceFrom(ctx)
	err = sqlcgen.New(db).InsertPlatformSecurityEvent(ctx, sqlcgen.InsertPlatformSecurityEventParams{
		Action:    string(ev.Action),
		Outcome:   string(ev.Outcome),
		RequestID: optional(src.RequestID),
		SourceIp:  optionalAddr(src.IP),
		UserAgent: optional(NormalizeUserAgent(src.UserAgent)),
		Details:   details,
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
	case (ev.Outcome == OutcomeEnforced) != (ev.Action == ActionAccountLocked):
		return fmt.Errorf("%w: outcome %q is not valid for %q", ErrInvalidEvent, ev.Outcome, ev.Action)
	case ev.DedupKey != "" && !dedupKeyAllowed(ev.DedupKey):
		return fmt.Errorf("%w: deduplication key %q", ErrInvalidEvent, ev.DedupKey)
	case (ev.Details.SenderID != "") != (ev.Action == ActionSenderIDApprove):
		return fmt.Errorf("%w: a sender ID belongs to %q events only, and is required there", ErrInvalidEvent, ActionSenderIDApprove)
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
