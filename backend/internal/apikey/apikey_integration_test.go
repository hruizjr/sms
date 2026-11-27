//go:build integration

package apikey

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/dbtest"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

type fixture struct {
	ctx     context.Context
	svc     *Service
	admin   *database.DB
	tenants *tenancy.Repository
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	roles := dbtest.NewMigratedDatabase(t)
	app, err := database.Open(ctx, roles.App, "sms-apikey-test", database.RequireApplicationRole())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Close)
	admin, err := database.Open(ctx, roles.Admin, "sms-apikey-test-admin")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	return fixture{ctx: ctx, svc: NewService(app), admin: admin, tenants: tenancy.NewRepository(app.Pool())}
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

func TestIntegrationCreateAndAuthenticate(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tenantID := f.activeTenant(t, "Key Co")
	key, token, err := f.svc.Create(f.ctx, tenantID, "ci key", []Scope{ScopeMessagesSend})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(token, key.Prefix()+"_") || len(token) < 100 {
		t.Errorf("token %q does not start with its visible prefix %q", token, key.Prefix())
	}
	p, err := f.svc.Authenticate(f.ctx, token)
	if err != nil || p.TenantID != tenantID || p.KeyID != key.ID || !p.Has(ScopeMessagesSend) || p.Has(ScopeMessagesRead) {
		t.Fatalf("Authenticate() = %+v, %v", p, err)
	}

	// Only the SHA-256 hash of the secret is stored; neither the key nor its secret appear in the
	// database.
	secret := token[strings.LastIndex(token, "_")+1:]
	raw, err := decodeCanonical(secret)
	if err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256(raw)
	var stored []byte
	var asText string
	if err := f.admin.Pool().QueryRow(f.ctx, `SELECT secret_hash, row_to_json(k)::text FROM api_keys k WHERE id = $1`, key.ID[:]).Scan(&stored, &asText); err != nil {
		t.Fatal(err)
	}
	if string(stored) != string(want[:]) {
		t.Error("stored hash is not the SHA-256 of the secret")
	}
	if strings.Contains(asText, secret) || strings.Contains(asText, token) {
		t.Error("the plain-text key is stored")
	}
}

func TestIntegrationAuthenticationRefusals(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	a, b := f.activeTenant(t, "Tenant A"), f.activeTenant(t, "Tenant B")
	key, token, err := f.svc.Create(f.ctx, a, "a key", []Scope{ScopeMessagesSend, ScopeMessagesRead})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(strings.TrimPrefix(token, tokenPrefix), "_")
	otherSecret := encoding.EncodeToString(make([]byte, secretBytes))

	for name, bad := range map[string]string{
		"empty":                      "",
		"wrong prefix":               "sk_" + strings.TrimPrefix(token, tokenPrefix),
		"truncated":                  token[:len(token)-1],
		"wrong secret":               tokenPrefix + parts[0] + "_" + parts[1] + "_" + otherSecret,
		"tenant part swapped":        publicPart(b, key.ID) + "_" + parts[2],
		"non-canonical upper case":   strings.ToUpper(token),
		"extra part":                 token + "_x",
		"nil tenant":                 publicPart(tenancy.ID{}, key.ID) + "_" + parts[2],
		"unknown key of real tenant": publicPart(a, [16]byte{1, 2, 3, 4, 5, 6, 0x70, 8, 0x80}) + "_" + parts[2],
	} {
		if _, err := f.svc.Authenticate(f.ctx, bad); !errors.Is(err, ErrUnauthenticated) {
			t.Errorf("%s: Authenticate() = %v, want ErrUnauthenticated", name, err)
		}
	}

	// Revocation takes effect immediately and only once.
	if _, err := f.svc.Revoke(f.ctx, a, key.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Authenticate(f.ctx, token); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("revoked key: %v, want ErrUnauthenticated", err)
	}
	if _, err := f.svc.Revoke(f.ctx, a, key.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second revocation: %v, want ErrNotFound", err)
	}
	// Another tenant cannot revoke or list the key.
	_, tokenA2, err := f.svc.Create(f.ctx, a, "a key 2", []Scope{ScopeMessagesRead})
	if err != nil {
		t.Fatal(err)
	}
	p, _ := f.svc.Authenticate(f.ctx, tokenA2)
	if _, err := f.svc.Revoke(f.ctx, b, p.KeyID); !errors.Is(err, ErrNotFound) {
		t.Errorf("tenant B revoking tenant A's key: %v, want ErrNotFound", err)
	}
	if keys, err := f.svc.List(f.ctx, b); err != nil || len(keys) != 0 {
		t.Errorf("tenant B lists %d keys, %v", len(keys), err)
	}

	// A valid key of a suspended tenant is refused with ErrTenantNotActive (HTTP 403).
	if _, err := f.tenants.Suspend(f.ctx, a); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Authenticate(f.ctx, tokenA2); !errors.Is(err, ErrTenantNotActive) {
		t.Errorf("suspended tenant: %v, want ErrTenantNotActive", err)
	}
}

func TestIntegrationCreateValidatesScopes(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tenantID := f.activeTenant(t, "Scope Co")
	for _, scopes := range [][]Scope{nil, {"messages:delete"}, {ScopeMessagesSend, "admin"}} {
		if _, _, err := f.svc.Create(f.ctx, tenantID, "k", scopes); !errors.Is(err, ErrInvalidScopes) {
			t.Errorf("scopes %v: %v, want ErrInvalidScopes", scopes, err)
		}
	}
	k, _, err := f.svc.Create(f.ctx, tenantID, "k", []Scope{ScopeMessagesRead, ScopeMessagesRead})
	if err != nil || len(k.Scopes) != 1 {
		t.Errorf("duplicate scopes: %+v, %v; want one scope", k.Scopes, err)
	}
}

func TestIntegrationCreateValidatesLabel(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	tenantID := f.activeTenant(t, "Label Co")
	for _, label := range []string{"", "   ", strings.Repeat("x", MaxLabelCharacters+1), "tab\there", "line\nbreak", "\xff"} {
		if _, _, err := f.svc.Create(f.ctx, tenantID, label, []Scope{ScopeMessagesSend}); !errors.Is(err, ErrInvalidLabel) {
			t.Errorf("label %q: %v, want ErrInvalidLabel", label, err)
		}
	}
	k, _, err := f.svc.Create(f.ctx, tenantID, "  "+strings.Repeat("ñ", MaxLabelCharacters)+" ", []Scope{ScopeMessagesSend})
	if err != nil || k.Label != strings.Repeat("ñ", MaxLabelCharacters) {
		t.Errorf("100-character label: %q, %v", k.Label, err)
	}
}
