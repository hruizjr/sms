//go:build integration

package httpserver

import (
	"context"
	"encoding/json"
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

// API key management through the portal, end to end with the real services and a throwaway
// database (as sms_app, row-level security in force): owners create (after re-entering their
// password, G4-7), list and revoke keys; the public API accepts a new key and refuses it once
// revoked; other roles, other tenants, wrong passwords, locked accounts, requests without a session
// and requests without the CSRF token are refused.
func TestIntegrationPortalAPIKeys(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	roles := dbtest.NewMigratedDatabase(t)
	db, err := database.Open(ctx, roles.App, "sms-portal-apikeys-test", database.RequireApplicationRole())
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
	signedIn := func(tenantID tenancy.ID, email string, role portal.Role) *portalClient {
		if _, err := accounts.CreateUser(ctx, tenantID, email, role, portalTestPassword); err != nil {
			t.Fatal(err)
		}
		c := &portalClient{t: t, h: h}
		if code, b := c.signIn(email, portalTestPassword); code != http.StatusOK {
			t.Fatalf("sign-in %s: %d %s", email, code, b)
		}
		return c
	}
	tenantA := newTenant("Key Tenant A", "ACOMPANY")
	tenantB := newTenant("Key Tenant B", "BCOMPANY")
	owner := signedIn(tenantA, "owner@a.example.invalid", portal.RoleOwner)
	sender := signedIn(tenantA, "sender@a.example.invalid", portal.RoleSender)
	viewer := signedIn(tenantA, "viewer@a.example.invalid", portal.RoleViewer)
	ownerB := signedIn(tenantB, "owner@b.example.invalid", portal.RoleOwner)

	withPassword := func(fields string) string {
		return `{` + fields + `,"current_password":"` + portalTestPassword + `"}`
	}
	createBody := withPassword(`"name":"Order notifications","scopes":["messages:send","messages:read"]`)
	existing, _, err := keys.Create(ctx, tenantA, "Existing key", []apikey.Scope{apikey.ScopeMessagesRead})
	if err != nil {
		t.Fatal(err)
	}
	existingPath := "/portal/v1/api-keys/" + existing.IDString()

	// 1. Without a session: 401.
	anonymous := &portalClient{t: t, h: h}
	for _, c := range []struct{ method, target, body string }{
		{http.MethodGet, "/portal/v1/api-keys", ""},
		{http.MethodPost, "/portal/v1/api-keys", createBody},
		{http.MethodDelete, existingPath, ""},
	} {
		if code, _, b := anonymous.do(c.method, c.target, c.body, nil); code != http.StatusUnauthorized || problemType(t, b) != "unauthenticated" {
			t.Errorf("%s %s without a session: %d %s", c.method, c.target, code, b)
		}
	}

	// 2. Senders and viewers may not manage keys (MVP-03: owners only).
	for name, c := range map[string]*portalClient{"sender": sender, "viewer": viewer} {
		if code, _, b := c.do(http.MethodGet, "/portal/v1/api-keys", "", nil); code != http.StatusForbidden || problemType(t, b) != "forbidden" {
			t.Errorf("%s lists keys: %d %s", name, code, b)
		}
		if code, _, b := c.send(http.MethodPost, "/portal/v1/api-keys", createBody, nil); code != http.StatusForbidden || problemType(t, b) != "forbidden" {
			t.Errorf("%s creates a key: %d %s", name, code, b)
		}
		if code, _, b := c.send(http.MethodDelete, existingPath, "", nil); code != http.StatusForbidden || problemType(t, b) != "forbidden" {
			t.Errorf("%s revokes a key: %d %s", name, code, b)
		}
	}

	// 3. State-changing requests need the CSRF token.
	if code, _, b := owner.do(http.MethodPost, "/portal/v1/api-keys", createBody, nil); code != http.StatusForbidden || problemType(t, b) != "csrf-token-invalid" {
		t.Errorf("create without CSRF token: %d %s", code, b)
	}
	if code, _, b := owner.do(http.MethodDelete, existingPath, "", map[string]string{"X-CSRF-Token": sender.csrf}); code != http.StatusForbidden || problemType(t, b) != "csrf-token-invalid" {
		t.Errorf("revoke with another session's CSRF token: %d %s", code, b)
	}

	// 4. Validation.
	for name, body := range map[string]string{
		"missing name":          withPassword(`"scopes":["messages:send"]`),
		"blank name":            withPassword(`"name":"   ","scopes":["messages:send"]`),
		"long name":             withPassword(`"name":"` + strings.Repeat("x", 101) + `","scopes":["messages:send"]`),
		"control character":     withPassword(`"name":"a\u0007b","scopes":["messages:send"]`),
		"name not a string":     withPassword(`"name":7,"scopes":["messages:send"]`),
		"missing scopes":        withPassword(`"name":"k"`),
		"no scopes":             withPassword(`"name":"k","scopes":[]`),
		"unknown scope":         withPassword(`"name":"k","scopes":["messages:delete"]`),
		"duplicate scope":       withPassword(`"name":"k","scopes":["messages:send","messages:send"]`),
		"scopes not strings":    withPassword(`"name":"k","scopes":[1]`),
		"tenant field":          withPassword(`"name":"k","scopes":["messages:send"],"tenant_id":"` + tenantB.String() + `"`),
		"no confirmation field": `{"name":"k","scopes":["messages:send"]}`,
		"empty confirmation":    `{"name":"k","scopes":["messages:send"],"current_password":""}`,
		"confirmation not text": `{"name":"k","scopes":["messages:send"],"current_password":1}`,
		"confirmation too long": `{"name":"k","scopes":["messages:send"],"current_password":"` + strings.Repeat("x", 1025) + `"}`,
		"not an object":         `["k"]`,
	} {
		if code, _, b := owner.send(http.MethodPost, "/portal/v1/api-keys", body, nil); code != http.StatusBadRequest || problemType(t, b) != "validation-error" {
			t.Errorf("%s: %d %s", name, code, b)
		}
	}
	if code, _, b := owner.send(http.MethodPost, "/portal/v1/api-keys", `{"name":`, nil); code != http.StatusBadRequest || problemType(t, b) != "malformed-request" {
		t.Errorf("malformed JSON: %d %s", code, b)
	}

	// 5. Re-authentication (G4-7): a wrong password is 403, creates no key and returns nothing
	// secret; the password is never logged or echoed.
	wrongPassword := "not the owner's password"
	code, _, b := owner.send(http.MethodPost, "/portal/v1/api-keys", `{"name":"k","scopes":["messages:send"],"current_password":"`+wrongPassword+`"}`, nil)
	if code != http.StatusForbidden || problemType(t, b) != "reauthentication-failed" || strings.Contains(string(b), "smsk_") || strings.Contains(string(b), wrongPassword) {
		t.Errorf("wrong password: %d %s", code, b)
	}
	if keysNow, err := keys.List(ctx, tenantA); err != nil || len(keysNow) != 1 {
		t.Errorf("a key was created despite the wrong password: %d keys, %v", len(keysNow), err)
	}

	// 6. Create: 201 with the full key once, uncached; the key is never logged.
	code, hdr, b := owner.send(http.MethodPost, "/portal/v1/api-keys", withPassword(`"name":"  Order notifications ","scopes":["messages:send","messages:read"]`), nil)
	var created portalapi.ApiKeyCreated
	_ = json.Unmarshal(b, &created)
	if code != http.StatusCreated || created.ApiKey.Name != "Order notifications" || created.ApiKey.Status != "active" ||
		created.ApiKey.RevokedAt != nil || len(created.ApiKey.Scopes) != 2 || !strings.HasPrefix(created.Key, created.ApiKey.Prefix+"_") ||
		hdr.Get("Cache-Control") != "no-store" {
		t.Fatalf("create: %d %q %s", code, hdr.Get("Cache-Control"), b)
	}
	secret := strings.TrimPrefix(created.Key, created.ApiKey.Prefix+"_")
	if strings.Contains(logs.String(), secret) || !strings.Contains(logs.String(), created.ApiKey.Id) {
		t.Error("the key secret was logged, or the creation was not logged by key ID")
	}
	if strings.Contains(logs.String(), portalTestPassword) || strings.Contains(logs.String(), wrongPassword) {
		t.Error("a submitted password was logged")
	}

	// 7. The list shows metadata only: never the secret or a hash.
	code, _, b = owner.do(http.MethodGet, "/portal/v1/api-keys", "", nil)
	var list portalapi.ApiKeyList
	_ = json.Unmarshal(b, &list)
	if code != http.StatusOK || len(list.Data) != 2 || list.Data[0].Id != existing.IDString() || list.Data[1].Id != created.ApiKey.Id {
		t.Fatalf("list: %d %s", code, b)
	}
	for _, forbidden := range []string{secret, "hash", tenantB.String()} {
		if strings.Contains(string(b), forbidden) {
			t.Errorf("the key list contains %q: %s", forbidden, b)
		}
	}

	// 8. The new key works on the public API with its scopes; a read-only key cannot send.
	publicSend := func(token, idem string) (int, []byte) {
		req := newRequest(t, http.MethodPost, "/v1/messages", `{"to":"+639000000001","sender_id":"ACOMPANY","body":"Test"}`)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Idempotency-Key", idem)
		rec, b := serve(h, req)
		return rec.Code, b
	}
	if code, b := publicSend(created.Key, "k1"); code != http.StatusAccepted {
		t.Fatalf("public API with the new key: %d %s", code, b)
	}
	_, readOnly, err := keys.Create(ctx, tenantA, "Read only", []apikey.Scope{apikey.ScopeMessagesRead})
	if err != nil {
		t.Fatal(err)
	}
	if code, b := publicSend(readOnly, "k2"); code != http.StatusForbidden {
		t.Errorf("read-only key sends: %d %s", code, b)
	}

	// 9. Tenant isolation: tenant B's owner neither sees nor revokes tenant A's keys.
	if code, _, b := ownerB.do(http.MethodGet, "/portal/v1/api-keys", "", nil); code != http.StatusOK || string(b) != "{\"data\":[]}\n" {
		t.Errorf("tenant B list: %d %s", code, b)
	}
	if code, _, b := ownerB.send(http.MethodDelete, "/portal/v1/api-keys/"+created.ApiKey.Id, "", nil); code != http.StatusNotFound {
		t.Errorf("tenant B revokes tenant A's key: %d %s", code, b)
	}
	if code, b := publicSend(created.Key, "k3"); code != http.StatusAccepted {
		t.Errorf("tenant A's key after tenant B's revocation attempt: %d %s", code, b)
	}

	// 10. Revoke: the key is refused by the public API from the next request on; revoking again
	// returns it unchanged; unknown and malformed IDs are not found.
	code, _, b = owner.send(http.MethodDelete, "/portal/v1/api-keys/"+created.ApiKey.Id, "", nil)
	var revoked portalapi.ApiKey
	_ = json.Unmarshal(b, &revoked)
	if code != http.StatusOK || revoked.Status != "revoked" || revoked.RevokedAt == nil || revoked.Id != created.ApiKey.Id {
		t.Fatalf("revoke: %d %s", code, b)
	}
	if code, b := publicSend(created.Key, "k4"); code != http.StatusUnauthorized {
		t.Errorf("public API with a revoked key: %d %s", code, b)
	}
	code, _, b = owner.send(http.MethodDelete, "/portal/v1/api-keys/"+created.ApiKey.Id, "", nil)
	var again portalapi.ApiKey
	_ = json.Unmarshal(b, &again)
	if code != http.StatusOK || again.Status != "revoked" || !again.RevokedAt.Equal(*revoked.RevokedAt) {
		t.Errorf("revoke again: %d %s", code, b)
	}
	for _, id := range []string{"06h1wh346hsf13tppjr44p43pw", "not-a-key", strings.ToUpper(created.ApiKey.Id) + "x"} {
		if code, _, b := owner.send(http.MethodDelete, "/portal/v1/api-keys/"+id, "", nil); code != http.StatusNotFound {
			t.Errorf("revoke %q: %d %s", id, code, b)
		}
	}
	code, _, b = owner.do(http.MethodGet, "/portal/v1/api-keys", "", nil)
	list = portalapi.ApiKeyList{}
	_ = json.Unmarshal(b, &list)
	if code != http.StatusOK || len(list.Data) != 3 || list.Data[1].Status != "revoked" || list.Data[0].Status != "active" {
		t.Errorf("list after revocation: %d %s", code, b)
	}

	// 11. Failed re-authentications count toward the sign-in lockout: after five, the account is
	// locked, the right password is refused too, and so is a new sign-in.
	lockedOwner := signedIn(tenantA, "locked-owner@a.example.invalid", portal.RoleOwner)
	for range 5 {
		if code, _, b := lockedOwner.send(http.MethodPost, "/portal/v1/api-keys", `{"name":"k","scopes":["messages:send"],"current_password":"`+wrongPassword+`"}`, nil); code != http.StatusForbidden {
			t.Fatalf("wrong password: %d %s", code, b)
		}
	}
	if code, _, b := lockedOwner.send(http.MethodPost, "/portal/v1/api-keys", withPassword(`"name":"k","scopes":["messages:send"]`), nil); code != http.StatusForbidden || problemType(t, b) != "reauthentication-failed" {
		t.Errorf("locked account, right password: %d %s", code, b)
	}
	relogin := &portalClient{t: t, h: h}
	if code, b := relogin.signIn("locked-owner@a.example.invalid", portalTestPassword); code != http.StatusUnauthorized {
		t.Errorf("sign-in while locked: %d %s", code, b)
	}
	if err := accounts.Unlock(ctx, tenantA, "locked-owner@a.example.invalid"); err != nil {
		t.Fatal(err)
	}
	if code, _, b := lockedOwner.send(http.MethodPost, "/portal/v1/api-keys", withPassword(`"name":"After unlock","scopes":["messages:send"]`), nil); code != http.StatusCreated {
		t.Errorf("after unlock: %d %s", code, b)
	}
}
