package portal

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	t.Parallel()
	h := newHasher(HashLimits{Concurrency: 1, Queue: 1, MaxWait: time.Minute})
	ctx := context.Background()
	encoded, err := h.hash(ctx, "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$v=19$m=19456,t=2,p=1$") || strings.Contains(encoded, "correct horse") {
		t.Errorf("hash = %q, want an Argon2id PHC string without the password", encoded)
	}
	if ok, err := h.verify(ctx, "correct horse battery", encoded); !ok || err != nil {
		t.Errorf("verify(correct) = %v, %v", ok, err)
	}
	if ok, err := h.verify(ctx, "correct horse batterY", encoded); ok || err != nil {
		t.Errorf("verify(wrong) = %v, %v", ok, err)
	}
	// Same password, new salt: a different hash.
	again, _ := h.hash(ctx, "correct horse battery")
	if again == encoded {
		t.Error("two hashes of the same password are equal; the salt is not random")
	}
}

func TestParseHashRejectsMalformedAndExcessiveParameters(t *testing.T) {
	t.Parallel()
	// Synthetic, fixed-length values (16-byte salt, 32-byte hash), encoded as in a stored hash.
	salt := base64.RawStdEncoding.EncodeToString(make([]byte, 16))
	key := base64.RawStdEncoding.EncodeToString(make([]byte, 32))
	good := "$argon2id$v=19$m=19456,t=2,p=1$" + salt + "$" + key
	if _, err := parseHash(good); err != nil {
		t.Fatalf("parseHash(good) = %v", err)
	}
	for name, bad := range map[string]string{
		"empty":           "",
		"argon2i":         "$argon2i$v=19$m=19456,t=2,p=1$" + salt + "$" + key,
		"old version":     "$argon2id$v=16$m=19456,t=2,p=1$" + salt + "$" + key,
		"huge memory":     "$argon2id$v=19$m=4194304,t=2,p=1$" + salt + "$" + key,
		"many iterations": "$argon2id$v=19$m=19456,t=1000,p=1$" + salt + "$" + key,
		"zero iterations": "$argon2id$v=19$m=19456,t=0,p=1$" + salt + "$" + key,
		"many lanes":      "$argon2id$v=19$m=19456,t=2,p=255$" + salt + "$" + key,
		"unknown param":   "$argon2id$v=19$m=19456,t=2,p=1,x=1$" + salt + "$" + key,
		"bad base64":      "$argon2id$v=19$m=19456,t=2,p=1$!!!$" + key,
		"short key":       "$argon2id$v=19$m=19456,t=2,p=1$" + salt + "$a2V5",
		"missing part":    "$argon2id$v=19$m=19456,t=2,p=1$" + salt,
	} {
		if _, err := parseHash(bad); !errors.Is(err, errMalformedHash) {
			t.Errorf("%s: parseHash = %v, want errMalformedHash", name, err)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	t.Parallel()
	for _, ok := range []string{"twelve chars", "ññññññññññññ", strings.Repeat("x", MaxPasswordBytes)} {
		if err := validatePassword(ok); err != nil {
			t.Errorf("validatePassword(%q) = %v", ok, err)
		}
	}
	for _, bad := range []string{"", "eleven char", strings.Repeat("x", MaxPasswordBytes+1), "\xff\xfe invalid utf8"} {
		if err := validatePassword(bad); !errors.Is(err, ErrWeakPassword) {
			t.Errorf("validatePassword(%q) = %v, want ErrWeakPassword", bad, err)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"Owner@Example.invalid":     "owner@example.invalid",
		"  user.name+x@example.ph ": "user.name+x@example.ph",
	} {
		if got, err := NormalizeEmail(in); got != want || err != nil {
			t.Errorf("NormalizeEmail(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "a@", "@b", "no-at-sign", "a@b@c", "a b@c.d", "ä@example.invalid", "a@b\x00", strings.Repeat("a", 65) + "@x.y", "a@" + strings.Repeat("b", 260)} {
		if _, err := NormalizeEmail(bad); !errors.Is(err, ErrInvalidEmail) {
			t.Errorf("NormalizeEmail(%q) = %v, want ErrInvalidEmail", bad, err)
		}
	}
}

func TestRolePermissions(t *testing.T) {
	t.Parallel()
	for role, want := range map[Role][3]bool{
		RoleOwner: {true, true, true}, RoleSender: {true, true, false}, RoleViewer: {true, false, false}, "admin": {false, false, false},
	} {
		p := Principal{Role: role}
		got := [3]bool{p.Has(PermissionMessagesRead), p.Has(PermissionMessagesSend), p.Has(PermissionAPIKeysManage)}
		if got != want {
			t.Errorf("%s: read, send, manage API keys = %v, want %v", role, got, want)
		}
	}
	if Role("admin").Valid() || !RoleViewer.Valid() {
		t.Error("Role.Valid is wrong")
	}
}

func TestSessionTokenAndCSRF(t *testing.T) {
	t.Parallel()
	secret := make([]byte, secretBytes)
	secret[0] = 1
	tenant, session := [16]byte{0x01, 0x9a}, [16]byte{0x01, 0x9b}
	token := encoding.EncodeToString(tenant[:]) + "." + encoding.EncodeToString(session[:]) + "." + encoding.EncodeToString(secret)
	gotTenant, gotSession, gotSecret, ok := parseToken(token)
	if !ok || gotTenant != tenant || gotSession != session || string(gotSecret) != string(secret) {
		t.Fatalf("parseToken round trip failed")
	}
	for _, bad := range []string{"", token + ".x", strings.ToUpper(token), strings.Replace(token, ".", "_", 1), token[:len(token)-1]} {
		if _, _, _, ok := parseToken(bad); ok {
			t.Errorf("parseToken(%q) accepted", bad)
		}
	}

	// The CSRF token is derived from the secret, differs from the stored hash form, and is checked
	// exactly.
	csrf := csrfToken(secret)
	other := make([]byte, secretBytes)
	if csrf == csrfToken(other) || csrf == "" {
		t.Fatal("CSRF tokens of different secrets are equal")
	}
	p := Principal{CSRFToken: csrf}
	if !p.CheckCSRF(csrf) || p.CheckCSRF("") || p.CheckCSRF(csrf[:len(csrf)-1]) || p.CheckCSRF(csrfToken(other)) {
		t.Error("CheckCSRF accepted a wrong token or refused the right one")
	}
	if (Principal{}).CheckCSRF("") {
		t.Error("an empty token matched an empty principal")
	}
}

// The hasher never runs more than Concurrency hashes at once, lets at most Queue callers wait, and
// refuses the rest at once with ErrBusy (a distributed flood cannot pile up unbounded waiting work).
func TestHasherBoundsConcurrencyAndQueue(t *testing.T) {
	t.Parallel()
	h := newHasher(HashLimits{Concurrency: 2, Queue: 3, MaxWait: 5 * time.Second})
	ctx := context.Background()
	// Hold both slots.
	for range 2 {
		if err := h.acquire(ctx); err != nil {
			t.Fatal(err)
		}
	}
	// Three callers queue; the next ones are refused immediately.
	var queued sync.WaitGroup
	results := make(chan error, 3)
	for range 3 {
		queued.Add(1)
		go func() {
			queued.Done()
			results <- h.acquire(ctx)
		}()
	}
	queued.Wait()
	deadline := time.Now().Add(5 * time.Second)
	for len(h.queue) < 3 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond) // waiting for the goroutines to enter the queue (test only)
	}
	start := time.Now()
	for range 10 {
		if err := h.acquire(ctx); !errors.Is(err, ErrBusy) {
			t.Fatalf("acquire with a full queue = %v, want ErrBusy", err)
		}
	}
	if time.Since(start) > time.Second {
		t.Error("refusals at a full queue were not immediate")
	}
	// Releasing the slots admits the queued callers, two at a time.
	h.release()
	h.release()
	for range 2 {
		if err := <-results; err != nil {
			t.Fatalf("queued acquire = %v", err)
		}
	}
	h.release()
	if err := <-results; err != nil {
		t.Fatalf("queued acquire = %v", err)
	}
	h.release()
	h.release()
}

func TestHasherMaxWaitAndCancellation(t *testing.T) {
	t.Parallel()
	h := newHasher(HashLimits{Concurrency: 1, Queue: 4, MaxWait: 50 * time.Millisecond})
	if err := h.acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer h.release()
	start := time.Now()
	if err := h.acquire(context.Background()); !errors.Is(err, ErrBusy) {
		t.Fatalf("acquire after MaxWait = %v, want ErrBusy", err)
	}
	if d := time.Since(start); d < 40*time.Millisecond || d > 2*time.Second {
		t.Errorf("waited %v, want about 50ms", d)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := h.acquire(ctx); !errors.Is(err, ErrBusy) || !errors.Is(err, context.Canceled) {
		t.Errorf("acquire with a cancelled context = %v, want ErrBusy wrapping context.Canceled", err)
	}
	if len(h.queue) != 0 {
		t.Errorf("queue holds %d callers after they left", len(h.queue))
	}
}

// Many concurrent verifications: the number running at once never exceeds Concurrency.
func TestHasherConcurrentLoad(t *testing.T) {
	t.Parallel()
	limits := HashLimits{Concurrency: 2, Queue: 4, MaxWait: 10 * time.Second}
	h := newHasher(limits)
	ctx := context.Background()
	encoded, err := h.hash(ctx, "synthetic password one")
	if err != nil {
		t.Fatal(err)
	}
	var running, peak, ok, busy atomic.Int64
	var wg sync.WaitGroup
	for range 40 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.acquire(ctx); err != nil {
				if errors.Is(err, ErrBusy) {
					busy.Add(1)
				}
				return
			}
			n := running.Add(1)
			for p := peak.Load(); n > p && !peak.CompareAndSwap(p, n); p = peak.Load() {
			}
			running.Add(-1)
			h.release()
			if _, err := h.verify(ctx, "synthetic password one", encoded); err == nil {
				ok.Add(1)
			} else if errors.Is(err, ErrBusy) {
				busy.Add(1)
			}
		}()
	}
	wg.Wait()
	if peak.Load() > int64(limits.Concurrency) {
		t.Errorf("peak concurrency %d > %d", peak.Load(), limits.Concurrency)
	}
	if ok.Load()+busy.Load() == 0 {
		t.Error("nothing ran")
	}
	if len(h.slots) != 0 || len(h.queue) != 0 {
		t.Errorf("slots %d, queue %d left after the load", len(h.slots), len(h.queue))
	}
}

func TestHashLimitsValidate(t *testing.T) {
	t.Parallel()
	if err := DefaultHashLimits.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, l := range []HashLimits{{0, 1, time.Second}, {1, -1, time.Second}, {1, 1, 0}, {257, 1, time.Second}} {
		if l.Validate() == nil {
			t.Errorf("Validate(%+v) succeeded", l)
		}
	}
}
