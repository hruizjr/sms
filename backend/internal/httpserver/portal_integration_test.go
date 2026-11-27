//go:build integration

package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hruizjr/sms/backend/internal/apikey"
	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/dbtest"
	"github.com/hruizjr/sms/backend/internal/messaging"
	"github.com/hruizjr/sms/backend/internal/messaging/provider/simulator"
	"github.com/hruizjr/sms/backend/internal/portal"
	"github.com/hruizjr/sms/backend/internal/portalapi"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

const portalTestPassword = "portal test password"

// portalClient is a browser stand-in: it keeps the session cookie and the CSRF token.
type portalClient struct {
	t      *testing.T
	h      http.Handler
	cookie *http.Cookie
	csrf   string
}

func (c *portalClient) do(method, target, body string, header map[string]string) (int, http.Header, []byte) {
	c.t.Helper()
	req := newRequest(c.t, method, target, body)
	if c.cookie != nil {
		req.AddCookie(c.cookie)
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec, b := serve(c.h, req)
	return rec.Code, rec.Header(), b
}

// send makes a state-changing request with the CSRF token.
func (c *portalClient) send(method, target, body string, header map[string]string) (int, http.Header, []byte) {
	c.t.Helper()
	h := map[string]string{"X-CSRF-Token": c.csrf}
	for k, v := range header {
		h[k] = v
	}
	return c.do(method, target, body, h)
}

func (c *portalClient) signIn(email, password string) (int, []byte) {
	c.t.Helper()
	code, hdr, b := c.do(http.MethodPost, "/portal/v1/session", `{"email":"`+email+`","password":"`+password+`"}`, nil)
	if code == http.StatusOK {
		resp := http.Response{Header: hdr}
		for _, ck := range resp.Cookies() {
			if ck.Name == PortalSessionCookie {
				c.cookie = sessionCookie(ck.Value)
			}
		}
		var s portalapi.Session
		if err := json.Unmarshal(b, &s); err != nil {
			c.t.Fatal(err)
		}
		c.csrf = s.CsrfToken
	}
	return code, b
}

// sessionCookie is the cookie a browser sends back (its attributes are not sent; they are set to
// the server's values for clarity).
func sessionCookie(value string) *http.Cookie {
	return &http.Cookie{Name: PortalSessionCookie, Value: value, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode}
}

func problemType(t *testing.T, b []byte) string {
	t.Helper()
	var p portalapi.Problem
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatalf("%v: %s", err, b)
	}
	return strings.TrimPrefix(p.Type, problemTypeBase)
}

// End-to-end test of the client portal API with the real services, a throwaway database (as
// sms_app, row-level security in force), the dispatcher and the simulator. Verified against the
// simulator only.
func TestIntegrationPortalEndToEnd(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	roles := dbtest.NewMigratedDatabase(t)
	db, err := database.Open(ctx, roles.App, "sms-portalapi-test", database.RequireApplicationRole())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)

	sim := simulator.New(simulator.Config{})
	msgs := messaging.NewService(db, sim.Capabilities())
	keys := apikey.NewService(db)
	accounts, err := portal.NewService(db)
	if err != nil {
		t.Fatal(err)
	}
	dispatcher, err := messaging.NewDispatcher(db, sim, messaging.DispatcherConfig{
		PollInterval: time.Millisecond, BatchSize: 10, Concurrency: 2, SubmitTimeout: time.Second, Lease: 5 * time.Second,
		MaxAttempts: 2, RetryBaseDelay: time.Nanosecond, RetryMaxDelay: time.Nanosecond, MaxMessageAge: time.Hour,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	deps, logs := testDeps(true)
	deps.PublicAPI = &PublicAPIDeps{Messages: msgs, Auth: keys}
	deps.Portal = &PortalDeps{Messages: msgs, Auth: accounts, APIKeys: keys}
	h := NewRouter(deps)

	tenants := tenancy.NewRepository(db.Pool())
	newTenant := func(name, sender string) tenancy.ID {
		tn, err := tenants.Create(ctx, name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tenants.Activate(ctx, tn.ID); err != nil {
			t.Fatal(err)
		}
		if err := msgs.ApproveSenderID(ctx, tn.ID, sender); err != nil {
			t.Fatal(err)
		}
		return tn.ID
	}
	newUser := func(tenantID tenancy.ID, email string, role portal.Role) *portalClient {
		if _, err := accounts.CreateUser(ctx, tenantID, email, role, portalTestPassword); err != nil {
			t.Fatal(err)
		}
		return &portalClient{t: t, h: h}
	}
	tenantA := newTenant("Portal Tenant A", "ACOMPANY")
	tenantB := newTenant("Portal Tenant B", "BCOMPANY")
	owner := newUser(tenantA, "owner@a.example.invalid", portal.RoleOwner)
	viewer := newUser(tenantA, "viewer@a.example.invalid", portal.RoleViewer)
	other := newUser(tenantB, "sender@b.example.invalid", portal.RoleSender)

	// 1. Without a session everything but sign-in is 401.
	for _, target := range []string{"/portal/v1/session", "/portal/v1/messages", "/portal/v1/sender-ids"} {
		if code, _, b := owner.do(http.MethodGet, target, "", nil); code != http.StatusUnauthorized || problemType(t, b) != "unauthenticated" {
			t.Errorf("GET %s without a session: %d %s", target, code, b)
		}
	}

	// 2. Sign-in: wrong password and unknown address are the same 401; a form post is refused.
	if code, b := owner.signIn("owner@a.example.invalid", "wrong password!!"); code != http.StatusUnauthorized || problemType(t, b) != "invalid-credentials" {
		t.Errorf("wrong password: %d %s", code, b)
	}
	if code, b := owner.signIn("nobody@a.example.invalid", portalTestPassword); code != http.StatusUnauthorized || problemType(t, b) != "invalid-credentials" {
		t.Errorf("unknown address: %d %s", code, b)
	}
	if code, _, b := owner.do(http.MethodPost, "/portal/v1/session", "", map[string]string{"Content-Type": "application/x-www-form-urlencoded"}); code != http.StatusUnsupportedMediaType {
		t.Errorf("form sign-in: %d %s", code, b)
	}
	if code, _, b := owner.do(http.MethodPost, "/portal/v1/session", `{"email":"owner@a.example.invalid","password":"x","tenant_id":"`+tenantB.String()+`"}`, nil); code != http.StatusBadRequest || problemType(t, b) != "validation-error" {
		t.Errorf("sign-in with a tenant field: %d %s", code, b)
	}

	// 3. Successful sign-in sets a hardened session cookie and returns the session.
	code, hdr, b := owner.do(http.MethodPost, "/portal/v1/session", `{"email":"Owner@A.example.invalid","password":"`+portalTestPassword+`"}`, nil)
	if code != http.StatusOK {
		t.Fatalf("sign-in: %d %s", code, b)
	}
	setCookie := hdr.Get("Set-Cookie")
	for _, attr := range []string{PortalSessionCookie + "=", "Path=/", "HttpOnly", "Secure", "SameSite=Strict", "Max-Age="} {
		if !strings.Contains(setCookie, attr) {
			t.Errorf("Set-Cookie %q lacks %q", setCookie, attr)
		}
	}
	if strings.Contains(strings.ToLower(setCookie), "domain=") || hdr.Get("Cache-Control") != "no-store" {
		t.Errorf("Set-Cookie %q / Cache-Control %q", setCookie, hdr.Get("Cache-Control"))
	}
	if strings.Contains(logs.String(), portalTestPassword) || strings.Contains(logs.String(), "owner@a.example.invalid") {
		t.Error("the password or the email address was logged")
	}
	if code, b := owner.signIn("owner@a.example.invalid", portalTestPassword); code != http.StatusOK {
		t.Fatalf("sign-in: %d %s", code, b)
	}
	code, _, b = owner.do(http.MethodGet, "/portal/v1/session", "", nil)
	var session portalapi.Session
	_ = json.Unmarshal(b, &session)
	if code != http.StatusOK || session.User.Email != "owner@a.example.invalid" || session.User.Role != "owner" ||
		session.Organization.Name != "Portal Tenant A" || len(session.Permissions) != 3 || session.CsrfToken != owner.csrf {
		t.Fatalf("GET session: %d %s", code, b)
	}
	if strings.Contains(string(b), tenantA.String()) {
		t.Error("the session response exposes the tenant ID")
	}

	// 4. Sender IDs and the segment estimate (the shared segmentation function).
	if code, _, b := owner.do(http.MethodGet, "/portal/v1/sender-ids", "", nil); code != http.StatusOK || string(b) != "{\"data\":[\"ACOMPANY\"]}\n" {
		t.Errorf("sender IDs: %d %s", code, b)
	}
	code, _, b = owner.send(http.MethodPost, "/portal/v1/message-estimates", `{"body":"`+strings.Repeat("a", 161)+`"}`, nil)
	var est portalapi.MessageEstimate
	_ = json.Unmarshal(b, &est)
	if code != http.StatusOK || est.Encoding != "gsm7" || est.SegmentCount != 2 || est.Characters != 161 || est.MaxCharacters != 1600 {
		t.Errorf("estimate: %d %s", code, b)
	}
	code, _, b = owner.send(http.MethodPost, "/portal/v1/message-estimates", `{"body":"Salamat po 😊"}`, nil)
	est = portalapi.MessageEstimate{}
	_ = json.Unmarshal(b, &est)
	if code != http.StatusOK || est.Encoding != "ucs2" || est.SegmentCount != 1 || est.Characters != 12 {
		t.Errorf("UCS-2 estimate: %d %s", code, b)
	}

	// 5. CSRF: a state-changing request without or with a wrong token is refused before anything
	// is processed.
	submit := `{"to":"+639000000001","sender_id":"ACOMPANY","body":"Your code is 000000."}`
	wrong := "x" + owner.csrf[1:]
	if owner.csrf[0] == 'x' {
		wrong = "y" + owner.csrf[1:]
	}
	for name, token := range map[string]string{"missing": "", "wrong": wrong, "truncated": owner.csrf[:len(owner.csrf)-1], "other": "abc"} {
		hdrs := map[string]string{"Idempotency-Key": "portal-1"}
		if token != "" {
			hdrs["X-CSRF-Token"] = token
		}
		if code, _, b := owner.do(http.MethodPost, "/portal/v1/messages", submit, hdrs); code != http.StatusForbidden || problemType(t, b) != "csrf-token-invalid" {
			t.Errorf("CSRF %s: %d %s", name, code, b)
		}
	}
	if code, _, b := owner.do(http.MethodDelete, "/portal/v1/session", "", nil); code != http.StatusForbidden {
		t.Errorf("sign-out without CSRF token: %d %s", code, b)
	}

	// 6. Submit through the shared messaging engine; a repeated submission replays; a different
	// body with the same key is refused.
	if code, _, b := owner.send(http.MethodPost, "/portal/v1/messages", submit, nil); code != http.StatusBadRequest || problemType(t, b) != "idempotency-key-missing" {
		t.Errorf("no Idempotency-Key: %d %s", code, b)
	}
	code, hdr, b = owner.send(http.MethodPost, "/portal/v1/messages", submit, map[string]string{"Idempotency-Key": "portal-1"})
	var created portalapi.Message
	_ = json.Unmarshal(b, &created)
	if code != http.StatusAccepted || created.Status != "accepted" || created.To != "+639000000001" || hdr.Get("Idempotent-Replayed") != "" {
		t.Fatalf("submit: %d %s", code, b)
	}
	code, hdr, b = owner.send(http.MethodPost, "/portal/v1/messages", submit, map[string]string{"Idempotency-Key": "portal-1"})
	var replay portalapi.Message
	_ = json.Unmarshal(b, &replay)
	if code != http.StatusAccepted || hdr.Get("Idempotent-Replayed") != "true" || replay.Id != created.Id {
		t.Errorf("repeated submit: %d %q %s", code, hdr.Get("Idempotent-Replayed"), b)
	}
	if code, _, b := owner.send(http.MethodPost, "/portal/v1/messages", strings.Replace(submit, "000000", "111111", 1), map[string]string{"Idempotency-Key": "portal-1"}); code != http.StatusUnprocessableEntity || problemType(t, b) != "idempotency-key-reused" {
		t.Errorf("same key, different body: %d %s", code, b)
	}
	for name, c := range map[string]struct {
		body, want string
		code       int
	}{
		"unapproved sender":   {`{"to":"+639000000001","sender_id":"BCOMPANY","body":"x"}`, "sender-id-not-approved", 422},
		"foreign destination": {`{"to":"+14155550100","sender_id":"ACOMPANY","body":"x"}`, "destination-not-supported", 422},
		"invalid number":      {`{"to":"0917","sender_id":"ACOMPANY","body":"x"}`, "validation-error", 400},
		"client_reference":    {`{"to":"+639000000001","sender_id":"ACOMPANY","body":"x","client_reference":"r"}`, "validation-error", 400},
		"tenant field":        {`{"to":"+639000000001","sender_id":"ACOMPANY","body":"x","tenant_id":"` + tenantB.String() + `"}`, "validation-error", 400},
	} {
		code, _, b := owner.send(http.MethodPost, "/portal/v1/messages", c.body, map[string]string{"Idempotency-Key": "portal-" + strings.ReplaceAll(name, " ", "-")})
		if code != c.code || problemType(t, b) != c.want {
			t.Errorf("%s: %d %s, want %d %s", name, code, b, c.code, c.want)
		}
	}

	// 7. The dispatcher and the simulator process the message like any other; the portal shows the
	// result and the customer-visible history.
	if _, err := dispatcher.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := sim.DeliverPending(ctx, messaging.NewInboxSink(db)); err != nil {
		t.Fatal(err)
	}
	if _, err := dispatcher.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	code, _, b = owner.do(http.MethodGet, "/portal/v1/messages/"+created.Id, "", nil)
	var detail portalapi.MessageDetail
	_ = json.Unmarshal(b, &detail)
	var history []string
	for _, h := range detail.StatusHistory {
		history = append(history, string(h.Status))
	}
	if code != http.StatusOK || detail.Status != "delivered" || detail.StatusCategory != "final" ||
		strings.Join(history, ",") != "accepted,provider_accepted,delivered" {
		t.Errorf("detail: %d %s (history %v)", code, b, history)
	}
	for _, internal := range []string{"sim-", "SIM_", "submitting", "queued", "Your code"} {
		if strings.Contains(string(b), internal) {
			t.Errorf("message detail exposes internal or content data %q: %s", internal, b)
		}
	}
	code, _, b = owner.do(http.MethodGet, "/portal/v1/messages?limit=10", "", nil)
	var list portalapi.MessageList
	_ = json.Unmarshal(b, &list)
	if code != http.StatusOK || len(list.Data) != 1 || list.Data[0].Id != created.Id || list.NextCursor != nil {
		t.Errorf("list: %d %s", code, b)
	}
	if code, _, b := owner.do(http.MethodGet, "/portal/v1/messages?status=delivered", "", nil); code != http.StatusOK || !strings.Contains(string(b), created.Id) {
		t.Errorf("list ?status=delivered: %d %s", code, b)
	}
	if code, _, b := owner.do(http.MethodGet, "/portal/v1/messages?status=bogus", "", nil); code != http.StatusBadRequest {
		t.Errorf("list ?status=bogus: %d %s", code, b)
	}

	// 8. The viewer reads but cannot send.
	if code, b := viewer.signIn("viewer@a.example.invalid", portalTestPassword); code != http.StatusOK {
		t.Fatalf("viewer sign-in: %d %s", code, b)
	}
	if code, _, b := viewer.do(http.MethodGet, "/portal/v1/messages/"+created.Id, "", nil); code != http.StatusOK {
		t.Errorf("viewer reads: %d %s", code, b)
	}
	for _, target := range []string{"/portal/v1/messages", "/portal/v1/message-estimates"} {
		if code, _, b := viewer.send(http.MethodPost, target, submit, map[string]string{"Idempotency-Key": "viewer-1"}); code != http.StatusForbidden || problemType(t, b) != "forbidden" {
			t.Errorf("viewer POST %s: %d %s", target, code, b)
		}
	}

	// 9. Tenant isolation: tenant B's user sees none of tenant A's messages and cannot read one
	// by its ID; tenant B's submissions land in tenant B.
	if code, b := other.signIn("sender@b.example.invalid", portalTestPassword); code != http.StatusOK {
		t.Fatalf("tenant B sign-in: %d %s", code, b)
	}
	if code, _, b := other.do(http.MethodGet, "/portal/v1/messages/"+created.Id, "", nil); code != http.StatusNotFound {
		t.Errorf("tenant B reads tenant A's message: %d %s", code, b)
	}
	if code, _, b := other.do(http.MethodGet, "/portal/v1/messages", "", nil); code != http.StatusOK || string(b) != "{\"data\":[],\"next_cursor\":null}\n" {
		t.Errorf("tenant B list: %d %s", code, b)
	}
	if code, _, b := other.send(http.MethodPost, "/portal/v1/messages", submit, map[string]string{"Idempotency-Key": "portal-1"}); code != http.StatusUnprocessableEntity || problemType(t, b) != "sender-id-not-approved" {
		t.Errorf("tenant B using tenant A's sender ID: %d %s", code, b)
	}
	code, _, b = other.send(http.MethodPost, "/portal/v1/messages", strings.Replace(submit, "ACOMPANY", "BCOMPANY", 1), map[string]string{"Idempotency-Key": "portal-1"})
	var bMsg portalapi.Message
	_ = json.Unmarshal(b, &bMsg)
	if code != http.StatusAccepted || bMsg.Id == created.Id {
		t.Errorf("tenant B submit with the same key as tenant A: %d %s", code, b)
	}
	if m, err := msgs.Get(ctx, tenantB, mustParseMessageID(t, bMsg.Id)); err != nil || m.SenderID != "BCOMPANY" {
		t.Errorf("tenant B's message is not tenant B's: %+v, %v", m, err)
	}
	if _, err := msgs.Get(ctx, tenantA, mustParseMessageID(t, bMsg.Id)); err == nil {
		t.Error("tenant B's message is visible to tenant A")
	}
	// A tenant B session cookie re-addressed to tenant A does not authenticate.
	parts := strings.Split(other.cookie.Value, ".")
	forged := &portalClient{t: t, h: h, cookie: sessionCookie(strings.Join([]string{owner.cookie.Value[:strings.Index(owner.cookie.Value, ".")], parts[1], parts[2]}, "."))}
	if code, _, b := forged.do(http.MethodGet, "/portal/v1/messages", "", nil); code != http.StatusUnauthorized {
		t.Errorf("forged cookie: %d %s", code, b)
	}

	// 10. Portal sessions and API keys are separate.
	_, apiToken, err := keys.Create(ctx, tenantA, "test", []apikey.Scope{apikey.ScopeMessagesRead})
	if err != nil {
		t.Fatal(err)
	}
	if code, _, b := owner.do(http.MethodGet, "/portal/v1/messages", "", map[string]string{"Authorization": "Bearer " + apiToken}); code != http.StatusOK {
		t.Errorf("portal with cookie and a stray bearer header: %d %s", code, b)
	}
	noCookie := &portalClient{t: t, h: h}
	if code, _, b := noCookie.do(http.MethodGet, "/portal/v1/messages", "", map[string]string{"Authorization": "Bearer " + apiToken}); code != http.StatusUnauthorized {
		t.Errorf("API key on the portal API: %d %s", code, b)
	}
	if code, _, b := owner.do(http.MethodGet, "/v1/messages", "", nil); code != http.StatusUnauthorized {
		t.Errorf("portal session on the public API: %d %s", code, b)
	}

	// 11. Sign-out revokes the session on the server; the old cookie no longer works.
	code, hdr, _ = owner.send(http.MethodDelete, "/portal/v1/session", "", nil)
	if code != http.StatusNoContent || !strings.Contains(hdr.Get("Set-Cookie"), "Max-Age=0") {
		t.Errorf("sign-out: %d %q", code, hdr.Get("Set-Cookie"))
	}
	if code, _, b := owner.do(http.MethodGet, "/portal/v1/session", "", nil); code != http.StatusUnauthorized {
		t.Errorf("after sign-out: %d %s", code, b)
	}

	// 12. Suspending the tenant ends its users' access at the next request.
	if _, err := tenants.Suspend(ctx, tenantA); err != nil {
		t.Fatal(err)
	}
	if code, _, b := viewer.do(http.MethodGet, "/portal/v1/messages", "", nil); code != http.StatusForbidden || problemType(t, b) != "account-not-active" {
		t.Errorf("suspended tenant: %d %s", code, b)
	}
	if code, b := viewer.signIn("viewer@a.example.invalid", portalTestPassword); code != http.StatusForbidden || problemType(t, b) != "account-not-active" {
		t.Errorf("suspended tenant sign-in: %d %s", code, b)
	}
}

func mustParseMessageID(t *testing.T, s string) messaging.MessageID {
	t.Helper()
	id, err := messaging.ParseMessageID(s)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
