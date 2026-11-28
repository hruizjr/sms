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
	// attempts while it is locked are recorded but start no new lock.
	for range 6 {
		if err := f.svc.Reauthenticate(f.ctx, s.Principal, "wrong password!!", audit.ActionAPIKeyCreate); !errors.Is(err, ErrReauthenticationFailed) {
			t.Fatalf("Reauthenticate() = %v", err)
		}
	}
	if _, err := f.svc.Login(f.ctx, "owner@accounts.example.invalid", testPassword); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() while locked = %v", err)
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
		"portal.sign_in success -",
		failed, failed, failed, failed, failed,
		"portal.account_locked success too_many_attempts",
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
