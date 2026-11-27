package portal

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	t.Parallel()
	h := newHasher(1)
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
	for role, want := range map[Role][2]bool{RoleOwner: {true, true}, RoleSender: {true, true}, RoleViewer: {true, false}, "admin": {false, false}} {
		p := Principal{Role: role}
		if p.Has(PermissionMessagesRead) != want[0] || p.Has(PermissionMessagesSend) != want[1] {
			t.Errorf("%s: read=%v send=%v, want %v", role, p.Has(PermissionMessagesRead), p.Has(PermissionMessagesSend), want)
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
