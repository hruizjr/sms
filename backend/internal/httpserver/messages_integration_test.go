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
	"github.com/hruizjr/sms/backend/internal/publicapi"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

// End-to-end test of the public API v1 with the real services, a throwaway database (as sms_app,
// row-level security in force) and the simulator. Verified against the simulator only.
func TestIntegrationPublicAPIEndToEnd(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	roles := dbtest.NewMigratedDatabase(t)
	db, err := database.Open(ctx, roles.App, "sms-publicapi-test", database.RequireApplicationRole())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)

	sim := simulator.New(simulator.Config{})
	msgs := messaging.NewService(db, sim.Capabilities())
	keys := apikey.NewService(db)
	dispatcher, err := messaging.NewDispatcher(db, sim, messaging.DispatcherConfig{
		PollInterval: time.Millisecond, BatchSize: 10, Concurrency: 2, SubmitTimeout: time.Second, Lease: 5 * time.Second,
		MaxAttempts: 2, RetryBaseDelay: time.Nanosecond, RetryMaxDelay: time.Nanosecond, MaxMessageAge: time.Hour,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	deps, _ := testDeps(true)
	deps.PublicAPI = &PublicAPIDeps{Messages: msgs, Auth: keys}
	h := NewRouter(deps)

	tenants := tenancy.NewRepository(db.Pool())
	newTenant := func(name string) (tenancy.ID, string) {
		tn, err := tenants.Create(ctx, name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tenants.Activate(ctx, tn.ID); err != nil {
			t.Fatal(err)
		}
		if err := msgs.ApproveSenderID(ctx, tn.ID, "EXAMPLECO"); err != nil {
			t.Fatal(err)
		}
		_, token, err := keys.Create(ctx, tn.ID, "test", []apikey.Scope{apikey.ScopeMessagesSend, apikey.ScopeMessagesRead})
		if err != nil {
			t.Fatal(err)
		}
		return tn.ID, token
	}
	_, tokenA := newTenant("API Tenant A")
	_, tokenB := newTenant("API Tenant B")

	call := func(token, method, target, body, key string) (int, http.Header, []byte) {
		t.Helper()
		req := newRequest(t, method, target, body)
		req.Header.Set("Authorization", "Bearer "+token)
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
		}
		rec, b := serve(h, req)
		return rec.Code, rec.Header(), b
	}
	message := func(b []byte) publicapi.Message {
		t.Helper()
		var m publicapi.Message
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("%v: %s", err, b)
		}
		return m
	}
	cycle := func() {
		t.Helper()
		if _, err := dispatcher.RunOnce(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := sim.DeliverPending(ctx, messaging.NewInboxSink(db)); err != nil {
			t.Fatal(err)
		}
		if _, err := dispatcher.RunOnce(ctx); err != nil {
			t.Fatal(err)
		}
	}

	// 1. Submit: 202 with a persisted message ID; status accepted.
	body := `{"to":"+639000000001","sender_id":"EXAMPLECO","body":"Your code is 000000.","client_reference":"login-1"}`
	code, hdr, b := call(tokenA, http.MethodPost, "/v1/messages", body, "e2e-1")
	if code != http.StatusAccepted {
		t.Fatalf("POST: %d %s", code, b)
	}
	created := message(b)
	if created.Status != "accepted" || hdr.Get("Location") != "/v1/messages/"+created.Id {
		t.Errorf("created %+v, Location %q", created, hdr.Get("Location"))
	}

	// 2. Same key, same body in another form (key order, whitespace): replayed, same ID.
	code, hdr, b = call(tokenA, http.MethodPost, "/v1/messages",
		` { "client_reference":"login-1", "body":"Your code is 000000.", "sender_id":"EXAMPLECO", "to":"+639000000001" } `, `"e2e-1"`)
	if code != http.StatusAccepted || hdr.Get("Idempotent-Replayed") != "true" || message(b).Id != created.Id {
		t.Errorf("replay: %d %q %s", code, hdr.Get("Idempotent-Replayed"), b)
	}
	// 3. Same key, different body: 422.
	if code, _, b = call(tokenA, http.MethodPost, "/v1/messages", strings.Replace(body, "000000", "111111", 1), "e2e-1"); code != 422 ||
		!strings.Contains(string(b), "idempotency-key-reused") {
		t.Errorf("key reuse: %d %s", code, b)
	}

	// 4. The dispatcher hands it to the simulator; the receipt makes it delivered.
	cycle()
	code, _, b = call(tokenA, http.MethodGet, "/v1/messages/"+created.Id, "", "")
	if got := message(b); code != 200 || got.Status != "delivered" || got.StatusCategory != "final" || got.StatusReason != nil {
		t.Errorf("after the receipt: %d %s", code, b)
	}
	if len(sim.Submissions()) != 1 {
		t.Errorf("%d simulator submissions, want 1", len(sim.Submissions()))
	}

	// 5. An uncertain submission is reported as such and not resent.
	code, _, b = call(tokenA, http.MethodPost, "/v1/messages", `{"to":"+639000000030","sender_id":"EXAMPLECO","body":"Alert"}`, "e2e-2")
	if code != 202 {
		t.Fatalf("POST uncertain: %d %s", code, b)
	}
	uncertain := message(b)
	for range 4 {
		cycle()
	}
	_, _, b = call(tokenA, http.MethodGet, "/v1/messages/"+uncertain.Id, "", "")
	if got := message(b); got.Status != "submission_uncertain" || got.StatusReason == nil || got.StatusReason.Code != "submission_outcome_unknown" {
		t.Errorf("uncertain: %s", b)
	}
	if n := len(sim.Submissions()); n != 2 {
		t.Errorf("%d simulator submissions, want 2 (the uncertain one exactly once)", n)
	}

	// 6. Tenant isolation through the API: B cannot see A's messages.
	if code, _, b = call(tokenB, http.MethodGet, "/v1/messages/"+created.Id, "", ""); code != 404 {
		t.Errorf("tenant B GET of tenant A's message: %d %s", code, b)
	}
	_, _, b = call(tokenB, http.MethodGet, "/v1/messages", "", "")
	var page publicapi.MessageList
	if err := json.Unmarshal(b, &page); err != nil || len(page.Data) != 0 {
		t.Errorf("tenant B list: %s", b)
	}
	_, _, b = call(tokenA, http.MethodGet, "/v1/messages?client_reference=login-1", "", "")
	if err := json.Unmarshal(b, &page); err != nil || len(page.Data) != 1 || page.Data[0].Id != created.Id {
		t.Errorf("tenant A list by client_reference: %s", b)
	}

	// 7. Unapproved sender ID and unsupported destination: 422, not recorded.
	if code, _, b = call(tokenA, http.MethodPost, "/v1/messages", `{"to":"+639000000001","sender_id":"OTHER","body":"x"}`, "e2e-3"); code != 422 || !strings.Contains(string(b), "sender-id-not-approved") {
		t.Errorf("unapproved sender: %d %s", code, b)
	}
	if code, _, b = call(tokenA, http.MethodPost, "/v1/messages", `{"to":"+12025550123","sender_id":"EXAMPLECO","body":"x"}`, "e2e-4"); code != 422 || !strings.Contains(string(b), "destination-not-supported") {
		t.Errorf("unsupported destination: %d %s", code, b)
	}

	// 8. A revoked key is refused immediately.
	p, err := keys.Authenticate(ctx, tokenB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := keys.Revoke(ctx, p.TenantID, p.KeyID); err != nil {
		t.Fatal(err)
	}
	if code, _, _ = call(tokenB, http.MethodGet, "/v1/messages", "", ""); code != 401 {
		t.Errorf("revoked key: %d, want 401", code)
	}
}
