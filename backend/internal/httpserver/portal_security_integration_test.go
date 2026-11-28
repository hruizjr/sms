//go:build integration

package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hruizjr/sms/backend/internal/apikey"
	"github.com/hruizjr/sms/backend/internal/audit"
	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/dbtest"
	"github.com/hruizjr/sms/backend/internal/messaging"
	"github.com/hruizjr/sms/backend/internal/messaging/provider/simulator"
	"github.com/hruizjr/sms/backend/internal/portal"
	"github.com/hruizjr/sms/backend/internal/portalapi"
	"github.com/hruizjr/sms/backend/internal/signinlimit"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

// securityEnv is a throwaway database with the real services, as sms_app, and an administrative
// connection for checks outside row-level security.
type securityEnv struct {
	ctx      context.Context
	roles    dbtest.Roles
	db       *database.DB
	admin    *database.DB
	msgs     *messaging.Service
	keys     *apikey.Service
	accounts *portal.Service
	tenants  *tenancy.Repository
}

func newSecurityEnv(t *testing.T) securityEnv {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)
	roles := dbtest.NewMigratedDatabase(t)
	db := openPool(t, ctx, roles.App, database.RequireApplicationRole())
	accounts, err := portal.NewService(db)
	if err != nil {
		t.Fatal(err)
	}
	return securityEnv{
		ctx: ctx, roles: roles, db: db, admin: openPool(t, ctx, roles.Admin),
		msgs: messaging.NewService(db, simulator.New(simulator.Config{}).Capabilities()), keys: apikey.NewService(db),
		accounts: accounts, tenants: tenancy.NewRepository(db.Pool()),
	}
}

func openPool(t *testing.T, ctx context.Context, cfg config.DatabaseConfig, opts ...database.Option) *database.DB {
	t.Helper()
	db, err := database.Open(ctx, cfg, "sms-portal-security-test", opts...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	return db
}

// router builds one API instance: its own connection pool and limiter on the shared database.
func (e securityEnv) router(t *testing.T, limiter *signinlimit.Limiter, trusted ...netip.Prefix) http.Handler {
	t.Helper()
	deps, _ := testDeps(true)
	deps.TrustedProxies = trusted
	deps.PublicAPI = &PublicAPIDeps{Messages: e.msgs, Auth: e.keys}
	deps.Portal = &PortalDeps{Messages: e.msgs, Auth: e.accounts, APIKeys: e.keys, Audit: audit.NewService(e.db), SignInLimiter: limiter}
	return NewRouter(deps)
}

func (e securityEnv) limiter(t *testing.T, cfg signinlimit.Config, now func() time.Time) *signinlimit.Limiter {
	t.Helper()
	// A separate pool per limiter, like a separate API process.
	l, err := signinlimit.New(openPool(t, e.ctx, e.roles.App, database.RequireApplicationRole()), cfg, signinlimit.WithClock(now))
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func (e securityEnv) tenant(t *testing.T, name string) tenancy.ID {
	t.Helper()
	tn, err := e.tenants.Create(e.ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.tenants.Activate(e.ctx, tn.ID); err != nil {
		t.Fatal(err)
	}
	return tn.ID
}

func (e securityEnv) count(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := e.admin.Pool().QueryRow(e.ctx, sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func signInBody(email, password string) string {
	return `{"email":"` + email + `","password":"` + password + `"}`
}

// The per-source sign-in limit end to end: the same and different sources, two API instances and a
// restart sharing one count, 429 with Retry-After, spoofed X-Forwarded-For with and without trusted
// proxies, concurrent requests, the end of the window, and the account lockout (G3-3), which is
// unchanged and independent.
func TestIntegrationPortalSignInRateLimit(t *testing.T) {
	t.Parallel()
	e := newSecurityEnv(t)
	var mu sync.Mutex
	now := time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)
	clock := func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	cfg := signinlimit.Config{MaxAttempts: 6, Window: 10 * time.Minute}
	proxies := netip.MustParsePrefix("10.0.0.0/8")
	instance1 := e.router(t, e.limiter(t, cfg, clock), proxies) // behind a trusted proxy network
	instance2 := e.router(t, e.limiter(t, cfg, clock))          // no trusted proxy

	tenantID := e.tenant(t, "Rate Limit Co")
	const email = "owner@ratelimit.example.invalid"
	if _, err := e.accounts.CreateUser(e.ctx, tenantID, email, portal.RoleOwner, portalTestPassword); err != nil {
		t.Fatal(err)
	}
	attacker := "203.0.113.10:40000"
	try := func(h http.Handler, remote, password string, header map[string]string) (int, http.Header, []byte) {
		c := &portalClient{t: t, h: h, remote: remote}
		return c.do(http.MethodPost, "/portal/v1/session", signInBody(email, password), header)
	}
	lockState := func() string {
		var s string
		if err := e.admin.Pool().QueryRow(e.ctx, `SELECT failed_login_count || ' ' || coalesce(locked_until::text, 'unlocked') FROM portal_users`).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}

	// 1. Six attempts from one source, alternating between the instances: the account lockout works
	// exactly as before (five failures lock it; a locked account refuses even the right password).
	for i := range 5 {
		h := []http.Handler{instance1, instance2}[i%2]
		if code, _, b := try(h, attacker, "wrong password!!", nil); code != http.StatusUnauthorized || problemType(t, b) != "invalid-credentials" {
			t.Fatalf("failed attempt %d: %d %s", i+1, code, b)
		}
	}
	locked := lockState()
	if !strings.HasPrefix(locked, "5 ") || strings.HasSuffix(locked, "unlocked") {
		t.Fatalf("after five failures the account state is %q, want locked with count 5", locked)
	}
	if code, _, b := try(instance2, attacker, portalTestPassword, nil); code != http.StatusUnauthorized {
		t.Fatalf("6th attempt (locked account, right password): %d %s", code, b)
	}

	// 2. The 7th attempt from that source is refused with 429 and Retry-After, before any credential
	// check: the account state does not change and nothing reveals whether the account exists.
	mu.Lock()
	now = now.Add(4 * time.Minute)
	mu.Unlock()
	code, hdr, b := try(instance1, attacker, portalTestPassword, nil)
	if code != http.StatusTooManyRequests || problemType(t, b) != "too-many-sign-in-attempts" || hdr.Get("Retry-After") != "360" {
		t.Fatalf("7th attempt: %d Retry-After=%q %s; want 429, Retry-After 360", code, hdr.Get("Retry-After"), b)
	}
	if hdr.Get("Set-Cookie") != "" || strings.Contains(string(b), email) {
		t.Errorf("429 response sets a cookie or names the account: %v %s", hdr, b)
	}
	if code, _, b := (&portalClient{t: t, h: instance2, remote: attacker}).do(http.MethodPost, "/portal/v1/session",
		signInBody("nobody@ratelimit.example.invalid", "x"), nil); code != http.StatusTooManyRequests {
		t.Errorf("unknown account from the limited source: %d %s, want the same 429", code, b)
	}
	if got := lockState(); got != locked {
		t.Errorf("a rate-limited attempt changed the account state: %q -> %q", locked, got)
	}

	// 3. A restart (a new instance with a new pool) does not reset the count.
	restarted := e.router(t, e.limiter(t, cfg, clock))
	if code, _, _ := try(restarted, attacker, portalTestPassword, nil); code != http.StatusTooManyRequests {
		t.Errorf("after a restart: %d, want 429", code)
	}

	// 4. Spoofed X-Forwarded-For: ignored from a peer that is not a trusted proxy; behind the trusted
	// proxy the address the proxy appended is the client, whatever the client prepended.
	spoof := map[string]string{"X-Forwarded-For": "198.51.100.99", "X-Real-IP": "198.51.100.98"}
	if code, _, _ := try(instance2, attacker, portalTestPassword, spoof); code != http.StatusTooManyRequests {
		t.Errorf("spoofed header to an instance without trusted proxies: %d, want 429", code)
	}
	if code, _, _ := try(instance1, attacker, portalTestPassword, spoof); code != http.StatusTooManyRequests {
		t.Errorf("spoofed header from an untrusted peer: %d, want 429", code)
	}
	behindProxy := map[string]string{"X-Forwarded-For": "198.51.100.99, 203.0.113.10"}
	if code, _, _ := try(instance1, "10.0.0.5:443", portalTestPassword, behindProxy); code != http.StatusTooManyRequests {
		t.Errorf("limited client behind the trusted proxy with a prepended address: %d, want 429", code)
	}

	// 5. Other sources are not affected by this limit (the account itself is still locked: 401).
	for _, c := range []struct {
		h      http.Handler
		remote string
		header map[string]string
	}{
		{instance2, "198.51.100.20:1", nil},
		{instance1, "10.0.0.5:443", map[string]string{"X-Forwarded-For": "192.0.2.200"}},
	} {
		if code, _, b := try(c.h, c.remote, portalTestPassword, c.header); code != http.StatusUnauthorized || problemType(t, b) != "invalid-credentials" {
			t.Errorf("another source (%s %v): %d %s, want 401 (account locked)", c.remote, c.header, code, b)
		}
	}

	// 6. One platform audit event for the window, however many refusals; none in the tenant's log,
	// and the tenant's log has no event for the refused attempts.
	if n := e.count(t, `SELECT count(*) FROM platform_security_events WHERE action = 'portal.sign_in_rate_limited' AND source_ip = '203.0.113.10'`); n != 1 {
		t.Errorf("%d rate-limit audit events, want 1 for the window", n)
	}
	tenantEvents := e.count(t, `SELECT count(*) FROM security_audit_events`)

	// 7. When the window has ended the source may try again (the account lock is a separate matter).
	mu.Lock()
	now = now.Add(6 * time.Minute)
	mu.Unlock()
	if code, _, b := try(instance2, attacker, portalTestPassword, nil); code != http.StatusUnauthorized {
		t.Errorf("after the window: %d %s, want 401 (no longer rate limited; account still locked)", code, b)
	}
	if n := e.count(t, `SELECT count(*) FROM security_audit_events`); n != tenantEvents+1 {
		t.Errorf("tenant events: %d, want %d (one refused sign-in of the locked account)", n, tenantEvents+1)
	}

	// 8. Concurrent requests from one new source on both instances: exactly MaxAttempts get past the
	// limit.
	var passed atomic.Int32
	var wg sync.WaitGroup
	for i := range 14 {
		wg.Go(func() {
			c := &portalClient{t: t, h: []http.Handler{instance1, instance2}[i%2], remote: "192.0.2.77:9"}
			code, _, _ := c.do(http.MethodPost, "/portal/v1/session", signInBody("nobody@ratelimit.example.invalid", "x"), nil)
			if code != http.StatusTooManyRequests {
				passed.Add(1)
			}
		})
	}
	wg.Wait()
	if int(passed.Load()) != cfg.MaxAttempts {
		t.Errorf("14 concurrent attempts: %d passed the limit, want %d", passed.Load(), cfg.MaxAttempts)
	}
}

type auditPage struct {
	Data       []portalapi.AuditEvent `json:"data"`
	NextCursor *string                `json:"next_cursor"`
}

func listAudit(t *testing.T, c *portalClient, query string) (int, auditPage, []byte) {
	t.Helper()
	code, _, b := c.do(http.MethodGet, "/portal/v1/audit-events"+query, "", nil)
	var page auditPage
	if code == http.StatusOK {
		if err := json.Unmarshal(b, &page); err != nil {
			t.Fatal(err)
		}
	}
	return code, page, b
}

// The audit log end to end: sign-ins, failures, lockouts, sign-outs, API key creation (including a
// failed password confirmation) and revocation, refused requests of other roles; owners only; tenant
// isolation; filters and pages; no secret in any event; API key last use.
func TestIntegrationPortalAuditLog(t *testing.T) {
	t.Parallel()
	e := newSecurityEnv(t)
	h := e.router(t, e.limiter(t, signinlimit.Config{MaxAttempts: 1000, Window: time.Hour}, time.Now))
	tenantA, tenantB := e.tenant(t, "Audit Co A"), e.tenant(t, "Audit Co B")
	if err := e.msgs.ApproveSenderID(e.ctx, tenantA, "AUDITCO"); err != nil {
		t.Fatal(err)
	}
	user := func(tenantID tenancy.ID, email string, role portal.Role) {
		if _, err := e.accounts.CreateUser(e.ctx, tenantID, email, role, portalTestPassword); err != nil {
			t.Fatal(err)
		}
	}
	user(tenantA, "owner@audit-a.example.invalid", portal.RoleOwner)
	user(tenantA, "sender@audit-a.example.invalid", portal.RoleSender)
	user(tenantA, "viewer@audit-a.example.invalid", portal.RoleViewer)
	user(tenantB, "owner@audit-b.example.invalid", portal.RoleOwner)
	browser := map[string]string{"User-Agent": "Mozilla/5.0 (Audit Test)\r\n", "X-Request-ID": "audit-req-1"}

	signIn := func(email string) *portalClient {
		c := &portalClient{t: t, h: h, remote: "203.0.113.50:5000"}
		code, hdr, b := c.do(http.MethodPost, "/portal/v1/session", signInBody(email, portalTestPassword), browser)
		if code != http.StatusOK {
			t.Fatalf("sign-in %s: %d %s", email, code, b)
		}
		c.cookie = sessionCookie((&http.Response{Header: hdr}).Cookies()[0].Value)
		var s portalapi.Session
		if err := json.Unmarshal(b, &s); err != nil {
			t.Fatal(err)
		}
		c.csrf = s.CsrfToken
		return c
	}
	anonymous := &portalClient{t: t, h: h, remote: "198.51.100.7:1"}
	// A failed sign-in of a known user, then a sign-in for an address that is not registered.
	if code, _, _ := anonymous.do(http.MethodPost, "/portal/v1/session", signInBody("owner@audit-a.example.invalid", "wrong password!!"), nil); code != http.StatusUnauthorized {
		t.Fatal(code)
	}
	if code, _, _ := anonymous.do(http.MethodPost, "/portal/v1/session", signInBody("ghost@audit-a.example.invalid", "wrong password!!"), nil); code != http.StatusUnauthorized {
		t.Fatal(code)
	}
	owner := signIn("owner@audit-a.example.invalid")
	sender := signIn("sender@audit-a.example.invalid")
	viewer := signIn("viewer@audit-a.example.invalid")
	ownerB := signIn("owner@audit-b.example.invalid")

	// API keys: a failed password confirmation, a created key (used on the public API), a read-only
	// key refused for sending, and a revocation.
	create := func(name, scopes, password string) (int, portalapi.ApiKeyCreated) {
		code, _, b := owner.send(http.MethodPost, "/portal/v1/api-keys",
			`{"name":"`+name+`","scopes":`+scopes+`,"current_password":"`+password+`"}`, nil)
		var created portalapi.ApiKeyCreated
		_ = json.Unmarshal(b, &created)
		return code, created
	}
	if code, _ := create("Never created", `["messages:send"]`, "wrong password!!"); code != http.StatusForbidden {
		t.Fatalf("create with a wrong password: %d", code)
	}
	code, sendKey := create("Order notifications", `["messages:send","messages:read"]`, portalTestPassword)
	if code != http.StatusCreated {
		t.Fatalf("create: %d", code)
	}
	code, readKey := create("Reporting", `["messages:read"]`, portalTestPassword)
	if code != http.StatusCreated {
		t.Fatalf("create read-only: %d", code)
	}
	public := func(token, method, body string) int {
		req := newRequest(t, method, "/v1/messages", body)
		req.Header.Set("Authorization", "Bearer "+token)
		if method == http.MethodPost {
			req.Header.Set("Idempotency-Key", "audit-last-used-1")
		}
		rec, _ := serve(h, req)
		return rec.Code
	}
	if c := public(sendKey.Key, http.MethodGet, ""); c != http.StatusOK {
		t.Fatalf("public API with the new key: %d", c)
	}
	if c := public(readKey.Key, http.MethodPost, `{"to":"+639000000001","sender_id":"AUDITCO","body":"x"}`); c != http.StatusForbidden {
		t.Fatalf("read-only key sending: %d, want 403", c)
	}
	code, _, b := owner.do(http.MethodGet, "/portal/v1/api-keys", "", nil)
	var keyList portalapi.ApiKeyList
	if err := json.Unmarshal(b, &keyList); code != http.StatusOK || err != nil {
		t.Fatalf("list keys: %d %v", code, err)
	}
	for _, k := range keyList.Data {
		switch k.Id {
		case sendKey.ApiKey.Id:
			if k.LastUsedAt == nil || time.Since(*k.LastUsedAt) > time.Minute {
				t.Errorf("used key: last_used_at = %v, want about now", k.LastUsedAt)
			}
		case readKey.ApiKey.Id:
			if k.LastUsedAt != nil {
				t.Errorf("key refused for missing permission: last_used_at = %v, want null", k.LastUsedAt)
			}
		}
	}
	if !strings.Contains(string(b), `"last_used_at":null`) {
		t.Errorf("never-used keys are not reported as null: %s", b)
	}
	if code, _, b := owner.send(http.MethodDelete, "/portal/v1/api-keys/"+readKey.ApiKey.Id, "", nil); code != http.StatusOK {
		t.Fatalf("revoke: %d %s", code, b)
	}

	// Refused requests of other roles, including hand-made requests and URLs.
	for _, c := range []struct {
		client         *portalClient
		method, target string
		body           string
	}{
		{sender, http.MethodGet, "/portal/v1/api-keys", ""},
		{viewer, http.MethodPost, "/portal/v1/api-keys", `{"name":"x","scopes":["messages:read"],"current_password":"` + portalTestPassword + `"}`},
		{sender, http.MethodDelete, "/portal/v1/api-keys/" + sendKey.ApiKey.Id, ""},
		{sender, http.MethodGet, "/portal/v1/audit-events", ""},
		{viewer, http.MethodGet, "/portal/v1/audit-events?limit=100&action=api_key.create", ""},
	} {
		if code, _, b := c.client.send(c.method, c.target, c.body, nil); code != http.StatusForbidden || problemType(t, b) != "forbidden" {
			t.Errorf("%s %s: %d %s, want 403 forbidden", c.method, c.target, code, b)
		}
	}
	// A sender locks their account with five wrong passwords.
	for range 5 {
		anonymous.do(http.MethodPost, "/portal/v1/session", signInBody("sender@audit-a.example.invalid", "wrong password!!"), nil)
	}
	if code, _, _ := viewer.send(http.MethodDelete, "/portal/v1/session", "", nil); code != http.StatusNoContent {
		t.Fatalf("viewer sign-out: %d", code)
	}

	// The owner's view of the log, newest first.
	code, page, raw := listAudit(t, owner, "?limit=100")
	if code != http.StatusOK || page.NextCursor != nil {
		t.Fatalf("owner lists the audit log: %d %s", code, raw)
	}
	type summary struct{ action, outcome, actor, target, reason string }
	var got []summary
	for i := len(page.Data) - 1; i >= 0; i-- { // oldest first, for readability
		ev := page.Data[i]
		s := summary{action: string(ev.Action), outcome: string(ev.Outcome), actor: string(ev.Actor.Type)}
		if ev.Actor.Email != nil {
			s.actor += " " + *ev.Actor.Email
		}
		if ev.Target != nil {
			s.target = string(ev.Target.Type)
			if ev.Target.Email != nil {
				s.target += " " + *ev.Target.Email
			}
		}
		if ev.Details.Reason != nil {
			s.reason = *ev.Details.Reason
		}
		got = append(got, s)
	}
	const (
		o = "owner@audit-a.example.invalid"
		s = "sender@audit-a.example.invalid"
		v = "viewer@audit-a.example.invalid"
	)
	failedSender := summary{"portal.sign_in", "failure", "anonymous", "portal_user " + s, "invalid_password"}
	want := []summary{
		{"portal.sign_in", "failure", "anonymous", "portal_user " + o, "invalid_password"},
		{"portal.sign_in", "success", "portal_user " + o, "portal_session", ""},
		{"portal.sign_in", "success", "portal_user " + s, "portal_session", ""},
		{"portal.sign_in", "success", "portal_user " + v, "portal_session", ""},
		{"api_key.create", "failure", "portal_user " + o, "portal_user " + o, "invalid_password"},
		{"api_key.create", "success", "portal_user " + o, "api_key", ""},
		{"api_key.create", "success", "portal_user " + o, "api_key", ""},
		{"api_key.revoke", "success", "portal_user " + o, "api_key", ""},
		{"api_key.list", "denied", "portal_user " + s, "", "insufficient_role"},
		{"api_key.create", "denied", "portal_user " + v, "", "insufficient_role"},
		{"api_key.revoke", "denied", "portal_user " + s, "", "insufficient_role"},
		{"audit_log.list", "denied", "portal_user " + s, "", "insufficient_role"},
		{"audit_log.list", "denied", "portal_user " + v, "", "insufficient_role"},
		failedSender, failedSender, failedSender, failedSender, failedSender,
		{"portal.account_locked", "success", "system", "portal_user " + s, "too_many_attempts"},
		{"portal.sign_out", "success", "portal_user " + v, "portal_session", ""},
	}
	// The lock is recorded in the transaction of the fifth failure, right after its sign-in event
	// (same time; the later UUIDv7 sorts after it).
	if len(got) != len(want) {
		t.Fatalf("audit log has %d events, want %d:\n%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d = %+v, want %+v", i+1, got[i], want[i])
		}
	}

	// Event details: request source, key names and permissions, the lock end; nothing secret.
	for _, ev := range page.Data {
		switch {
		case ev.Action == "portal.sign_in" && ev.Outcome == "success" && ev.Actor.Email != nil && *ev.Actor.Email == o:
			if ev.RequestId == nil || *ev.RequestId != "audit-req-1" || ev.SourceIp == nil || *ev.SourceIp != "203.0.113.50" ||
				ev.UserAgent == nil || *ev.UserAgent != "Mozilla/5.0 (Audit Test)" {
				t.Errorf("sign-in source: request %v ip %v agent %v", ev.RequestId, ev.SourceIp, ev.UserAgent)
			}
		case ev.Action == "api_key.create" && ev.Outcome == "success" && *ev.Details.ApiKeyName == "Order notifications":
			if ev.Target.Id == nil || *ev.Target.Id != sendKey.ApiKey.Id || len(*ev.Details.ApiKeyScopes) != 2 {
				t.Errorf("key creation event: %+v", ev)
			}
		case ev.Action == "portal.account_locked":
			if ev.Details.LockedUntil == nil || time.Until(*ev.Details.LockedUntil) < 14*time.Minute {
				t.Errorf("lock event: locked until %v", ev.Details.LockedUntil)
			}
		case ev.Outcome == "denied":
			if ev.Details.Role == nil || ev.Details.RequiredPermission == nil {
				t.Errorf("refusal without role or permission: %+v", ev.Details)
			}
		}
	}
	for _, secret := range []string{portalTestPassword, "wrong password!!", sendKey.Key, readKey.Key, "smsk_",
		owner.cookie.Value, owner.csrf, sender.cookie.Value, "ghost@"} {
		if strings.Contains(string(raw), secret) {
			t.Errorf("the audit log response contains %q", secret)
		}
	}
	// (The reason code invalid_password names a cause, not a value, and is excluded.)
	if n := e.count(t, `SELECT count(*) FROM security_audit_events
		WHERE (details - 'reason')::text ~* '(password|secret|hash|token|smsk_|cookie|authorization)'
		   OR details->>'reason' NOT IN ('invalid_password', 'account_locked', 'organization_not_active', 'insufficient_role', 'too_many_attempts')`); n != 0 {
		t.Errorf("%d stored events have credential-like details", n)
	}
	// The unknown address is a platform event, never in an organization's log, and not stored.
	if n := e.count(t, `SELECT count(*) FROM platform_security_events WHERE action = 'portal.sign_in' AND details->>'reason' = 'unknown_account'`); n != 1 {
		t.Errorf("platform events for the unknown address: %d, want 1", n)
	}
	if n := e.count(t, `SELECT count(*) FROM platform_security_events WHERE details::text LIKE '%ghost%'`); n != 0 {
		t.Error("the unknown address was stored")
	}

	// Tenant isolation: the other organization's owner sees only their own sign-in.
	code, pageB, rawB := listAudit(t, ownerB, "?limit=100")
	if code != http.StatusOK || len(pageB.Data) != 1 || pageB.Data[0].Actor.Email == nil || *pageB.Data[0].Actor.Email != "owner@audit-b.example.invalid" {
		t.Errorf("tenant B's audit log: %d %s", code, rawB)
	}

	// Filters and pages.
	if code, p, b := listAudit(t, owner, "?action=api_key.create&outcome=failure"); code != http.StatusOK || len(p.Data) != 1 {
		t.Errorf("action+outcome filter: %d %s", code, b)
	}
	from := page.Data[len(page.Data)-1].OccurredAt.Add(time.Microsecond).Format(time.RFC3339Nano)
	if code, p, b := listAudit(t, owner, "?occurred_from="+from); code != http.StatusOK || len(p.Data) != len(want)-1 {
		t.Errorf("occurred_from filter: %d %d events %s", code, len(p.Data), b)
	}
	code, p1, _ := listAudit(t, owner, "?limit=3&outcome=denied")
	if code != http.StatusOK || len(p1.Data) != 3 || p1.NextCursor == nil {
		t.Fatalf("page 1: %d %+v", code, p1)
	}
	code, p2, _ := listAudit(t, owner, "?limit=3&outcome=denied&cursor="+*p1.NextCursor)
	if code != http.StatusOK || len(p2.Data) != 2 || p2.NextCursor != nil || p2.Data[0].Id == p1.Data[2].Id {
		t.Errorf("page 2: %d %+v", code, p2)
	}
	for _, bad := range []string{
		"?action=portal.sign_in_rate_limited", // a platform action is not an organization's
		"?outcome=maybe", "?limit=0", "?limit=101", "?occurred_from=yesterday",
		"?occurred_from=2026-11-28T10:00:00Z&occurred_before=2026-11-28T10:00:00Z",
		"?cursor=" + *p1.NextCursor, // issued for other filters
	} {
		if code, _, b := listAudit(t, owner, bad); code != http.StatusBadRequest {
			t.Errorf("GET audit-events%s: %d %s, want 400", bad, code, b)
		}
	}
	// Read-only: no other method exists.
	for _, method := range []string{http.MethodPost, http.MethodDelete, http.MethodPut, http.MethodPatch} {
		if code, _, _ := owner.send(method, "/portal/v1/audit-events", "", nil); code != http.StatusMethodNotAllowed {
			t.Errorf("%s audit-events: %d, want 405", method, code)
		}
	}
}
