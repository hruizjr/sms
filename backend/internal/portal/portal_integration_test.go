//go:build integration

package portal

import (
	"context"
	"crypto/sha256"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/dbtest"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

const testPassword = "a long test password"

// clock is a controllable test clock.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }

func (c *clock) Advance(d time.Duration) { c.mu.Lock(); c.now = c.now.Add(d); c.mu.Unlock() }

type fixture struct {
	ctx     context.Context
	svc     *Service
	clock   *clock
	db      *database.DB
	admin   *database.DB
	tenants *tenancy.Repository
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	roles := dbtest.NewMigratedDatabase(t)
	db, err := database.Open(ctx, roles.App, "sms-portal-test", database.RequireApplicationRole())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	admin, err := database.Open(ctx, roles.Admin, "sms-portal-test-admin")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	c := &clock{now: time.Date(2026, 11, 27, 8, 0, 0, 0, time.UTC)}
	svc, err := NewService(db, WithClock(c.Now))
	if err != nil {
		t.Fatal(err)
	}
	return fixture{ctx: ctx, svc: svc, clock: c, db: db, admin: admin, tenants: tenancy.NewRepository(db.Pool())}
}

func (f fixture) activeTenant(t *testing.T, name string) tenancy.ID {
	t.Helper()
	tn, err := f.tenants.Create(f.ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.tenants.Activate(f.ctx, tn.ID); err != nil {
		t.Fatal(err)
	}
	return tn.ID
}

func (f fixture) user(t *testing.T, tenantID tenancy.ID, email string, role Role) {
	t.Helper()
	if _, err := f.svc.CreateUser(f.ctx, tenantID, email, role, testPassword); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationLoginAuthenticateLogout(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tenantID := f.activeTenant(t, "Portal Co")
	f.user(t, tenantID, "Owner@Example.invalid", RoleOwner)

	// The email is case-insensitive; the session names the tenant and the user.
	s, err := f.svc.Login(f.ctx, "  OWNER@example.INVALID ", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	p := s.Principal
	if p.TenantID != tenantID || p.TenantName != "Portal Co" || p.Email != "owner@example.invalid" || p.Role != RoleOwner ||
		p.CSRFToken == "" || !p.ExpiresAt.Equal(f.clock.Now().Add(12*time.Hour)) || !p.IdleExpiresAt.Equal(f.clock.Now().Add(30*time.Minute)) {
		t.Fatalf("Login() principal = %+v", p)
	}
	got, err := f.svc.Authenticate(f.ctx, s.Token)
	if err != nil || got.SessionID != p.SessionID || got.CSRFToken != p.CSRFToken || got.UserID != p.UserID {
		t.Fatalf("Authenticate() = %+v, %v", got, err)
	}

	// Only hashes are stored: neither the password, the session token, its secret nor the CSRF
	// token appears anywhere in the user or session rows.
	secret := s.Token[strings.LastIndex(s.Token, ".")+1:]
	raw, _ := decodeCanonical(secret)
	want := sha256.Sum256(raw)
	var stored []byte
	var rows string
	if err := f.admin.Pool().QueryRow(f.ctx, `
		SELECT s.secret_hash, row_to_json(s)::text || row_to_json(u)::text
		FROM portal_sessions s JOIN portal_users u ON u.id = s.user_id`).Scan(&stored, &rows); err != nil {
		t.Fatal(err)
	}
	if string(stored) != string(want[:]) {
		t.Error("stored session hash is not the SHA-256 of the secret")
	}
	for _, leaked := range []string{testPassword, secret, s.Token, p.CSRFToken} {
		if strings.Contains(rows, leaked) {
			t.Errorf("a secret value is stored in plain text: %q", leaked)
		}
	}

	// Logout revokes the session on the server; the token no longer authenticates.
	if err := f.svc.Logout(f.ctx, got); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Authenticate(f.ctx, s.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("after logout: %v, want ErrUnauthenticated", err)
	}
	if err := f.svc.Logout(f.ctx, got); err != nil {
		t.Errorf("second logout: %v", err)
	}
}

func TestIntegrationInvalidCredentials(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tenantID := f.activeTenant(t, "Portal Co")
	f.user(t, tenantID, "user@example.invalid", RoleSender)
	for name, c := range map[string][2]string{
		"unknown email":   {"nobody@example.invalid", testPassword},
		"wrong password":  {"user@example.invalid", testPassword + "!"},
		"empty password":  {"user@example.invalid", ""},
		"malformed email": {"not an email", testPassword},
		"huge password":   {"user@example.invalid", strings.Repeat("x", 5000)},
	} {
		if _, err := f.svc.Login(f.ctx, c[0], c[1]); !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("%s: Login() = %v, want ErrInvalidCredentials", name, err)
		}
	}
	// Tampered or unknown tokens.
	s, err := f.svc.Login(f.ctx, "user@example.invalid", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(s.Token, ".")
	otherTenant := f.activeTenant(t, "Other Co")
	other := encoding.EncodeToString(otherTenant[:])
	zeroSecret := encoding.EncodeToString(make([]byte, secretBytes))
	for name, bad := range map[string]string{
		"empty":         "",
		"wrong secret":  parts[0] + "." + parts[1] + "." + zeroSecret,
		"tenant swap":   other + "." + parts[1] + "." + parts[2],
		"nil tenant":    encoding.EncodeToString(make([]byte, 16)) + "." + parts[1] + "." + parts[2],
		"upper case":    strings.ToUpper(s.Token),
		"api key shape": "smsk_" + strings.ReplaceAll(s.Token, ".", "_"),
	} {
		if _, err := f.svc.Authenticate(f.ctx, bad); !errors.Is(err, ErrUnauthenticated) {
			t.Errorf("%s: Authenticate() = %v, want ErrUnauthenticated", name, err)
		}
	}
}

func TestIntegrationLockout(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tenantID := f.activeTenant(t, "Portal Co")
	f.user(t, tenantID, "user@example.invalid", RoleSender)
	fail := func() {
		t.Helper()
		if _, err := f.svc.Login(f.ctx, "user@example.invalid", "wrong password!!"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("wrong password: %v", err)
		}
	}

	// A success resets the count: 4 failures, a success, 4 more failures do not lock.
	for range 4 {
		fail()
	}
	if _, err := f.svc.Login(f.ctx, "user@example.invalid", testPassword); err != nil {
		t.Fatalf("after 4 failures: %v", err)
	}
	for range 4 {
		fail()
	}
	if _, err := f.svc.Login(f.ctx, "user@example.invalid", testPassword); err != nil {
		t.Fatalf("after 4 more failures: %v", err)
	}

	// Five consecutive failures lock the account for 15 minutes, even for the right password.
	for range 5 {
		fail()
	}
	if _, err := f.svc.Login(f.ctx, "user@example.invalid", testPassword); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("locked account with the right password: %v, want ErrInvalidCredentials", err)
	}
	users, _ := f.svc.ListUsers(f.ctx, tenantID)
	if len(users) != 1 || users[0].LockedUntil == nil || !users[0].LockedUntil.Equal(f.clock.Now().Add(15*time.Minute)) {
		t.Fatalf("locked user = %+v", users)
	}
	// Attempts while locked do not extend the lock.
	f.clock.Advance(14 * time.Minute)
	fail()
	if _, err := f.svc.Login(f.ctx, "user@example.invalid", testPassword); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("still locked: %v", err)
	}
	f.clock.Advance(time.Minute)
	if _, err := f.svc.Login(f.ctx, "user@example.invalid", testPassword); err != nil {
		t.Fatalf("after the lock ran out: %v", err)
	}

	// After an expired lock, the count starts again: one failure does not re-lock.
	for range 5 {
		fail()
	}
	f.clock.Advance(16 * time.Minute)
	fail()
	if _, err := f.svc.Login(f.ctx, "user@example.invalid", testPassword); err != nil {
		t.Fatalf("one failure after an expired lock re-locked the account: %v", err)
	}

	// An operator can unlock.
	for range 5 {
		fail()
	}
	if err := f.svc.Unlock(f.ctx, tenantID, "USER@example.invalid"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Login(f.ctx, "user@example.invalid", testPassword); err != nil {
		t.Fatalf("after unlock: %v", err)
	}
}

func TestIntegrationSessionExpiryAndRevocation(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tenantID := f.activeTenant(t, "Portal Co")
	f.user(t, tenantID, "user@example.invalid", RoleViewer)
	login := func() string {
		t.Helper()
		s, err := f.svc.Login(f.ctx, "user@example.invalid", testPassword)
		if err != nil {
			t.Fatal(err)
		}
		return s.Token
	}

	// Idle timeout: 30 minutes without a request ends the session.
	idle := login()
	f.clock.Advance(30 * time.Minute)
	if _, err := f.svc.Authenticate(f.ctx, idle); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("after 30 idle minutes: %v, want ErrUnauthenticated", err)
	}

	// Activity keeps it alive, but never beyond 12 hours after sign-in.
	active := login()
	for range 24 { // 24 × 29 minutes = 11 h 36 min
		f.clock.Advance(29 * time.Minute)
		p, err := f.svc.Authenticate(f.ctx, active)
		if err != nil {
			t.Fatalf("active session ended early: %v", err)
		}
		if p.IdleExpiresAt.After(p.ExpiresAt) {
			t.Fatalf("idle expiry %v is after the absolute expiry %v", p.IdleExpiresAt, p.ExpiresAt)
		}
	}
	f.clock.Advance(24 * time.Minute) // 12 h exactly
	if _, err := f.svc.Authenticate(f.ctx, active); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("after 12 hours: %v, want ErrUnauthenticated", err)
	}

	// Revocation by an operator ends every session of the user, and only that user's.
	f.user(t, tenantID, "other@example.invalid", RoleViewer)
	a, b := login(), login()
	o, err := f.svc.Login(f.ctx, "other@example.invalid", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	// Two new sessions plus the two expired ones above (expired sessions are not revoked until now).
	if n, err := f.svc.RevokeUserSessions(f.ctx, tenantID, "user@example.invalid"); n != 4 || err != nil {
		t.Fatalf("RevokeUserSessions() = %d, %v; want 4", n, err)
	}
	for _, token := range []string{a, b} {
		if _, err := f.svc.Authenticate(f.ctx, token); !errors.Is(err, ErrUnauthenticated) {
			t.Errorf("revoked session: %v", err)
		}
	}
	if _, err := f.svc.Authenticate(f.ctx, o.Token); err != nil {
		t.Errorf("another user's session was revoked: %v", err)
	}
}

func TestIntegrationTenantNotActive(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tenantID := f.activeTenant(t, "Portal Co")
	f.user(t, tenantID, "user@example.invalid", RoleOwner)
	s, err := f.svc.Login(f.ctx, "user@example.invalid", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.tenants.Suspend(f.ctx, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Authenticate(f.ctx, s.Token); !errors.Is(err, ErrTenantNotActive) {
		t.Errorf("suspended tenant session: %v, want ErrTenantNotActive", err)
	}
	if _, err := f.svc.Login(f.ctx, "user@example.invalid", testPassword); !errors.Is(err, ErrTenantNotActive) {
		t.Errorf("suspended tenant login: %v, want ErrTenantNotActive", err)
	}
	// A wrong password still reports invalid credentials (nothing about the tenant is revealed).
	if _, err := f.svc.Login(f.ctx, "user@example.invalid", "wrong password!!"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("suspended tenant, wrong password: %v", err)
	}
	// A pending (never activated) tenant cannot sign in either.
	pending, err := f.tenants.Create(f.ctx, "Pending Co")
	if err != nil {
		t.Fatal(err)
	}
	f.user(t, pending.ID, "pending@example.invalid", RoleOwner)
	if _, err := f.svc.Login(f.ctx, "pending@example.invalid", testPassword); !errors.Is(err, ErrTenantNotActive) {
		t.Errorf("pending tenant login: %v, want ErrTenantNotActive", err)
	}
}

func TestIntegrationCreateUserValidationAndUniqueness(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	a, b := f.activeTenant(t, "Tenant A"), f.activeTenant(t, "Tenant B")
	f.user(t, a, "shared@example.invalid", RoleOwner)
	// An email address is unique across the platform (it identifies the tenant at sign-in).
	for _, tenantID := range []tenancy.ID{a, b} {
		if _, err := f.svc.CreateUser(f.ctx, tenantID, "SHARED@example.invalid", RoleViewer, testPassword); !errors.Is(err, ErrEmailTaken) {
			t.Errorf("duplicate email in tenant %s: %v, want ErrEmailTaken", tenantID, err)
		}
	}
	if _, err := f.svc.CreateUser(f.ctx, a, "x@example.invalid", "admin", testPassword); !errors.Is(err, ErrInvalidRole) {
		t.Errorf("unknown role: %v", err)
	}
	if _, err := f.svc.CreateUser(f.ctx, a, "x@example.invalid", RoleViewer, "short"); !errors.Is(err, ErrWeakPassword) {
		t.Errorf("weak password: %v", err)
	}
	if _, err := f.svc.CreateUser(f.ctx, a, "not-an-email", RoleViewer, testPassword); !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("invalid email: %v", err)
	}
	if _, err := f.svc.CreateUser(f.ctx, tenancy.ID{}, "y@example.invalid", RoleViewer, testPassword); err == nil {
		t.Error("user created for the nil tenant")
	}
}

// Tenant isolation: tenant A's context sees only tenant A's users and sessions, a session of
// tenant A cannot be presented as tenant B's, and the login directory holds nothing but the
// approved columns (G3-1).
func TestIntegrationTenantIsolation(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	a, b := f.activeTenant(t, "Tenant A"), f.activeTenant(t, "Tenant B")
	f.user(t, a, "a@example.invalid", RoleOwner)
	f.user(t, b, "b@example.invalid", RoleOwner)
	sa, err := f.svc.Login(f.ctx, "a@example.invalid", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Login(f.ctx, "b@example.invalid", testPassword); err != nil {
		t.Fatal(err)
	}

	usersA, _ := f.svc.ListUsers(f.ctx, a)
	if len(usersA) != 1 || usersA[0].Email != "a@example.invalid" {
		t.Errorf("tenant A lists %+v", usersA)
	}
	if n, err := f.svc.RevokeUserSessions(f.ctx, a, "b@example.invalid"); !errors.Is(err, ErrUserNotFound) || n != 0 {
		t.Errorf("tenant A revoking tenant B's user: %d, %v", n, err)
	}

	// Row-level security, as the application role, inside tenant A's context.
	err = f.db.InTenantTx(f.ctx, a, func(tx pgx.Tx) error {
		var users, sessions int
		if err := tx.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM portal_users), (SELECT count(*) FROM portal_sessions)`).Scan(&users, &sessions); err != nil {
			return err
		}
		if users != 1 || sessions != 1 {
			t.Errorf("tenant A sees %d users and %d sessions, want 1 and 1", users, sessions)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Inside tenant A's context, a session row for tenant B is refused by the policy's WITH CHECK.
	err = f.db.InTenantTx(f.ctx, a, func(tx pgx.Tx) error {
		_, err := tx.Exec(f.ctx, `INSERT INTO portal_sessions (tenant_id, user_id, secret_hash, created_at, last_seen_at, expires_at)
			SELECT tenant_id, user_id, sha256('x'::bytea), now(), now(), now() + interval '1 hour'
			FROM portal_login_directory WHERE email = 'b@example.invalid'`)
		return err
	})
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		t.Errorf("tenant A inserting a session for tenant B: %v, want a row-level security violation (42501)", err)
	}
	// Without a tenant context nothing is visible.
	var n int
	if err := f.db.Pool().QueryRow(f.ctx, `SELECT count(*) FROM portal_users`).Scan(&n); err != nil || n != 0 {
		t.Errorf("without tenant context: %d users, %v", n, err)
	}

	// A's session token re-addressed to tenant B does not exist there.
	parts := strings.Split(sa.Token, ".")
	if _, err := f.svc.Authenticate(f.ctx, encoding.EncodeToString(b[:])+"."+parts[1]+"."+parts[2]); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("tenant A session presented as tenant B: %v", err)
	}

	// The platform-owned directory has exactly the approved columns.
	var cols []string
	if err := f.admin.Pool().QueryRow(f.ctx, `SELECT array_agg(column_name::text ORDER BY column_name)
		FROM information_schema.columns WHERE table_name = 'portal_login_directory'`).Scan(&cols); err != nil {
		t.Fatal(err)
	}
	if want := []string{"created_at", "email", "tenant_id", "user_id"}; !slices.Equal(cols, want) {
		t.Errorf("portal_login_directory columns = %v, want %v", cols, want)
	}
}
