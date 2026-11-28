//go:build integration

package audit_test

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hruizjr/sms/backend/internal/apikey"
	"github.com/hruizjr/sms/backend/internal/audit"
	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/dbtest"
	"github.com/hruizjr/sms/backend/internal/portal"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

const testPassword = "audit test password"

type fixture struct {
	ctx      context.Context
	app      *database.DB
	migrator *database.DB
	admin    *database.DB
	svc      *audit.Service
	tenants  *tenancy.Repository
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	roles := dbtest.NewMigratedDatabase(t)
	openAs := func(cfg config.DatabaseConfig, opts ...database.Option) *database.DB {
		db, err := database.Open(ctx, cfg, "sms-audit-test", opts...)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(db.Close)
		return db
	}
	app := openAs(roles.App, database.RequireApplicationRole())
	return fixture{
		ctx: ctx, app: app, migrator: openAs(roles.Migrator), admin: openAs(roles.Admin),
		svc: audit.NewService(app), tenants: tenancy.NewRepository(app.Pool()),
	}
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

func (f fixture) count(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := f.admin.Pool().QueryRow(f.ctx, sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func uuid(id [16]byte) pgtype.UUID { return pgtype.UUID{Bytes: id, Valid: true} }

func isPermissionDenied(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42501"
}

// Tenant A can neither read nor write tenant B's audit events (forced row-level security, as sms_app).
func TestIntegrationAuditTenantIsolation(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	a := f.activeTenant(t, "Audit A")
	b := f.activeTenant(t, "Audit B")
	now := time.Now()
	ev := audit.Event{OccurredAt: now, Action: audit.ActionSignOut, Outcome: audit.OutcomeSuccess, Actor: audit.System}
	if err := f.svc.Record(f.ctx, a, ev); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Record(f.ctx, b, ev); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		tenant tenancy.ID
		name   string
	}{{a, "A"}, {b, "B"}} {
		entries, _, err := f.svc.List(f.ctx, tc.tenant, audit.Filter{Limit: 100})
		if err != nil || len(entries) != 1 {
			t.Errorf("tenant %s sees %d events (%v), want only its own 1", tc.name, len(entries), err)
		}
	}
	// Under tenant A's context, a query without the tenant filter still sees only A's rows, and an
	// event for B cannot be inserted.
	err := f.app.InTenantTx(f.ctx, a, func(tx pgx.Tx) error {
		var n int
		if err := tx.QueryRow(f.ctx, `SELECT count(*) FROM security_audit_events`).Scan(&n); err != nil || n != 1 {
			t.Errorf("unfiltered count under tenant A = %d, %v; want 1", n, err)
		}
		return audit.Insert(f.ctx, tx, b, ev)
	})
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" { // new row violates row-level security policy
		t.Errorf("insert of tenant B's event under tenant A = %v, want a row-level security violation", err)
	}
	// Without a tenant context nothing is visible.
	var n int
	if err := f.app.Pool().QueryRow(f.ctx, `SELECT count(*) FROM security_audit_events`).Scan(&n); err != nil || n != 0 {
		t.Errorf("count without tenant context = %d, %v; want 0", n, err)
	}
}

// The audit tables are append-only: sms_app may only insert (and read its tenant's events); the table
// owner is stopped by the triggers; platform events cannot be read by the application.
func TestIntegrationAuditAppendOnly(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	a := f.activeTenant(t, "Append Only")
	if err := f.svc.Record(f.ctx, a, audit.Event{OccurredAt: time.Now(), Action: audit.ActionSignOut, Outcome: audit.OutcomeSuccess, Actor: audit.System}); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.RecordPlatform(f.ctx, audit.PlatformEvent{OccurredAt: time.Now(), Action: audit.ActionSignInRateLimited, Outcome: audit.OutcomeDenied}); err != nil {
		t.Fatal(err)
	}

	// As sms_app, inside the tenant's context (where the row is visible): no change is allowed.
	for _, sql := range []string{
		`UPDATE security_audit_events SET outcome = 'failure'`,
		`UPDATE security_audit_events SET details = '{}'`,
		`DELETE FROM security_audit_events`,
		`TRUNCATE security_audit_events`,
		`SELECT count(*) FROM platform_security_events`,
		`UPDATE platform_security_events SET outcome = 'failure'`,
		`DELETE FROM platform_security_events`,
	} {
		err := f.app.InTenantTx(f.ctx, a, func(tx pgx.Tx) error {
			_, err := tx.Exec(f.ctx, sql)
			return err
		})
		if !isPermissionDenied(err) {
			t.Errorf("as sms_app, %q: %v, want permission denied", sql, err)
		}
	}
	// As the table owner (sms_migrator, no BYPASSRLS but subject to the triggers): the row-level
	// triggers reject every UPDATE and DELETE of a row, the statement triggers every TRUNCATE.
	for _, sql := range []string{
		`UPDATE security_audit_events SET outcome = 'failure'`,
		`DELETE FROM security_audit_events`,
		`TRUNCATE security_audit_events`,
		`UPDATE platform_security_events SET outcome = 'failure'`,
		`DELETE FROM platform_security_events`,
		`TRUNCATE platform_security_events`,
	} {
		err := f.migrator.InTenantTx(f.ctx, a, func(tx pgx.Tx) error {
			_, err := tx.Exec(f.ctx, sql)
			return err
		})
		if !isPermissionDenied(err) {
			t.Errorf("as the table owner, %q: %v, want the append-only trigger's error", sql, err)
		}
	}
	if n := f.count(t, `SELECT count(*) FROM security_audit_events`) + f.count(t, `SELECT count(*) FROM platform_security_events`); n != 2 {
		t.Errorf("%d audit rows remain, want 2", n)
	}
	// The limit of this protection, stated in the migration: a superuser can disable the triggers.
	// (Not exercised here: the guarantee is "append-only under the application's permissions".)
}

// List filters by action, outcome and time, pages with a cursor bound to its filters, and returns the
// request source and the users' email addresses.
func TestIntegrationAuditList(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	a := f.activeTenant(t, "Audit List")
	accounts, err := portal.NewService(f.app)
	if err != nil {
		t.Fatal(err)
	}
	user, err := accounts.CreateUser(f.ctx, a, "owner@list.example.invalid", portal.RoleOwner, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 11, 28, 1, 0, 0, 0, time.UTC)
	ctx := audit.WithSource(f.ctx, audit.Source{RequestID: "req-1", IP: netip.MustParseAddr("::ffff:203.0.113.9"), UserAgent: "Agent/1.0\n"})
	events := []audit.Event{
		{OccurredAt: base, Action: audit.ActionSignIn, Outcome: audit.OutcomeFailure, Actor: audit.Anonymous,
			Target: &audit.Target{Type: audit.TargetPortalUser, ID: user.ID}, Details: audit.Details{Reason: audit.ReasonInvalidPassword}},
		{OccurredAt: base.Add(time.Minute), Action: audit.ActionSignIn, Outcome: audit.OutcomeSuccess, Actor: audit.PortalUser(user.ID),
			Target: &audit.Target{Type: audit.TargetPortalSession, ID: [16]byte{9}}},
		{OccurredAt: base.Add(2 * time.Minute), Action: audit.ActionAPIKeyList, Outcome: audit.OutcomeDenied, Actor: audit.PortalUser(user.ID)},
		{OccurredAt: base.Add(3 * time.Minute), Action: audit.ActionSignOut, Outcome: audit.OutcomeSuccess, Actor: audit.PortalUser(user.ID)},
	}
	for _, ev := range events {
		if err := f.svc.Record(ctx, a, ev); err != nil {
			t.Fatal(err)
		}
	}
	list := func(flt audit.Filter) []audit.Entry {
		t.Helper()
		entries, _, err := f.svc.List(f.ctx, a, flt)
		if err != nil {
			t.Fatal(err)
		}
		return entries
	}
	all := list(audit.Filter{Limit: 100})
	if len(all) != 4 || all[0].Action != audit.ActionSignOut || all[3].Action != audit.ActionSignIn {
		t.Fatalf("List() = %+v, want 4 events newest first", all)
	}
	first := all[3]
	if first.RequestID != "req-1" || first.SourceIP != netip.MustParseAddr("203.0.113.9") || first.UserAgent != "Agent/1.0" ||
		first.Actor.Type != "anonymous" || first.Target == nil || first.Target.Email != "owner@list.example.invalid" ||
		first.Details.Reason != audit.ReasonInvalidPassword || !first.OccurredAt.Equal(base) {
		t.Errorf("oldest event = %+v", first)
	}
	if all[2].Actor.Email != "owner@list.example.invalid" || all[2].Target.Type != "portal_session" || all[2].Target.Email != "" {
		t.Errorf("sign-in event = %+v", all[2])
	}

	signIn, success := audit.ActionSignIn, audit.OutcomeSuccess
	if got := list(audit.Filter{Limit: 100, Action: &signIn}); len(got) != 2 {
		t.Errorf("action filter: %d events, want 2", len(got))
	}
	if got := list(audit.Filter{Limit: 100, Action: &signIn, Outcome: &success}); len(got) != 1 || got[0].Outcome != success {
		t.Errorf("action and outcome filter: %+v", got)
	}
	from, before := base.Add(time.Minute), base.Add(3*time.Minute)
	if got := list(audit.Filter{Limit: 100, OccurredFrom: &from, OccurredBefore: &before}); len(got) != 2 {
		t.Errorf("time range [from, before): %d events, want 2", len(got))
	}

	// Pages of two, then a cursor reused with other filters is refused.
	page1, next, err := f.svc.List(f.ctx, a, audit.Filter{Limit: 2})
	if err != nil || len(page1) != 2 || next == "" {
		t.Fatalf("page 1 = %d, %q, %v", len(page1), next, err)
	}
	page2, next2, err := f.svc.List(f.ctx, a, audit.Filter{Limit: 2, Cursor: next})
	if err != nil || len(page2) != 2 || next2 != "" || page2[0].ID == page1[1].ID || page2[1].Action != audit.ActionSignIn {
		t.Errorf("page 2 = %+v, %q, %v", page2, next2, err)
	}
	if _, _, err := f.svc.List(f.ctx, a, audit.Filter{Limit: 2, Cursor: next, Action: &signIn}); !errors.Is(err, audit.ErrInvalidCursor) {
		t.Errorf("cursor with other filters: %v, want ErrInvalidCursor", err)
	}
	if _, _, err := f.svc.List(f.ctx, a, audit.Filter{Limit: 2, Cursor: "not a cursor"}); !errors.Is(err, audit.ErrInvalidCursor) {
		t.Errorf("malformed cursor: %v, want ErrInvalidCursor", err)
	}
}

// When the audit event cannot be written, the security-sensitive change it records does not happen:
// no API key is created or revoked and no session is started. (The failure is simulated by revoking
// the application role's INSERT privilege on the audit table for the duration of the test.)
func TestIntegrationAuditFailurePreventsUnauditedChanges(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	a := f.activeTenant(t, "Audit Failure")
	keys := apikey.NewService(f.app)
	accounts, err := portal.NewService(f.app)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := accounts.CreateUser(f.ctx, a, "owner@failure.example.invalid", portal.RoleOwner, testPassword); err != nil {
		t.Fatal(err)
	}
	existing, _, err := keys.Create(f.ctx, a, "Existing", []apikey.Scope{apikey.ScopeMessagesRead}, audit.System, "")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.admin.Pool().Exec(f.ctx, `REVOKE INSERT ON security_audit_events FROM sms_app`); err != nil {
		t.Fatal(err)
	}
	keysBefore := f.count(t, `SELECT count(*) FROM api_keys`)
	if _, token, err := keys.Create(f.ctx, a, "Unaudited", []apikey.Scope{apikey.ScopeMessagesSend}, audit.System, ""); err == nil || token != "" {
		t.Errorf("Create() without an audit event = %v (token returned: %t), want an error and no key", err, token != "")
	}
	if n := f.count(t, `SELECT count(*) FROM api_keys`); n != keysBefore {
		t.Errorf("api keys: %d, want %d (the key was created without its audit event)", n, keysBefore)
	}
	if _, err := keys.Revoke(f.ctx, a, existing.ID, audit.System, ""); err == nil {
		t.Error("Revoke() without an audit event succeeded")
	}
	if n := f.count(t, `SELECT count(*) FROM api_keys WHERE id = $1 AND revoked_at IS NULL`, uuid(existing.ID)); n != 1 {
		t.Error("the key was revoked without its audit event")
	}
	if _, err := accounts.Login(f.ctx, "owner@failure.example.invalid", testPassword); err == nil || errors.Is(err, portal.ErrInvalidCredentials) {
		t.Errorf("Login() without an audit event = %v, want an internal error", err)
	}
	if n := f.count(t, `SELECT count(*) FROM portal_sessions`); n != 0 {
		t.Errorf("%d sessions were started without their audit event", n)
	}
	// A failed sign-in that cannot be audited is not counted either (the count and its event commit
	// together), and is reported as an error rather than as a plain refusal.
	if _, err := accounts.Login(f.ctx, "owner@failure.example.invalid", "wrong password!!"); err == nil || errors.Is(err, portal.ErrInvalidCredentials) {
		t.Errorf("failed Login() without an audit event = %v, want an internal error", err)
	}
	if n := f.count(t, `SELECT failed_login_count FROM portal_users`); n != 0 {
		t.Errorf("failed_login_count = %d, want 0 (rolled back with the event)", n)
	}

	if _, err := f.admin.Pool().Exec(f.ctx, `GRANT INSERT (tenant_id, occurred_at, action, outcome, actor_type, actor_id, target_type,
		target_id, request_id, source_ip, user_agent, details) ON security_audit_events TO sms_app`); err != nil {
		t.Fatal(err)
	}
	if _, err := accounts.Login(f.ctx, "owner@failure.example.invalid", testPassword); err != nil {
		t.Errorf("Login() after the privilege was restored: %v", err)
	}
}
