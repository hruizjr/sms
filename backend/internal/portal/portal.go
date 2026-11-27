// Package portal implements client portal accounts and their server-side sessions (Phase 4 Group 3;
// owner decisions G3-1 to G3-6, decision log §20). It is separate from the public API's key
// authentication (package apikey): a portal session never grants API access and an API key never
// opens a portal session.
//
// # Sign-in
//
// A user signs in with an email address and a password. The platform-owned login directory maps the
// normalized email to the user's tenant (G3-1); everything else — the Argon2id password hash (G3-2),
// the role, the lockout state and the sessions — lives in tenant-owned tables that are read only inside
// the tenant's transaction-local context, under forced row-level security. Unknown addresses, wrong
// passwords and locked accounts all return ErrInvalidCredentials after the same amount of hashing
// work, so a caller cannot tell them apart. Five consecutive failures lock the account for 15 minutes
// (G3-3).
//
// # Sessions
//
// A session token has the form
//
//	<tenant>.<session id>.<secret>
//
// (lower-case Crockford base32; the secret is 32 random bytes). It is sent only in an HttpOnly,
// Secure, SameSite=Strict cookie. Only the SHA-256 hash of the secret is stored. Because the token
// names its tenant, authentication sets the tenant's context first and looks the session up under
// row-level security, like API keys. A session ends at the first of: 30 minutes without a request
// (idle timeout), 12 hours after sign-in (absolute lifetime), sign-out, or revocation (G3-3).
//
// # CSRF
//
// The CSRF token of a session is derived from its secret (SHA-256 with a separate label), so it is
// not stored and cannot be computed from the stored hash. The portal reads it from the session
// response (same-origin JavaScript only) and sends it in the X-CSRF-Token header of every
// state-changing request; the server recomputes it from the cookie (synchronizer token pattern).
package portal

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/sqlcgen"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

// Role is a client portal role (MVP-03).
type Role string

// Roles.
const (
	RoleOwner  Role = "owner"
	RoleSender Role = "sender"
	RoleViewer Role = "viewer"
)

// Valid reports whether r is a known role.
func (r Role) Valid() bool { return r == RoleOwner || r == RoleSender || r == RoleViewer }

// Permission is something a portal user may do. The portal API checks it on every request.
type Permission string

// Permissions of this group (G3-3): Owner and Sender may read and send; Viewer may only read. The
// Owner's management functions (users, API keys) are not part of this group.
const (
	PermissionMessagesRead Permission = "messages:read"
	PermissionMessagesSend Permission = "messages:send"
)

// Permissions returns the permissions of r, in a fixed order.
func (r Role) Permissions() []Permission {
	switch r {
	case RoleOwner, RoleSender:
		return []Permission{PermissionMessagesRead, PermissionMessagesSend}
	case RoleViewer:
		return []Permission{PermissionMessagesRead}
	}
	return nil
}

// Policy is the account and session policy (owner decision G3-3).
type Policy struct {
	IdleTimeout  time.Duration // a session ends after this long without a request
	MaxLifetime  time.Duration // a session ends this long after sign-in, whatever its activity
	MaxFailures  int           // consecutive failed sign-ins that lock the account
	LockDuration time.Duration // how long a locked account stays locked
}

// DefaultPolicy is the owner-approved policy (G3-3).
var DefaultPolicy = Policy{
	IdleTimeout:  30 * time.Minute,
	MaxLifetime:  12 * time.Hour,
	MaxFailures:  5,
	LockDuration: 15 * time.Minute,
}

// touchInterval limits how often a session's last-seen time is written: at most once a minute, so
// the idle timeout is enforced to within a minute without a database write on every request.
const touchInterval = time.Minute

// Errors.
var (
	// ErrInvalidCredentials: unknown email, wrong password, or locked account (HTTP 401). The cause
	// is not distinguished.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUnauthenticated: no valid session — malformed, unknown, expired or revoked (HTTP 401).
	ErrUnauthenticated = errors.New("unauthenticated")
	// ErrTenantNotActive: the credentials or session are valid but the tenant is not active (HTTP 403).
	ErrTenantNotActive = errors.New("tenant is not active")
	// ErrInvalidEmail: the email address is not acceptable.
	ErrInvalidEmail = errors.New("invalid email address")
	// ErrInvalidRole: unknown role.
	ErrInvalidRole = errors.New("invalid role")
	// ErrEmailTaken: a portal user with this email address already exists.
	ErrEmailTaken = errors.New("a portal user with this email address already exists")
	// ErrUserNotFound: no such portal user of the tenant.
	ErrUserNotFound = errors.New("portal user not found")
)

const (
	secretBytes = 32
	csrfLabel   = "sms-portal-csrf-v1\x00"
)

var encoding = base32.NewEncoding("0123456789abcdefghjkmnpqrstvwxyz").WithPadding(base32.NoPadding)

// User is a portal user (never its password hash).
type User struct {
	ID          [16]byte
	TenantID    tenancy.ID
	Email       string
	Role        Role
	LockedUntil *time.Time
	LastLoginAt *time.Time
	CreatedAt   time.Time
}

// Principal is an authenticated portal session.
type Principal struct {
	TenantID   tenancy.ID
	TenantName string
	UserID     [16]byte
	SessionID  [16]byte
	Email      string
	Role       Role
	// CSRFToken is the session's CSRF token (see the package documentation).
	CSRFToken string
	// ExpiresAt is the end of the absolute lifetime; IdleExpiresAt is when the session ends without
	// further requests.
	ExpiresAt     time.Time
	IdleExpiresAt time.Time
}

// Has reports whether the principal's role has permission.
func (p Principal) Has(permission Permission) bool {
	return slices.Contains(p.Role.Permissions(), permission)
}

// CheckCSRF reports whether token is the session's CSRF token (constant time).
func (p Principal) CheckCSRF(token string) bool {
	return token != "" && subtle.ConstantTimeCompare([]byte(token), []byte(p.CSRFToken)) == 1
}

// Session is a new session returned by Login: the cookie value (shown to the browser only) and the
// authenticated principal.
type Session struct {
	Token     string
	Principal Principal
}

// Service manages portal users and sessions. It is safe for concurrent use.
type Service struct {
	db        *database.DB
	policy    Policy
	now       func() time.Time
	hasher    *hasher
	dummyHash string
}

// Option configures a Service.
type Option func(*Service)

// WithClock sets the clock (tests).
func WithClock(now func() time.Time) Option { return func(s *Service) { s.now = now } }

// WithPolicy replaces DefaultPolicy (tests).
func WithPolicy(p Policy) Option { return func(s *Service) { s.policy = p } }

// NewService returns a Service using db (connected as the application role).
func NewService(db *database.DB, opts ...Option) (*Service, error) {
	s := &Service{db: db, policy: DefaultPolicy, now: time.Now, hasher: newHasher(4)}
	for _, o := range opts {
		o(s)
	}
	// A hash of a random password, verified for unknown addresses and locked accounts so that
	// every failed sign-in costs the same hashing work.
	random := make([]byte, 24)
	if _, err := rand.Read(random); err != nil {
		return nil, fmt.Errorf("portal: %w", err)
	}
	dummy, err := s.hasher.hash(context.Background(), encoding.EncodeToString(random))
	if err != nil {
		return nil, fmt.Errorf("portal: %w", err)
	}
	s.dummyHash = dummy
	return s, nil
}

// NormalizeEmail trims and lower-cases an email address and checks its form: ASCII only, one "@",
// a non-empty local part and domain, no spaces or control characters, at most 254 characters.
func NormalizeEmail(email string) (string, error) {
	e := strings.ToLower(strings.TrimSpace(email))
	if len(e) < 3 || len(e) > 254 {
		return "", ErrInvalidEmail
	}
	for i := 0; i < len(e); i++ {
		if c := e[i]; c <= ' ' || c >= 0x7f {
			return "", ErrInvalidEmail
		}
	}
	local, domain, ok := strings.Cut(e, "@")
	if !ok || local == "" || domain == "" || strings.Contains(domain, "@") || len(local) > 64 {
		return "", ErrInvalidEmail
	}
	return e, nil
}

// CreateUser creates a portal user of tenantID. It is used by the local administration tool only:
// there is no self-registration and no user management endpoint in this group (G3-4).
func (s *Service) CreateUser(ctx context.Context, tenantID tenancy.ID, email string, role Role, password string) (User, error) {
	normalized, err := NormalizeEmail(email)
	if err != nil {
		return User{}, err
	}
	if !role.Valid() {
		return User{}, fmt.Errorf("%w: %q", ErrInvalidRole, role)
	}
	if err := validatePassword(password); err != nil {
		return User{}, err
	}
	hash, err := s.hasher.hash(ctx, password)
	if err != nil {
		return User{}, err
	}
	var u User
	err = s.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		row, err := q.CreatePortalUser(ctx, sqlcgen.CreatePortalUserParams{
			TenantID: pgUUID(tenantID), Email: normalized, Role: string(role), PasswordHash: hash,
		})
		if err != nil {
			return err
		}
		if err := q.InsertLoginDirectoryEntry(ctx, sqlcgen.InsertLoginDirectoryEntryParams{
			Email: normalized, TenantID: row.TenantID, UserID: row.ID,
		}); err != nil {
			return err
		}
		u = User{ID: row.ID.Bytes, TenantID: tenantID, Email: row.Email, Role: Role(row.Role), CreatedAt: row.CreatedAt}
		return nil
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return User{}, ErrEmailTaken
	}
	if err != nil {
		return User{}, fmt.Errorf("create portal user: %w", err)
	}
	return u, nil
}

// ListUsers returns the portal users of tenantID.
func (s *Service) ListUsers(ctx context.Context, tenantID tenancy.ID) ([]User, error) {
	var users []User
	err := s.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := sqlcgen.New(tx).ListPortalUsers(ctx, pgUUID(tenantID))
		for _, r := range rows {
			users = append(users, User{
				ID: r.ID.Bytes, TenantID: tenantID, Email: r.Email, Role: Role(r.Role),
				LockedUntil: r.LockedUntil, LastLoginAt: r.LastLoginAt, CreatedAt: r.CreatedAt,
			})
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list portal users: %w", err)
	}
	return users, nil
}

// userByEmail returns the user of tenantID with email, or ErrUserNotFound.
func (s *Service) userByEmail(ctx context.Context, q *sqlcgen.Queries, tenantID tenancy.ID, email string) (pgtype.UUID, error) {
	normalized, err := NormalizeEmail(email)
	if err != nil {
		return pgtype.UUID{}, ErrUserNotFound
	}
	row, err := q.GetPortalUserByEmail(ctx, sqlcgen.GetPortalUserByEmailParams{TenantID: pgUUID(tenantID), Email: normalized})
	if errors.Is(err, pgx.ErrNoRows) {
		return pgtype.UUID{}, ErrUserNotFound
	}
	return row.ID, err
}

// RevokeUserSessions ends every session of the user of tenantID with email and returns how many
// were not yet revoked (including sessions that had already expired).
func (s *Service) RevokeUserSessions(ctx context.Context, tenantID tenancy.ID, email string) (int64, error) {
	var n int64
	err := s.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		id, err := s.userByEmail(ctx, q, tenantID, email)
		if err != nil {
			return err
		}
		n, err = q.RevokeUserPortalSessions(ctx, sqlcgen.RevokeUserPortalSessionsParams{TenantID: pgUUID(tenantID), UserID: id, Now: s.now()})
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("revoke portal sessions: %w", err)
	}
	return n, nil
}

// Unlock clears the lockout of the user of tenantID with email.
func (s *Service) Unlock(ctx context.Context, tenantID tenancy.ID, email string) error {
	err := s.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		id, err := s.userByEmail(ctx, q, tenantID, email)
		if err != nil {
			return err
		}
		_, err = q.UnlockPortalUser(ctx, sqlcgen.UnlockPortalUserParams{TenantID: pgUUID(tenantID), ID: id, Now: s.now()})
		return err
	})
	if err != nil {
		return fmt.Errorf("unlock portal user: %w", err)
	}
	return nil
}

// Login verifies an email address and password and, if they are valid and the tenant is active,
// starts a session. See the package documentation for the error semantics.
func (s *Service) Login(ctx context.Context, email, password string) (Session, error) {
	normalized, emailErr := NormalizeEmail(email)
	if emailErr != nil || len(password) > MaxPasswordBytes {
		return Session{}, s.fail(ctx, password)
	}
	entry, err := sqlcgen.New(s.db.Pool()).LookupLoginDirectory(ctx, normalized)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, s.fail(ctx, password)
	}
	if err != nil {
		return Session{}, fmt.Errorf("portal login: %w", err)
	}
	tenantID, userID := tenancy.ID(entry.TenantID.Bytes), entry.UserID

	var cred sqlcgen.GetPortalUserCredentialRow
	var tenant tenancy.Tenant
	err = s.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		cred, err = sqlcgen.New(tx).GetPortalUserCredential(ctx, sqlcgen.GetPortalUserCredentialParams{TenantID: entry.TenantID, ID: userID})
		if err != nil {
			return err
		}
		tenant, err = tenancy.NewRepository(tx).Get(ctx, tenantID)
		return err
	})
	if err != nil {
		return Session{}, fmt.Errorf("portal login: %w", err)
	}

	now := s.now()
	if cred.LockedUntil != nil && cred.LockedUntil.After(now) {
		// Locked: spend the same work, record nothing (attempts while locked do not extend the lock).
		return Session{}, s.fail(ctx, password)
	}
	ok, err := s.hasher.verify(ctx, password, cred.PasswordHash)
	if err != nil {
		return Session{}, fmt.Errorf("portal login: verify password: %w", err)
	}
	if !ok {
		err := s.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
			_, err := sqlcgen.New(tx).RecordFailedLogin(ctx, sqlcgen.RecordFailedLoginParams{
				Now: now, MaxFailures: int32(s.policy.MaxFailures), LockUntil: now.Add(s.policy.LockDuration), //nolint:gosec // small configured value
				TenantID: entry.TenantID, ID: userID,
			})
			return err
		})
		if err != nil {
			return Session{}, fmt.Errorf("portal login: record failure: %w", err)
		}
		return Session{}, ErrInvalidCredentials
	}
	if tenant.Status != tenancy.StatusActive {
		return Session{}, ErrTenantNotActive
	}

	secret := make([]byte, secretBytes)
	if _, err := rand.Read(secret); err != nil {
		return Session{}, fmt.Errorf("portal login: generate session secret: %w", err)
	}
	hash := sha256.Sum256(secret)
	var sessionID pgtype.UUID
	err = s.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		if err := q.RecordSuccessfulLogin(ctx, sqlcgen.RecordSuccessfulLoginParams{Now: now, TenantID: entry.TenantID, ID: userID}); err != nil {
			return err
		}
		var err error
		sessionID, err = q.CreatePortalSession(ctx, sqlcgen.CreatePortalSessionParams{
			TenantID: entry.TenantID, UserID: userID, SecretHash: hash[:], Now: now, ExpiresAt: now.Add(s.policy.MaxLifetime),
		})
		return err
	})
	if err != nil {
		return Session{}, fmt.Errorf("portal login: create session: %w", err)
	}
	return Session{
		Token: encoding.EncodeToString(tenantID[:]) + "." + encoding.EncodeToString(sessionID.Bytes[:]) + "." + encoding.EncodeToString(secret),
		Principal: Principal{
			TenantID: tenantID, TenantName: tenant.DisplayName, UserID: userID.Bytes, SessionID: sessionID.Bytes,
			Email: cred.Email, Role: Role(cred.Role), CSRFToken: csrfToken(secret),
			ExpiresAt: now.Add(s.policy.MaxLifetime), IdleExpiresAt: minTime(now.Add(s.policy.IdleTimeout), now.Add(s.policy.MaxLifetime)),
		},
	}, nil
}

// fail verifies password against the dummy hash (equal work) and returns ErrInvalidCredentials.
func (s *Service) fail(ctx context.Context, password string) error {
	if len(password) > MaxPasswordBytes {
		password = password[:MaxPasswordBytes]
	}
	if _, err := s.hasher.verify(ctx, password, s.dummyHash); err != nil {
		return fmt.Errorf("portal login: %w", err)
	}
	return ErrInvalidCredentials
}

// Authenticate returns the principal of a session token. It returns ErrUnauthenticated for any
// malformed, unknown, expired or revoked session and ErrTenantNotActive if the tenant is not active.
// It records the activity that keeps the session alive.
func (s *Service) Authenticate(ctx context.Context, token string) (Principal, error) {
	tenantID, sessionID, secret, ok := parseToken(token)
	if !ok {
		return Principal{}, ErrUnauthenticated
	}
	presented := sha256.Sum256(secret)
	now := s.now()

	var p Principal
	var active bool
	err := s.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		row, err := q.GetPortalSession(ctx, sqlcgen.GetPortalSessionParams{TenantID: pgUUID(tenantID), ID: pgUUID(sessionID)})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUnauthenticated
		}
		if err != nil {
			return err
		}
		if subtle.ConstantTimeCompare(row.SecretHash, presented[:]) != 1 || row.RevokedAt != nil ||
			!now.Before(row.ExpiresAt) || !now.Before(row.LastSeenAt.Add(s.policy.IdleTimeout)) {
			return ErrUnauthenticated
		}
		tenant, err := tenancy.NewRepository(tx).Get(ctx, tenantID)
		if err != nil {
			return err
		}
		active = tenant.Status == tenancy.StatusActive
		lastSeen := row.LastSeenAt
		if active && now.Sub(row.LastSeenAt) >= touchInterval {
			if err := q.TouchPortalSession(ctx, sqlcgen.TouchPortalSessionParams{Now: now, TenantID: row.TenantID, ID: row.ID}); err != nil {
				return err
			}
			lastSeen = now
		}
		p = Principal{
			TenantID: tenantID, TenantName: tenant.DisplayName, UserID: row.UserID.Bytes, SessionID: sessionID,
			Email: row.Email, Role: Role(row.Role), CSRFToken: csrfToken(secret),
			ExpiresAt: row.ExpiresAt, IdleExpiresAt: minTime(lastSeen.Add(s.policy.IdleTimeout), row.ExpiresAt),
		}
		return nil
	})
	switch {
	case errors.Is(err, ErrUnauthenticated), errors.Is(err, database.ErrInvalidTenantID):
		return Principal{}, ErrUnauthenticated
	case err != nil:
		return Principal{}, fmt.Errorf("authenticate portal session: %w", err)
	case !active:
		return Principal{}, ErrTenantNotActive
	}
	return p, nil
}

// Logout revokes the principal's session. Revoking an already revoked session is not an error.
func (s *Service) Logout(ctx context.Context, p Principal) error {
	err := s.db.InTenantTx(ctx, p.TenantID, func(tx pgx.Tx) error {
		_, err := sqlcgen.New(tx).RevokePortalSession(ctx, sqlcgen.RevokePortalSessionParams{
			Now: s.now(), TenantID: pgUUID(p.TenantID), ID: pgUUID(p.SessionID),
		})
		return err
	})
	if err != nil {
		return fmt.Errorf("portal logout: %w", err)
	}
	return nil
}

func csrfToken(secret []byte) string {
	sum := sha256.Sum256(append([]byte(csrfLabel), secret...))
	return encoding.EncodeToString(sum[:])
}

// parseToken splits a session token; ok is false for anything malformed or non-canonical.
func parseToken(token string) (tenantID tenancy.ID, sessionID [16]byte, secret []byte, ok bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return tenantID, sessionID, nil, false
	}
	t, err1 := decodeCanonical(parts[0])
	sid, err2 := decodeCanonical(parts[1])
	sec, err3 := decodeCanonical(parts[2])
	if err1 != nil || err2 != nil || err3 != nil || len(t) != 16 || len(sid) != 16 || len(sec) != secretBytes {
		return tenantID, sessionID, nil, false
	}
	copy(tenantID[:], t)
	copy(sessionID[:], sid)
	return tenantID, sessionID, sec, true
}

func decodeCanonical(s string) ([]byte, error) {
	b, err := encoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if encoding.EncodeToString(b) != s {
		return nil, errors.New("non-canonical encoding")
	}
	return b, nil
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func pgUUID(b [16]byte) pgtype.UUID { return pgtype.UUID{Bytes: b, Valid: true} }
