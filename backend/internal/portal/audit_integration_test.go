//go:build integration

package portal

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hruizjr/sms/backend/internal/audit"
)

// events returns "action outcome reason" for every audit event of the database, oldest first, read
// as the bootstrap role.
func (f fixture) events(t *testing.T, table string) []string {
	t.Helper()
	rows, err := f.admin.Pool().Query(f.ctx, `SELECT action || ' ' || outcome || ' ' || coalesce(details->>'reason', '-') FROM `+
		pgx.Identifier{table}.Sanitize()+` ORDER BY occurred_at, id`)
	if err != nil {
		t.Fatal(err)
	}
	got, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// Sign-in, re-authentication, lock and sign-out events, written by the service with the state they
// describe; a lock is recorded once, by the attempt that starts it.
func TestIntegrationAccountEventsAreAudited(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tenantID := f.activeTenant(t, "Audited Accounts")
	f.user(t, tenantID, "owner@accounts.example.invalid", RoleOwner)

	s, err := f.svc.Login(f.ctx, "owner@accounts.example.invalid", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	// Re-authentication failures count toward the same lockout: the fifth locks the account, and
	// attempts while it is locked start no new lock and are recorded once per action and lock, however
	// often they are repeated.
	for range 25 {
		if err := f.svc.Reauthenticate(f.ctx, s.Principal, "wrong password!!", audit.ActionAPIKeyCreate); !errors.Is(err, ErrReauthenticationFailed) {
			t.Fatalf("Reauthenticate() = %v", err)
		}
	}
	for range 20 {
		if _, err := f.svc.Login(f.ctx, "owner@accounts.example.invalid", testPassword); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("Login() while locked = %v", err)
		}
	}
	// When the lock has expired, a new failure starts a new count without a lock event.
	f.clock.Advance(16 * time.Minute)
	if _, err := f.svc.Login(f.ctx, "owner@accounts.example.invalid", "wrong password!!"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal(err)
	}
	s, err = f.svc.Login(f.ctx, "owner@accounts.example.invalid", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Logout(f.ctx, s.Principal); err != nil {
		t.Fatal(err)
	}
	// An organization that is not active: the right password is refused and recorded as denied.
	if _, err := f.tenants.Suspend(f.ctx, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Login(f.ctx, "owner@accounts.example.invalid", testPassword); !errors.Is(err, ErrTenantNotActive) {
		t.Fatalf("Login() of a suspended organization = %v", err)
	}

	failed := "api_key.create failure invalid_password"
	want := []string{
		"portal_user.create success -",
		"portal.sign_in success -",
		failed, failed, failed, failed, failed,
		"portal.account_locked enforced too_many_attempts",
		"api_key.create failure account_locked",
		"portal.sign_in failure account_locked",
		"portal.sign_in failure invalid_password",
		"portal.sign_in success -",
		"portal.sign_out success -",
		"portal.sign_in denied organization_not_active",
	}
	if got := f.events(t, "security_audit_events"); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("audit events:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	var lockedUntil time.Time
	if err := f.admin.Pool().QueryRow(f.ctx, `SELECT (details->>'locked_until')::timestamptz FROM security_audit_events
		WHERE action = 'portal.account_locked'`).Scan(&lockedUntil); err != nil || !lockedUntil.Equal(time.Date(2026, 11, 27, 8, 15, 0, 0, time.UTC)) {
		t.Errorf("lock event: locked until %v, %v", lockedUntil, err)
	}
	// The deduplicated attempts on the locked account name the lock they belong to.
	var withLock int
	if err := f.admin.Pool().QueryRow(f.ctx, `SELECT count(*) FROM security_audit_events
		WHERE details->>'reason' = 'account_locked' AND (details->>'locked_until')::timestamptz = $1 AND dedup_key LIKE 'locked:%'`,
		lockedUntil).Scan(&withLock); err != nil || withLock != 2 {
		t.Errorf("locked attempts with their lock: %d, %v; want 2", withLock, err)
	}

	// Addresses that are not registered, and input that cannot be an account, are platform events
	// without the address.
	for _, email := range []string{"nobody@accounts.example.invalid", "not an email"} {
		if _, err := f.svc.Login(f.ctx, email, testPassword); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatal(err)
		}
	}
	if _, err := f.svc.Login(f.ctx, "owner@accounts.example.invalid", strings.Repeat("x", MaxPasswordBytes+1)); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal(err)
	}
	if got, want := f.events(t, "platform_security_events"), []string{
		"portal.sign_in failure unknown_account", "portal.sign_in failure invalid_input", "portal.sign_in failure invalid_input",
	}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("platform events = %v, want %v", got, want)
	}
	var stored int
	if err := f.admin.Pool().QueryRow(f.ctx, `SELECT count(*) FROM platform_security_events WHERE details::text LIKE '%@%'`).Scan(&stored); err != nil || stored != 0 {
		t.Errorf("platform events store email addresses: %d, %v", stored, err)
	}
}

// The local administration tool's user creation, unlock and session revocation are recorded as
// system actions, with the tool's name, in the same transaction.
func TestIntegrationAdministrationIsAudited(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tenantID := f.activeTenant(t, "Administered")
	admin, err := NewService(f.db, WithClock(f.clock.Now), WithAdministrationTool("smsctl"))
	if err != nil {
		t.Fatal(err)
	}
	u, err := admin.CreateUser(f.ctx, tenantID, "sender@admin.example.invalid", RoleSender, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.Unlock(f.ctx, tenantID, "sender@admin.example.invalid"); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.RevokeUserSessions(f.ctx, tenantID, "sender@admin.example.invalid"); err != nil {
		t.Fatal(err)
	}
	if err := admin.Unlock(f.ctx, tenantID, "nobody@admin.example.invalid"); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("Unlock(unknown) = %v", err)
	}
	rows, err := f.admin.Pool().Query(f.ctx, `SELECT action || ' ' || outcome || ' ' || actor_type || ' ' || coalesce(details->>'via', '-') || ' ' ||
		coalesce(details->>'role', '-') || ' ' || (target_id = $1)::text FROM security_audit_events WHERE tenant_id = $2 ORDER BY occurred_at, id`,
		pgUUID(u.ID), pgUUID(tenantID))
	if err != nil {
		t.Fatal(err)
	}
	got, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"portal_user.create success system smsctl sender true",
		"portal_user.unlock success system smsctl - true",
		"portal_user.sessions_revoke success system smsctl - true",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("administration events:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	// A user is created only together with its event: with the audit insert impossible, nothing is created.
	if _, err := f.admin.Pool().Exec(f.ctx, `REVOKE INSERT ON security_audit_events FROM sms_app`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.admin.Pool().Exec(f.ctx, `GRANT INSERT (tenant_id, action, outcome, actor_type, actor_id, target_type,
			target_id, request_id, source_ip, user_agent, details, dedup_key) ON security_audit_events TO sms_app`)
	})
	if _, err := admin.CreateUser(f.ctx, tenantID, "unaudited@admin.example.invalid", RoleViewer, testPassword); err == nil {
		t.Error("CreateUser() without its audit event succeeded")
	}
	var n int
	if err := f.admin.Pool().QueryRow(f.ctx, `SELECT count(*) FROM portal_users WHERE email = 'unaudited@admin.example.invalid'`).Scan(&n); err != nil || n != 0 {
		t.Errorf("unaudited user created: %d, %v", n, err)
	}
}

// When the password-hashing capacity is exhausted, Login and Reauthenticate return ErrBusy without
// checking the password: no failure is counted and nothing is recorded.
func TestIntegrationBusyHasherChecksNothing(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tenantID := f.activeTenant(t, "Busy")
	f.user(t, tenantID, "owner@busy.example.invalid", RoleOwner)
	svc, err := NewService(f.db, WithClock(f.clock.Now), WithHashLimits(HashLimits{Concurrency: 1, Queue: 0, MaxWait: time.Second}))
	if err != nil {
		t.Fatal(err)
	}
	s, err := svc.Login(f.ctx, "owner@busy.example.invalid", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	before := len(f.events(t, "security_audit_events"))
	if err := svc.hasher.acquire(f.ctx); err != nil { // the only slot is taken; the queue holds nobody
		t.Fatal(err)
	}
	for range 10 {
		if _, err := svc.Login(f.ctx, "owner@busy.example.invalid", "wrong password!!"); !errors.Is(err, ErrBusy) {
			t.Fatalf("Login() while busy = %v, want ErrBusy", err)
		}
		if _, err := svc.Login(f.ctx, "nobody@busy.example.invalid", "wrong password!!"); !errors.Is(err, ErrBusy) {
			t.Fatalf("Login(unknown) while busy = %v, want ErrBusy", err)
		}
		if err := svc.Reauthenticate(f.ctx, s.Principal, "wrong password!!", audit.ActionAPIKeyCreate); !errors.Is(err, ErrBusy) {
			t.Fatalf("Reauthenticate() while busy = %v, want ErrBusy", err)
		}
	}
	svc.hasher.release()
	if n := len(f.events(t, "security_audit_events")); n != before {
		t.Errorf("%d events recorded while busy", n-before)
	}
	if n := len(f.events(t, "platform_security_events")); n != 0 {
		t.Errorf("%d platform events recorded while busy", n)
	}
	var failures int
	if err := f.admin.Pool().QueryRow(f.ctx, `SELECT failed_login_count FROM portal_users`).Scan(&failures); err != nil || failures != 0 {
		t.Errorf("failed_login_count = %d, %v; want 0", failures, err)
	}
}
