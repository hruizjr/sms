//go:build integration

package main

import (
	"bytes"
	"context"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hruizjr/sms/backend/internal/apikey"
	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/dbtest"
	"github.com/hruizjr/sms/backend/internal/portal"
)

func TestIntegrationBootstrapIssuesAWorkingKeyAndRevokes(t *testing.T) {
	t.Parallel()
	roles := dbtest.NewMigratedDatabase(t)
	cfg := roles.App
	lookup := func(key string) (string, bool) {
		switch key {
		case config.EnvDBName:
			return cfg.Name, true
		case config.EnvDBUser:
			return cfg.User, true
		case config.EnvDBPassword:
			return cfg.Password.Reveal(), true
		case config.EnvAppEnv:
			return "test", true
		}
		return os.LookupEnv(key)
	}
	smsctlIn := func(stdin string, args ...string) (int, string) {
		t.Helper()
		var out, errOut bytes.Buffer
		code := run(context.Background(), args, lookup, strings.NewReader(stdin), &out, &errOut)
		if strings.Contains(out.String()+errOut.String(), cfg.Password.Reveal()) {
			t.Fatal("smsctl printed the database password")
		}
		return code, out.String() + errOut.String()
	}
	smsctl := func(args ...string) (int, string) {
		t.Helper()
		return smsctlIn("", args...)
	}

	code, out := smsctl("bootstrap", "--name", "Local Demo Co", "--sender-id", "EXAMPLECO")
	if code != exitOK {
		t.Fatalf("bootstrap: %d %s", code, out)
	}
	tenantID := regexp.MustCompile(`tenant: ([0-9a-f-]{36}) \(active\)`).FindStringSubmatch(out)
	token := regexp.MustCompile(`(?m)^(smsk_\S+)$`).FindStringSubmatch(out)
	keyID := regexp.MustCompile(`api key id: (\S+)`).FindStringSubmatch(out)
	if tenantID == nil || token == nil || keyID == nil || !strings.Contains(out, "sender ID approved: EXAMPLECO") {
		t.Fatalf("bootstrap output: %s", out)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	db, err := database.Open(ctx, cfg, "sms-smsctl-test")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	keys := apikey.NewService(db)
	p, err := keys.Authenticate(ctx, token[1])
	if err != nil || !p.Has(apikey.ScopeMessagesSend) || !p.Has(apikey.ScopeMessagesRead) {
		t.Fatalf("issued key: %+v, %v", p, err)
	}

	if code, out = smsctl("api-key", "list", "--tenant", tenantID[1]); code != exitOK || !strings.Contains(out, keyID[1]) || strings.Contains(out, token[1]) {
		t.Errorf("api-key list: %d %s", code, out)
	}
	if code, out = smsctl("api-key", "revoke", "--tenant", tenantID[1], "--key-id", keyID[1]); code != exitOK {
		t.Fatalf("revoke: %d %s", code, out)
	}
	if _, err := keys.Authenticate(ctx, token[1]); err == nil {
		t.Error("revoked key still authenticates")
	}
	if code, out = smsctl("message", "trace", "--tenant", tenantID[1], "--message-id", "msg_00000000000000000000000000"); code != exitError {
		t.Errorf("trace of an unknown message: %d %s", code, out)
	}
	if code, _ = smsctl("api-key", "create"); code != exitUsage {
		t.Errorf("missing --tenant: exit %d, want usage", code)
	}

	// Portal users: a generated password printed once, or one read from standard input.
	accounts, err := portal.NewService(db)
	if err != nil {
		t.Fatal(err)
	}
	code, out = smsctl("portal-user", "create", "--tenant", tenantID[1], "--email", "Owner@Example.invalid", "--role", "owner")
	password := regexp.MustCompile(`(?m)^([0-9a-z]{32})$`).FindStringSubmatch(out)
	if code != exitOK || password == nil || !strings.Contains(out, "portal user: owner@example.invalid (role owner)") {
		t.Fatalf("portal-user create: %d %s", code, out)
	}
	if _, err := accounts.Login(ctx, "owner@example.invalid", password[1]); err != nil {
		t.Errorf("sign-in with the printed password: %v", err)
	}
	const typed = "a typed password 123"
	if code, out = smsctlIn(typed+"\n", "portal-user", "create", "--tenant", tenantID[1], "--email", "viewer@example.invalid", "--role", "viewer", "--password-stdin"); code != exitOK || strings.Contains(out, typed) {
		t.Fatalf("portal-user create --password-stdin: %d %s", code, out)
	}
	if _, err := accounts.Login(ctx, "viewer@example.invalid", typed); err != nil {
		t.Errorf("sign-in with the password from standard input: %v", err)
	}
	if code, out = smsctlIn("short\n", "portal-user", "create", "--tenant", tenantID[1], "--email", "weak@example.invalid", "--role", "viewer", "--password-stdin"); code != exitError {
		t.Errorf("weak password accepted: %d %s", code, out)
	}
	if code, out = smsctl("portal-user", "create", "--tenant", tenantID[1], "--email", "x@example.invalid", "--role", "admin"); code != exitError {
		t.Errorf("unknown role accepted: %d %s", code, out)
	}
	if code, out = smsctl("portal-user", "list", "--tenant", tenantID[1]); code != exitOK ||
		!strings.Contains(out, "owner@example.invalid\towner\tactive") || !strings.Contains(out, "viewer@example.invalid\tviewer") || strings.Contains(out, "argon2") {
		t.Errorf("portal-user list: %d %s", code, out)
	}
	if code, out = smsctl("portal-user", "revoke-sessions", "--tenant", tenantID[1], "--email", "owner@example.invalid"); code != exitOK || !strings.Contains(out, "1 session(s) of owner@example.invalid revoked") {
		t.Errorf("portal-user revoke-sessions: %d %s", code, out)
	}
	if code, out = smsctl("portal-user", "unlock", "--tenant", tenantID[1], "--email", "owner@example.invalid"); code != exitOK {
		t.Errorf("portal-user unlock: %d %s", code, out)
	}
}
