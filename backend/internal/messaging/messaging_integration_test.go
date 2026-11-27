//go:build integration

package messaging

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/dbtest"
	"github.com/hruizjr/sms/backend/internal/database/sqlcgen"
	"github.com/hruizjr/sms/backend/internal/messaging/provider"
	"github.com/hruizjr/sms/backend/internal/messaging/provider/simulator"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

// Integration tests of the messaging engine against a throwaway PostgreSQL database (dbtest), as the
// application role sms_app with row-level security in force. Results are "verified against the
// simulator" only (decision log §15, S7).

const senderID = "EXAMPLECO"

type env struct {
	t     *testing.T
	ctx   context.Context
	app   *database.DB // sms_app: the role the API and the dispatcher use
	admin *database.DB // bootstrap superuser: only to inspect or backdate rows in tests
	svc   *Service
	sim   *simulator.Simulator
	d     *Dispatcher
	sink  *InboxSink
	cfg   DispatcherConfig
}

func testDispatcherConfig() DispatcherConfig {
	return DispatcherConfig{
		PollInterval: 10 * time.Millisecond, BatchSize: 50, Concurrency: 4,
		SubmitTimeout: 5 * time.Second, Lease: 10 * time.Second, MaxAttempts: 3,
		RetryBaseDelay: time.Nanosecond, RetryMaxDelay: time.Nanosecond, MaxMessageAge: time.Hour,
	}
}

func newEnv(t *testing.T, modify func(*DispatcherConfig)) *env {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	roles := dbtest.NewMigratedDatabase(t)
	app, err := database.Open(ctx, roles.App, "sms-messaging-test", database.RequireApplicationRole())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Close)
	admin, err := database.Open(ctx, roles.Admin, "sms-messaging-test-admin")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	cfg := testDispatcherConfig()
	if modify != nil {
		modify(&cfg)
	}
	sim := simulator.New(simulator.Config{})
	e := &env{t: t, ctx: ctx, app: app, admin: admin, svc: NewService(app, sim.Capabilities()), sim: sim, sink: NewInboxSink(app), cfg: cfg}
	e.d = e.dispatcher(sim)
	return e
}

func (e *env) dispatcher(p provider.Provider) *Dispatcher {
	e.t.Helper()
	d, err := NewDispatcher(e.app, p, e.cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		e.t.Fatal(err)
	}
	return d
}

// tenant creates an active tenant with the approved sender ID.
func (e *env) tenant(name string) tenancy.ID {
	e.t.Helper()
	repo := tenancy.NewRepository(e.app.Pool())
	t, err := repo.Create(e.ctx, name)
	if err != nil {
		e.t.Fatal(err)
	}
	if _, err := repo.Activate(e.ctx, t.ID); err != nil {
		e.t.Fatal(err)
	}
	if err := e.svc.ApproveSenderID(e.ctx, t.ID, senderID); err != nil {
		e.t.Fatal(err)
	}
	return t.ID
}

func request(to, key string) AcceptRequest {
	return AcceptRequest{Destination: to, SenderID: senderID, Body: "Your verification code is 000000.", IdempotencyKey: key}
}

func (e *env) accept(tenantID tenancy.ID, to, key string) Message {
	e.t.Helper()
	res, err := e.svc.Accept(e.ctx, tenantID, request(to, key))
	if err != nil {
		e.t.Fatalf("Accept(%s): %v", to, err)
	}
	if res.Replayed {
		e.t.Fatalf("Accept(%s) replayed", to)
	}
	return res.Message
}

// cycle runs one dispatch cycle, then delivers the simulator's pending receipts, then runs another
// cycle so that they are applied.
func (e *env) cycle() {
	e.t.Helper()
	if _, err := e.d.RunOnce(e.ctx); err != nil {
		e.t.Fatalf("RunOnce: %v", err)
	}
	if _, err := e.sim.DeliverPending(e.ctx, e.sink); err != nil {
		e.t.Fatalf("DeliverPending: %v", err)
	}
	if _, err := e.d.RunOnce(e.ctx); err != nil {
		e.t.Fatalf("RunOnce: %v", err)
	}
}

func (e *env) trace(tenantID tenancy.ID, id MessageID) Trace {
	e.t.Helper()
	tr, err := e.svc.Trace(e.ctx, tenantID, id)
	if err != nil {
		e.t.Fatalf("Trace: %v", err)
	}
	return tr
}

func (e *env) adminExec(sql string, args ...any) {
	e.t.Helper()
	if _, err := e.admin.Pool().Exec(e.ctx, sql, args...); err != nil {
		e.t.Fatalf("%s: %v", sql, err)
	}
}

func adminValue[T any](e *env, sql string, args ...any) T {
	e.t.Helper()
	var v T
	if err := e.admin.Pool().QueryRow(e.ctx, sql, args...).Scan(&v); err != nil {
		e.t.Fatalf("%s: %v", sql, err)
	}
	return v
}

func (e *env) submissionsTo(destination string) int {
	n := 0
	for _, s := range e.sim.Submissions() {
		if s.Destination == destination {
			n++
		}
	}
	return n
}

func TestIntegrationAcceptStoresMessageAndDispatchJob(t *testing.T) {
	t.Parallel()
	e := newEnv(t, nil)
	tenantID := e.tenant("Accept Co")
	m := e.accept(tenantID, "+639000000001", "k1")

	if m.State != StateQueued || m.State.Status() != StatusAccepted || m.Reason != nil {
		t.Errorf("accepted message state %s (%s), reason %v", m.State, m.State.Status(), m.Reason)
	}
	if m.Encoding != EncodingGSM7 || m.SegmentCount != 1 || m.AcceptedAt.IsZero() || !m.StatusUpdatedAt.Equal(m.AcceptedAt) {
		t.Errorf("message = %+v", m)
	}
	if m.ID[6]>>4 != 7 {
		t.Errorf("message ID version %d, want 7 (UUIDv7)", m.ID[6]>>4)
	}
	// The message, its body, its dispatch job, its history and its idempotency record are stored together.
	id := m.ID[:]
	if n := adminValue[int](e, `SELECT count(*) FROM dispatch_jobs WHERE message_id = $1 AND completed_at IS NULL`, id); n != 1 {
		t.Errorf("%d open dispatch jobs, want 1", n)
	}
	if body := adminValue[string](e, `SELECT body FROM message_bodies WHERE message_id = $1`, id); body != "Your verification code is 000000." {
		t.Errorf("stored body %q", body)
	}
	if n := adminValue[int](e, `SELECT count(*) FROM idempotency_keys WHERE message_id = $1 AND expires_at > now() + interval '6 days'`, id); n != 1 {
		t.Errorf("%d idempotency records with seven-day retention, want 1", n)
	}
	tr := e.trace(tenantID, m.ID)
	if len(tr.History) != 1 || tr.History[0].From != nil || tr.History[0].To != StateQueued || tr.History[0].Source != "api" {
		t.Errorf("history = %+v", tr.History)
	}
	got, err := e.svc.Get(e.ctx, tenantID, m.ID)
	if err != nil || got.ID != m.ID || got.Destination != "+639000000001" {
		t.Errorf("Get() = %+v, %v", got, err)
	}
}

func TestIntegrationAcceptRules(t *testing.T) {
	t.Parallel()
	e := newEnv(t, nil)
	tenantID := e.tenant("Rules Co")

	req := request("+639000000001", "k-rules")
	req.SenderID = "NOTAPPROVED"
	if _, err := e.svc.Accept(e.ctx, tenantID, req); !errors.Is(err, ErrSenderIDNotApproved) {
		t.Errorf("unapproved sender ID: %v", err)
	}
	// Rejections are not recorded: after approval, the same key is accepted.
	if err := e.svc.ApproveSenderID(e.ctx, tenantID, "NOTAPPROVED"); err != nil {
		t.Fatal(err)
	}
	if res, err := e.svc.Accept(e.ctx, tenantID, req); err != nil || res.Replayed {
		t.Errorf("after approval: %+v, %v", res, err)
	}
	for _, to := range []string{"+12025550123", "+63281234567", "+6390000000"} {
		if _, err := e.svc.Accept(e.ctx, tenantID, request(to, "k-"+to)); !errors.Is(err, ErrDestinationNotSupported) {
			t.Errorf("destination %s: %v, want ErrDestinationNotSupported (D-043)", to, err)
		}
	}
	limited := NewService(e.app, provider.Capabilities{MaxSegments: 1})
	long := request("+639000000001", "k-long")
	long.Body = strings.Repeat("a", 161)
	if _, err := limited.Accept(e.ctx, tenantID, long); !errors.Is(err, ErrMessageTooLong) {
		t.Errorf("two segments with a known limit of one: %v", err)
	}
	if res, err := e.svc.Accept(e.ctx, tenantID, long); err != nil || res.Message.SegmentCount != 2 {
		t.Errorf("unknown segment limit is not enforced: %+v, %v", res, err)
	}
}

func TestIntegrationIdempotency(t *testing.T) {
	t.Parallel()
	e := newEnv(t, nil)
	tenantID := e.tenant("Idempotency Co")
	first := e.accept(tenantID, "+639000000001", "key-1")

	// Identical request: the original acceptance is replayed; no second message.
	res, err := e.svc.Accept(e.ctx, tenantID, request("+639000000001", "key-1"))
	if err != nil || !res.Replayed || res.Message.ID != first.ID {
		t.Fatalf("replay = %+v, %v", res, err)
	}
	// Different request with the same key: refused, nothing stored.
	if _, err := e.svc.Accept(e.ctx, tenantID, request("+639000000002", "key-1")); !errors.Is(err, ErrIdempotencyKeyReused) {
		t.Errorf("reused key: %v", err)
	}
	if n := adminValue[int](e, `SELECT count(*) FROM messages`); n != 1 {
		t.Errorf("%d messages, want 1", n)
	}
	// The replay shows the acceptance-time state even after the status changed.
	e.cycle()
	res, err = e.svc.Accept(e.ctx, tenantID, request("+639000000001", "key-1"))
	if err != nil || !res.Replayed || res.Message.State != StateQueued || !res.Message.StatusUpdatedAt.Equal(first.AcceptedAt) {
		t.Errorf("replay after delivery = %+v, %v", res, err)
	}
	if cur, _ := e.svc.Get(e.ctx, tenantID, first.ID); cur.State != StateDelivered {
		t.Errorf("current state %s, want delivered", cur.State)
	}
	// After the seven-day retention the key is treated as new.
	e.adminExec(`UPDATE idempotency_keys SET created_at = now() - interval '8 days', expires_at = now() - interval '1 day'`)
	res, err = e.svc.Accept(e.ctx, tenantID, request("+639000000002", "key-1"))
	if err != nil || res.Replayed || res.Message.ID == first.ID {
		t.Errorf("expired key = %+v, %v; want a new message", res, err)
	}
}

func TestIntegrationConcurrentRequestsWithOneKeyCreateOneMessage(t *testing.T) {
	t.Parallel()
	e := newEnv(t, nil)
	tenantID := e.tenant("Concurrent Co")
	const n = 12
	results := make([]AcceptResult, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = e.svc.Accept(e.ctx, tenantID, request("+639000000001", "same-key"))
		}()
	}
	wg.Wait()
	originals := 0
	for i := range n {
		if errs[i] != nil {
			t.Fatalf("request %d: %v", i, errs[i])
		}
		if results[i].Message.ID != results[0].Message.ID {
			t.Errorf("request %d got message %s, want %s", i, results[i].Message.ID, results[0].Message.ID)
		}
		if !results[i].Replayed {
			originals++
		}
	}
	if originals != 1 {
		t.Errorf("%d original acceptances, want 1", originals)
	}
	if c := adminValue[int](e, `SELECT count(*) FROM messages`); c != 1 {
		t.Errorf("%d messages, want 1", c)
	}
}

func TestIntegrationTenantIsolation(t *testing.T) {
	t.Parallel()
	e := newEnv(t, nil)
	a, b := e.tenant("Tenant A"), e.tenant("Tenant B")
	ma := e.accept(a, "+639000000001", "shared-key")

	if _, err := e.svc.Get(e.ctx, b, ma.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("tenant B reading tenant A's message: %v, want ErrNotFound", err)
	}
	if _, err := e.svc.Trace(e.ctx, b, ma.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("tenant B tracing tenant A's message: %v", err)
	}
	if list, _, err := e.svc.List(e.ctx, b, ListFilter{Limit: 100}); err != nil || len(list) != 0 {
		t.Errorf("tenant B's list = %d messages, %v; want none", len(list), err)
	}
	// Idempotency keys are scoped per tenant: the same key and body create B's own message.
	mb := e.accept(b, "+639000000001", "shared-key")
	if mb.ID == ma.ID {
		t.Error("the same key in another tenant must not replay")
	}

	// Row-level security, below the application filters: without tenant context sms_app sees no
	// tenant rows; inside B's context it cannot see or write A's rows.
	for _, table := range []string{"messages", "message_bodies", "dispatch_jobs", "idempotency_keys", "message_status_history", "sender_ids"} {
		var n int
		if err := e.app.Pool().QueryRow(e.ctx, `SELECT count(*) FROM `+table).Scan(&n); err != nil || n != 0 {
			t.Errorf("%s without tenant context: %d rows, %v; want 0", table, n, err)
		}
	}
	err := e.app.InTenantTx(e.ctx, b, func(tx pgx.Tx) error {
		var n int
		if err := tx.QueryRow(e.ctx, `SELECT count(*) FROM messages WHERE id = $1`, ma.ID[:]).Scan(&n); err != nil {
			return err
		}
		if n != 0 {
			t.Errorf("tenant B sees tenant A's message")
		}
		tag, err := tx.Exec(e.ctx, `UPDATE messages SET status = 'failed', status_reason = 'retries_exhausted' WHERE id = $1`, ma.ID[:])
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 0 {
			t.Errorf("tenant B updated tenant A's message")
		}
		_, err = tx.Exec(e.ctx, `INSERT INTO messages (tenant_id, destination, sender_id, encoding, segment_count) VALUES ($1, '+639000000001', 'X', 'gsm7', 1)`, a[:])
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
			t.Errorf("inserting a row for tenant A in tenant B's context: %v, want a row-level security violation", err)
		}
		return errors.New("rollback")
	})
	if err == nil || err.Error() != "rollback" {
		t.Fatalf("InTenantTx: %v", err)
	}
	if cur, _ := e.svc.Get(e.ctx, a, ma.ID); cur.State != StateQueued {
		t.Errorf("tenant A's message changed: %s", cur.State)
	}
}

func TestIntegrationListPagination(t *testing.T) {
	t.Parallel()
	e := newEnv(t, nil)
	tenantID := e.tenant("List Co")
	var ids []MessageID
	for i := range 5 {
		req := request("+639000000001", "list-"+string(rune('a'+i)))
		if i%2 == 0 {
			ref := "even"
			req.ClientReference = &ref
		}
		res, err := e.svc.Accept(e.ctx, tenantID, req)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, res.Message.ID)
	}
	var got []MessageID
	cursor := ""
	for range 5 {
		page, next, err := e.svc.List(e.ctx, tenantID, ListFilter{Limit: 2, Cursor: cursor})
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range page {
			got = append(got, m.ID)
		}
		if next == "" {
			break
		}
		cursor = next
	}
	if len(got) != 5 {
		t.Fatalf("paged through %d messages, want 5", len(got))
	}
	for i := range got { // newest acceptance first
		if got[i] != ids[len(ids)-1-i] {
			t.Errorf("position %d: %s, want %s", i, got[i], ids[len(ids)-1-i])
		}
	}
	ref := "even"
	if page, _, err := e.svc.List(e.ctx, tenantID, ListFilter{Limit: 10, ClientReference: &ref}); err != nil || len(page) != 3 {
		t.Errorf("client_reference filter: %d, %v; want 3", len(page), err)
	}
	accepted := StatusAccepted
	delivered := StatusDelivered
	if page, _, _ := e.svc.List(e.ctx, tenantID, ListFilter{Limit: 10, Status: &accepted}); len(page) != 5 {
		t.Errorf("status=accepted: %d, want 5", len(page))
	}
	if page, _, _ := e.svc.List(e.ctx, tenantID, ListFilter{Limit: 10, Status: &delivered}); len(page) != 0 {
		t.Errorf("status=delivered: %d, want 0", len(page))
	}
	// A cursor is bound to its filters.
	_, next, _ := e.svc.List(e.ctx, tenantID, ListFilter{Limit: 2})
	if _, _, err := e.svc.List(e.ctx, tenantID, ListFilter{Limit: 2, Cursor: next, Status: &accepted}); !errors.Is(err, ErrInvalidCursor) {
		t.Errorf("cursor reused with other filters: %v", err)
	}
	if _, _, err := e.svc.List(e.ctx, tenantID, ListFilter{Limit: 2, Cursor: "not-a-cursor"}); !errors.Is(err, ErrInvalidCursor) {
		t.Errorf("malformed cursor: %v", err)
	}
}

// TestIntegrationDispatchScenarios runs every simulator scenario through the dispatcher.
func TestIntegrationDispatchScenarios(t *testing.T) {
	t.Parallel()
	e := newEnv(t, nil)
	tenantID := e.tenant("Scenario Co")
	type want struct {
		state    State
		reason   Reason
		attempts []string
		receipts []string
	}
	scenarios := map[string]want{
		"+639000000001": {StateDelivered, "", []string{"accepted"}, []string{"applied"}},
		"+639000000010": {StateRejected, ReasonProviderRejected, []string{"rejected"}, nil},
		"+639000000020": {StateFailed, ReasonRetriesExhausted, []string{"not_sent_retryable", "not_sent_retryable", "not_sent_retryable"}, nil},
		"+639000000025": {StateFailed, ReasonPermanentSubmissionError, []string{"not_sent_permanent"}, nil},
		"+639000000030": {StateSubmitUncertain, ReasonSubmissionOutcomeUnknown, []string{"uncertain"}, nil},
		"+639000000040": {StateUndelivered, ReasonProviderReportedUndelivered, []string{"accepted"}, []string{"applied"}},
		"+639000000041": {StateUndelivered, ReasonProviderReportedExpired, []string{"accepted"}, []string{"applied"}},
		"+639000000050": {StateDelivered, "", []string{"accepted"}, []string{"applied", "duplicate"}},
		"+639000000060": {StateSubmitted, "", []string{"accepted"}, nil},
	}
	ids := map[string]MessageID{}
	for to := range scenarios {
		ids[to] = e.accept(tenantID, to, "k"+to).ID
	}

	// After the first hand-off, accepted messages are provider_accepted, not delivered: delivery is
	// claimed only when a receipt has been processed.
	if _, err := e.d.RunOnce(e.ctx); err != nil {
		t.Fatal(err)
	}
	if m, _ := e.svc.Get(e.ctx, tenantID, ids["+639000000001"]); m.State != StateSubmitted || m.State.Status() != StatusProviderAccepted {
		t.Errorf("before any receipt: %s (%s), want submitted (provider_accepted)", m.State, m.State.Status())
	}
	for range 6 { // enough cycles for the retries and the receipts
		e.cycle()
	}

	for to, w := range scenarios {
		tr := e.trace(tenantID, ids[to])
		m := tr.Message
		if m.State != w.state || (w.reason == "") != (m.Reason == nil) || (m.Reason != nil && *m.Reason != w.reason) {
			t.Errorf("%s: state %s reason %v, want %s %q", to, m.State, m.Reason, w.state, w.reason)
		}
		var attempts []string
		for _, a := range tr.Attempts {
			attempts = append(attempts, a.State)
			if a.ProviderID != simulator.ID {
				t.Errorf("%s: attempt provider %q", to, a.ProviderID)
			}
		}
		if strings.Join(attempts, ",") != strings.Join(w.attempts, ",") {
			t.Errorf("%s: attempts %v, want %v", to, attempts, w.attempts)
		}
		var receipts []string
		for _, r := range tr.Receipts {
			receipts = append(receipts, r.Effect)
		}
		if strings.Join(receipts, ",") != strings.Join(w.receipts, ",") {
			t.Errorf("%s: receipts %v, want %v", to, receipts, w.receipts)
		}
		if got := e.submissionsTo(to); got != len(w.attempts) {
			t.Errorf("%s: simulator received %d submissions, want %d", to, got, len(w.attempts))
		}
	}

	// Audit trail of a delivered message.
	var path []string
	for _, h := range e.trace(tenantID, ids["+639000000001"]).History {
		path = append(path, string(h.To)+"/"+h.Source)
	}
	if want := "queued/api,submitting/dispatcher,submitted/dispatcher,delivered/receipt"; strings.Join(path, ",") != want {
		t.Errorf("history %v, want %s", path, want)
	}
	// Every job is finished: uncertain, final and submitted messages need no further hand-off.
	if n := adminValue[int](e, `SELECT count(*) FROM dispatch_jobs WHERE completed_at IS NULL`); n != 0 {
		t.Errorf("%d open dispatch jobs, want 0", n)
	}
}

// TestIntegrationUncertainSubmissionIsNeverRetried: the D-095 interim rule.
func TestIntegrationUncertainSubmissionIsNeverRetried(t *testing.T) {
	t.Parallel()
	e := newEnv(t, nil)
	tenantID := e.tenant("Uncertain Co")
	m := e.accept(tenantID, "+639000000030", "k1")
	for range 10 {
		e.cycle()
	}
	// Even with every lease expired, nothing resubmits it.
	e.adminExec(`UPDATE dispatch_jobs SET lease_expires_at = now() - interval '1 hour', available_at = now() - interval '1 hour'`)
	e.cycle()
	if got := e.submissionsTo("+639000000030"); got != 1 {
		t.Errorf("uncertain submission was sent %d times, want exactly 1 (never resubmitted)", got)
	}
	cur, _ := e.svc.Get(e.ctx, tenantID, m.ID)
	if cur.State != StateSubmitUncertain || cur.State.Status() != StatusSubmissionUncertain || cur.State.Status().Category() != CategoryUncertain {
		t.Errorf("state %s, want submit_uncertain (not failed, not queued)", cur.State)
	}
}

func TestIntegrationUnknownAfterReportWindowAndLateReceipt(t *testing.T) {
	t.Parallel()
	e := newEnv(t, func(c *DispatcherConfig) { c.ReportWindow = time.Hour })
	tenantID := e.tenant("Window Co")
	m := e.accept(tenantID, "+639000000060", "k1") // accepted, never a receipt
	e.cycle()
	if cur, _ := e.svc.Get(e.ctx, tenantID, m.ID); cur.State != StateSubmitted {
		t.Fatalf("state %s, want submitted within the window", cur.State)
	}
	e.adminExec(`UPDATE messages SET accepted_at = accepted_at - interval '2 hours', status_updated_at = status_updated_at - interval '2 hours'`)
	e.cycle()
	cur, _ := e.svc.Get(e.ctx, tenantID, m.ID)
	if cur.State != StateUnknown || cur.Reason == nil || *cur.Reason != ReasonNoFinalReportReceived {
		t.Fatalf("after the window: %s %v, want unknown / no_final_report_received", cur.State, cur.Reason)
	}
	// A late report still updates it.
	ref := *e.trace(tenantID, m.ID).Attempts[0].ProviderMessageRef
	if err := e.sink.Deliver(e.ctx, simulator.ID, provider.Receipt{ProviderMessageRef: ref, IdentityKey: ref + ":late", Status: provider.ReceiptDelivered}); err != nil {
		t.Fatal(err)
	}
	e.cycle()
	if cur, _ := e.svc.Get(e.ctx, tenantID, m.ID); cur.State != StateDelivered {
		t.Errorf("after a late receipt: %s, want delivered", cur.State)
	}
}

func TestIntegrationConflictingAndUnmatchedReceipts(t *testing.T) {
	t.Parallel()
	e := newEnv(t, nil)
	tenantID := e.tenant("Conflict Co")
	m := e.accept(tenantID, "+639000000001", "k1")
	e.cycle()
	ref := *e.trace(tenantID, m.ID).Attempts[0].ProviderMessageRef
	if err := e.sink.Deliver(e.ctx, simulator.ID, provider.Receipt{ProviderMessageRef: ref, IdentityKey: ref + ":other", Status: provider.ReceiptUndelivered}); err != nil {
		t.Fatal(err)
	}
	// A receipt for a reference no attempt has is kept, unmatched, for reconciliation.
	if err := e.sink.Deliver(e.ctx, simulator.ID, provider.Receipt{ProviderMessageRef: "sim-unknown", IdentityKey: "x", Status: provider.ReceiptDelivered}); err != nil {
		t.Fatal(err)
	}
	e.cycle()
	tr := e.trace(tenantID, m.ID)
	if tr.Message.State != StateDelivered {
		t.Errorf("state %s: a conflicting final report must not overwrite delivered", tr.Message.State)
	}
	if last := tr.Receipts[len(tr.Receipts)-1]; last.Effect != "conflict" {
		t.Errorf("last receipt effect %q, want conflict", last.Effect)
	}
	if !adminValue[bool](e, `SELECT has_conflicting_reports FROM messages WHERE id = $1`, m.ID[:]) {
		t.Error("the message is not flagged for reconciliation")
	}
	if n := adminValue[int](e, `SELECT count(*) FROM provider_receipts WHERE matched_at IS NULL AND provider_message_ref = 'sim-unknown'`); n != 1 {
		t.Errorf("unmatched receipt: %d rows, want 1", n)
	}
}

// earlyReceiptProvider reports a receipt before Submit returns, that is before the dispatcher has
// recorded the provider's acknowledgment.
type earlyReceiptProvider struct {
	*simulator.Simulator
	sink provider.ReceiptSink
}

func (p *earlyReceiptProvider) Submit(ctx context.Context, s provider.Submission) provider.SubmitResult {
	res := p.Simulator.Submit(ctx, s)
	if err := p.sink.Deliver(ctx, simulator.ID, provider.Receipt{ProviderMessageRef: res.ProviderMessageRef, IdentityKey: "early", Status: provider.ReceiptDelivered}); err != nil {
		panic(err)
	}
	return res
}

func TestIntegrationReceiptBeforeAcknowledgmentIsMatchedLater(t *testing.T) {
	t.Parallel()
	e := newEnv(t, nil)
	tenantID := e.tenant("Early Co")
	d := e.dispatcher(&earlyReceiptProvider{Simulator: simulator.New(simulator.Config{}), sink: e.sink})
	m := e.accept(tenantID, "+639000000060", "k1")
	if _, err := d.RunOnce(e.ctx); err != nil {
		t.Fatal(err)
	}
	if cur, _ := e.svc.Get(e.ctx, tenantID, m.ID); cur.State != StateDelivered {
		t.Errorf("state %s, want delivered once the early receipt is matched", cur.State)
	}
}

func TestIntegrationConcurrentDispatchersHandEachMessageOverOnce(t *testing.T) {
	t.Parallel()
	e := newEnv(t, func(c *DispatcherConfig) { c.BatchSize = 7 })
	a, b := e.tenant("Busy A"), e.tenant("Busy B")
	const perTenant = 25
	for i := range perTenant {
		e.accept(a, "+639000000001", "a"+string(rune('A'+i)))
		e.accept(b, "+639000000060", "b"+string(rune('A'+i)))
	}
	dispatchers := []*Dispatcher{e.d, e.dispatcher(e.sim), e.dispatcher(e.sim)}
	var wg sync.WaitGroup
	for _, d := range dispatchers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 8 {
				if _, err := d.RunOnce(e.ctx); err != nil {
					t.Errorf("RunOnce: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
	if n := len(e.sim.Submissions()); n != 2*perTenant {
		t.Errorf("%d submissions, want %d (each message handed over exactly once)", n, 2*perTenant)
	}
	seen := map[string]bool{}
	for _, s := range e.sim.Submissions() {
		if seen[s.AttemptID] {
			t.Errorf("attempt %s submitted twice", s.AttemptID)
		}
		seen[s.AttemptID] = true
	}
	if n := adminValue[int](e, `SELECT count(*) FROM (SELECT message_id FROM submission_attempts GROUP BY message_id HAVING count(*) <> 1) x`); n != 0 {
		t.Errorf("%d messages with other than one attempt", n)
	}
	if n := adminValue[int](e, `SELECT count(*) FROM messages WHERE status <> 'submitted'`); n != 0 {
		t.Errorf("%d messages not submitted", n)
	}
}

// TestIntegrationRecoveryAfterRestart simulates dispatcher crashes at each point of a hand-off.
func TestIntegrationRecoveryAfterRestart(t *testing.T) {
	t.Parallel()

	t.Run("crash after claiming, before the attempt: the job is handed over after the lease", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t, nil)
		tenantID := e.tenant("Crash Claim Co")
		m := e.accept(tenantID, "+639000000001", "k1")
		// The "crashed" dispatcher claims the job (taking its lease) and dies before the attempt.
		err := e.app.InTenantTx(e.ctx, tenantID, func(tx pgx.Tx) error {
			ids, err := sqlcgen.New(tx).ClaimDispatchJobs(e.ctx, sqlcgen.ClaimDispatchJobsParams{
				TenantID: pgUUID(tenantID), BatchSize: 10, LeaseSeconds: e.cfg.Lease.Seconds(),
			})
			if err == nil && len(ids) != 1 {
				err = errors.New("claimed no job")
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		restarted := e.dispatcher(e.sim)
		if _, err := restarted.RunOnce(e.ctx); err != nil {
			t.Fatal(err)
		}
		if got := e.submissionsTo("+639000000001"); got != 0 {
			t.Fatalf("%d submissions while the lease is held, want 0", got)
		}
		e.adminExec(`UPDATE dispatch_jobs SET lease_expires_at = now() - interval '1 second'`)
		if _, err := restarted.RunOnce(e.ctx); err != nil {
			t.Fatal(err)
		}
		if cur, _ := e.svc.Get(e.ctx, tenantID, m.ID); cur.State != StateSubmitted || e.submissionsTo("+639000000001") != 1 {
			t.Errorf("after the lease: %s, %d submissions; want submitted once", cur.State, e.submissionsTo("+639000000001"))
		}
	})

	t.Run("crash after recording the attempt: uncertain, never resent", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t, nil)
		tenantID := e.tenant("Crash Attempt Co")
		m := e.accept(tenantID, "+639000000001", "k1")
		start, err := e.d.startAttempt(e.ctx, tenantID, m.ID) // then the process dies
		if err != nil || start == nil {
			t.Fatalf("startAttempt: %v, %v", start, err)
		}
		restartedSim := simulator.New(simulator.Config{})
		restarted := e.dispatcher(restartedSim)
		for range 3 {
			if _, err := restarted.RunOnce(e.ctx); err != nil {
				t.Fatal(err)
			}
		}
		if cur, _ := e.svc.Get(e.ctx, tenantID, m.ID); cur.State != StateSubmitting {
			t.Fatalf("within the lease: %s, want still submitting", cur.State)
		}
		e.adminExec(`UPDATE submission_attempts SET started_at = started_at - interval '1 hour'`)
		e.adminExec(`UPDATE dispatch_jobs SET lease_expires_at = now() - interval '1 second'`)
		for range 3 {
			if _, err := restarted.RunOnce(e.ctx); err != nil {
				t.Fatal(err)
			}
		}
		tr := e.trace(tenantID, m.ID)
		if tr.Message.State != StateSubmitUncertain || len(tr.Attempts) != 1 || tr.Attempts[0].State != "uncertain" {
			t.Errorf("after recovery: %s, attempts %+v; want submit_uncertain with one uncertain attempt", tr.Message.State, tr.Attempts)
		}
		if last := tr.History[len(tr.History)-1]; last.Source != "recovery" {
			t.Errorf("last history entry %+v, want source recovery", last)
		}
		if n := len(restartedSim.Submissions()) + len(e.sim.Submissions()); n != 0 {
			t.Errorf("%d submissions after the crash, want 0", n)
		}
		// A result that arrives after recovery does not change the uncertain message.
		if err := e.d.finishAttempt(e.ctx, tenantID, m.ID, start, provider.SubmitResult{Outcome: provider.OutcomeAccepted, ProviderMessageRef: "sim-late"}); err != nil {
			t.Fatal(err)
		}
		if cur, _ := e.svc.Get(e.ctx, tenantID, m.ID); cur.State != StateSubmitUncertain {
			t.Errorf("after a late result: %s, want submit_uncertain", cur.State)
		}
	})
}

func TestIntegrationSuspendedTenantIsNotDispatched(t *testing.T) {
	t.Parallel()
	e := newEnv(t, nil)
	tenantID := e.tenant("Paused Co")
	m := e.accept(tenantID, "+639000000001", "k1")
	repo := tenancy.NewRepository(e.app.Pool())
	if _, err := repo.Suspend(e.ctx, tenantID); err != nil {
		t.Fatal(err)
	}
	e.cycle()
	if cur, _ := e.svc.Get(e.ctx, tenantID, m.ID); cur.State != StateQueued {
		t.Errorf("suspended tenant: %s, want queued (not sent, not failed)", cur.State)
	}
	if _, err := repo.Reactivate(e.ctx, tenantID); err != nil {
		t.Fatal(err)
	}
	e.cycle()
	if cur, _ := e.svc.Get(e.ctx, tenantID, m.ID); cur.State != StateDelivered {
		t.Errorf("after reactivation: %s, want delivered", cur.State)
	}
}

func TestIntegrationTooOldToSend(t *testing.T) {
	t.Parallel()
	e := newEnv(t, nil)
	tenantID := e.tenant("Old Co")
	m := e.accept(tenantID, "+639000000001", "k1")
	e.adminExec(`UPDATE messages SET accepted_at = accepted_at - interval '2 hours', status_updated_at = status_updated_at - interval '2 hours'`)
	e.cycle()
	cur, _ := e.svc.Get(e.ctx, tenantID, m.ID)
	if cur.State != StateFailed || cur.Reason == nil || *cur.Reason != ReasonExpiredBeforeSubmission || e.submissionsTo("+639000000001") != 0 {
		t.Errorf("state %s %v, %d submissions; want failed / expired_before_submission, none sent", cur.State, cur.Reason, e.submissionsTo("+639000000001"))
	}
}

// misbehavingProvider panics, or accepts without a reference.
type misbehavingProvider struct {
	*simulator.Simulator
	panic bool
}

func (p *misbehavingProvider) Submit(context.Context, provider.Submission) provider.SubmitResult {
	if p.panic {
		panic("adapter bug")
	}
	return provider.SubmitResult{Outcome: provider.OutcomeAccepted}
}

func TestIntegrationAdapterFailuresAreUncertain(t *testing.T) {
	t.Parallel()
	for _, panics := range []bool{true, false} {
		e := newEnv(t, nil)
		tenantID := e.tenant("Adapter Co")
		m := e.accept(tenantID, "+639000000001", "k1")
		d := e.dispatcher(&misbehavingProvider{Simulator: simulator.New(simulator.Config{}), panic: panics})
		if _, err := d.RunOnce(e.ctx); err != nil {
			t.Fatal(err)
		}
		if cur, _ := e.svc.Get(e.ctx, tenantID, m.ID); cur.State != StateSubmitUncertain {
			t.Errorf("panic=%v: state %s, want submit_uncertain", panics, cur.State)
		}
	}
}
