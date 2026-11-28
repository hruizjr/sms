//go:build integration

package apikey

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hruizjr/sms/backend/internal/audit"
)

// storedLastUsed reads a key's last_used_at as the bootstrap role (outside row-level security).
func (f fixture) storedLastUsed(t *testing.T, keyID [16]byte) *time.Time {
	t.Helper()
	var at *time.Time
	if err := f.admin.Pool().QueryRow(f.ctx, `SELECT last_used_at FROM api_keys WHERE id = $1`, pgUUID(keyID)).Scan(&at); err != nil {
		t.Fatal(err)
	}
	return at
}

// Last-used tracking (owner decision G4-6): written after a successful authentication, at most once
// per key every LastUsedInterval, never for invalid or revoked keys, and only within the key's tenant.
func TestIntegrationLastUsedTracking(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	// A controlled clock after the database's creation time (last_used_at is never before created_at).
	now := time.Now().UTC().Truncate(time.Second).Add(time.Hour)
	var mu sync.Mutex
	clock := func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	advance := func(d time.Duration) { mu.Lock(); defer mu.Unlock(); now = now.Add(d) }
	svc := NewService(f.svc.db, WithClock(clock))

	a := f.activeTenant(t, "Last Used A")
	b := f.activeTenant(t, "Last Used B")
	key, token, err := svc.Create(f.ctx, a, "tracked", []Scope{ScopeMessagesSend}, audit.System, "")
	if err != nil {
		t.Fatal(err)
	}
	// Created keys were never used; the listing says so.
	if key.LastUsedAt != nil || f.storedLastUsed(t, key.ID) != nil {
		t.Fatalf("new key has a last-used time: %v", key.LastUsedAt)
	}

	use := func() Principal {
		t.Helper()
		p, err := svc.Authenticate(f.ctx, token)
		if err != nil {
			t.Fatal(err)
		}
		if err := svc.RecordUse(f.ctx, p); err != nil {
			t.Fatal(err)
		}
		return p
	}

	// 1. First use: recorded at the time of use (UTC).
	if p := use(); p.LastUsedAt != nil {
		t.Errorf("principal before first use carries %v", p.LastUsedAt)
	}
	first := f.storedLastUsed(t, key.ID)
	if first == nil || !first.Equal(now) {
		t.Fatalf("after first use: last_used_at = %v, want %v", first, now)
	}

	// 2. Subsequent uses within the interval write nothing; the principal reports the stored time.
	advance(30 * time.Second)
	if p := use(); p.LastUsedAt == nil || !p.LastUsedAt.Equal(*first) {
		t.Errorf("principal last used = %v, want %v", p.LastUsedAt, first)
	}
	if got := f.storedLastUsed(t, key.ID); !got.Equal(*first) {
		t.Errorf("use within the interval changed last_used_at to %v", got)
	}

	// 3. After the interval, the next use is recorded.
	advance(LastUsedInterval)
	use()
	if got := f.storedLastUsed(t, key.ID); !got.Equal(now) {
		t.Errorf("after the interval: last_used_at = %v, want %v", got, now)
	}

	// 4. Concurrent uses after the interval write at most once (the update re-checks the stored time).
	advance(2 * LastUsedInterval)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			p, err := svc.Authenticate(f.ctx, token)
			if err == nil {
				err = svc.RecordUse(f.ctx, p)
			}
			if err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if got := f.storedLastUsed(t, key.ID); !got.Equal(now) {
		t.Errorf("after concurrent uses: last_used_at = %v, want %v", got, now)
	}
	listed, err := svc.List(f.ctx, a)
	if err != nil || len(listed) != 1 || listed[0].LastUsedAt == nil || !listed[0].LastUsedAt.Equal(now) {
		t.Errorf("List() = %+v, %v; want the key with last used %v", listed, err, now)
	}

	// 5. Invalid keys (unknown, wrong secret, altered tenant part) never authenticate and change nothing.
	before := f.storedLastUsed(t, key.ID)
	advance(2 * LastUsedInterval)
	i := len(token) - 10 // a character inside the secret, replaced by a different one
	wrongSecret := token[:i] + map[bool]string{true: "a", false: "b"}[token[i] != 'a'] + token[i+1:]
	for _, bad := range []string{"smsk_x", wrongSecret, "smsk_" + b.String()} {
		if _, err := svc.Authenticate(f.ctx, bad); !errors.Is(err, ErrUnauthenticated) {
			t.Errorf("Authenticate(%q) = %v, want ErrUnauthenticated", bad, err)
		}
	}
	if got := f.storedLastUsed(t, key.ID); !got.Equal(*before) {
		t.Errorf("invalid keys changed last_used_at to %v", got)
	}

	// 6. Tenant isolation: a principal naming key A's ID under tenant B updates nothing (row-level
	// security and the tenant filter hide key A).
	if err := svc.RecordUse(f.ctx, Principal{TenantID: b, KeyID: key.ID}); err != nil {
		t.Fatal(err)
	}
	if got := f.storedLastUsed(t, key.ID); !got.Equal(*before) {
		t.Errorf("RecordUse under another tenant changed last_used_at to %v", got)
	}

	// 7. Revoked keys: refused, and a request authenticated just before the revocation cannot record
	// its use afterwards.
	stale, err := svc.Authenticate(f.ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Revoke(f.ctx, a, key.ID, audit.System, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate(f.ctx, token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("revoked key authenticated: %v", err)
	}
	if err := svc.RecordUse(f.ctx, stale); err != nil {
		t.Fatal(err)
	}
	if got := f.storedLastUsed(t, key.ID); !got.Equal(*before) {
		t.Errorf("revoked key's last_used_at changed to %v", got)
	}
}

// Creating and revoking a key writes its audit event in the same transaction; nothing secret is in it.
func TestIntegrationKeyChangesAreAudited(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	a := f.activeTenant(t, "Audited Keys")
	actor := [16]byte{0x01, 0x93, 0, 0, 0, 0, 0x70, 0, 0x80} // any non-zero ID: actors are not foreign keys
	key, token, err := f.svc.Create(f.ctx, a, "Audited", []Scope{ScopeMessagesRead}, audit.PortalUser(actor), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Revoke(f.ctx, a, key.ID, audit.System, "smsctl"); err != nil {
		t.Fatal(err)
	}
	// Revoking again changes nothing and records nothing.
	if _, err := f.svc.Revoke(f.ctx, a, key.ID, audit.System, "smsctl"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	rows, err := f.admin.Pool().Query(f.ctx, `
		SELECT action, outcome, actor_type, coalesce(actor_id::text, ''), target_type, target_id, details::text
		FROM security_audit_events WHERE tenant_id = $1 ORDER BY occurred_at, id`, pgUUID(a))
	if err != nil {
		t.Fatal(err)
	}
	type row struct{ action, outcome, actorType, actorID, targetType, details string }
	got, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (row, error) {
		var x row
		var target [16]byte
		err := r.Scan(&x.action, &x.outcome, &x.actorType, &x.actorID, &x.targetType, &target, &x.details)
		if err == nil && target != key.ID {
			err = errors.New("target is not the key")
		}
		return x, err
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []row{
		{"api_key.create", "success", "portal_user", pgUUID(actor).String(), "api_key", `{"api_key_name": "Audited", "api_key_scopes": ["messages:read"]}`},
		{"api_key.revoke", "success", "system", "", "api_key", `{"via": "smsctl", "api_key_name": "Audited"}`},
	}
	if len(got) != len(want) {
		t.Fatalf("audit events = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	secret := token[len(token)-52:]
	var leaked int
	if err := f.admin.Pool().QueryRow(f.ctx, `SELECT count(*) FROM security_audit_events
		WHERE details::text LIKE '%' || $1 || '%' OR details::text LIKE '%smsk_%'`, secret).Scan(&leaked); err != nil || leaked != 0 {
		t.Errorf("audit details contain key material: %d rows, %v", leaked, err)
	}
}
