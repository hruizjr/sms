// Command smsctl is the LOCAL administration tool of the SMS platform (Phase 4 Group 2). It creates
// and activates tenants, approves sender IDs, issues and revokes API keys, and shows a message's
// internal audit trail. There is no public API for any of this: API key and sender ID management
// endpoints are later work.
//
// It refuses to run unless APP_ENV is explicitly "development" or "test", and it connects as the
// application role (sms_app), so it is subject to row-level security like the API. API keys are
// printed once and never stored in plain text.
//
// Usage:
//
//	smsctl bootstrap --name NAME --sender-id ID [--label LABEL]
//	smsctl tenant create --name NAME            (created and activated)
//	smsctl tenant list
//	smsctl tenant suspend|reactivate --tenant TENANT_ID
//	smsctl sender-id approve --tenant TENANT_ID --sender-id ID
//	smsctl api-key create --tenant TENANT_ID [--label LABEL] [--scopes messages:send,messages:read]
//	smsctl api-key list --tenant TENANT_ID
//	smsctl api-key revoke --tenant TENANT_ID --key-id KEY_ID
//	smsctl message trace --tenant TENANT_ID --message-id msg_...
//	smsctl portal-user create --tenant TENANT_ID --email EMAIL --role owner|sender|viewer [--password-stdin]
//	smsctl portal-user list --tenant TENANT_ID
//	smsctl portal-user unlock --tenant TENANT_ID --email EMAIL
//	smsctl portal-user revoke-sessions --tenant TENANT_ID --email EMAIL
//
// Portal users (Phase 4 Group 3; owner decision G3-4): there is no self-registration and no user
// management endpoint. "portal-user create" generates a random password and prints it once, or reads
// the password from standard input with --password-stdin (never from a command-line flag, which
// would be visible in the process list and shell history). Only its Argon2id hash is stored.
package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hruizjr/sms/backend/internal/apikey"
	"github.com/hruizjr/sms/backend/internal/audit"
	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/messaging"
	"github.com/hruizjr/sms/backend/internal/messaging/provider"
	"github.com/hruizjr/sms/backend/internal/portal"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

const usage = `usage:
  smsctl bootstrap --name NAME --sender-id ID [--label LABEL]
  smsctl tenant create --name NAME
  smsctl tenant list
  smsctl tenant suspend|reactivate --tenant TENANT_ID
  smsctl sender-id approve --tenant TENANT_ID --sender-id ID
  smsctl api-key create --tenant TENANT_ID [--label LABEL] [--scopes messages:send,messages:read]
  smsctl api-key list --tenant TENANT_ID
  smsctl api-key revoke --tenant TENANT_ID --key-id KEY_ID
  smsctl message trace --tenant TENANT_ID --message-id msg_...
  smsctl portal-user create --tenant TENANT_ID --email EMAIL --role owner|sender|viewer [--password-stdin]
  smsctl portal-user list --tenant TENANT_ID
  smsctl portal-user unlock --tenant TENANT_ID --email EMAIL
  smsctl portal-user revoke-sessions --tenant TENANT_ID --email EMAIL
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.LookupEnv, os.Stdin, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

type cli struct {
	tenants  *tenancy.Repository
	keys     *apikey.Service
	messages *messaging.Service
	portal   *portal.Service
	in       io.Reader
	out      io.Writer
}

func run(ctx context.Context, args []string, lookup config.LookupFunc, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		say(stderr, usage)
		return exitUsage
	}
	env, explicit := config.ExplicitEnvironment(lookup)
	if !explicit || (env != config.EnvironmentDevelopment && env != config.EnvironmentTest) {
		sayln(stderr, "smsctl: refusing to run: it is a local development tool; set APP_ENV=development or APP_ENV=test")
		return exitUsage
	}
	cfg, err := config.Load(lookup)
	if err != nil {
		sayln(stderr, "smsctl:", err)
		return exitError
	}
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	db, err := database.Open(ctx, cfg.Database, "sms-smsctl", database.RequireApplicationRole())
	if err != nil {
		sayln(stderr, "smsctl:", err)
		return exitError
	}
	defer db.Close()
	// Portal user administration is recorded in the audit log as system actions via smsctl.
	accounts, err := portal.NewService(db, portal.WithAdministrationTool("smsctl"))
	if err != nil {
		sayln(stderr, "smsctl:", err)
		return exitError
	}
	c := &cli{
		tenants:  tenancy.NewRepository(db.Pool()),
		keys:     apikey.NewService(db),
		messages: messaging.NewService(db, provider.Capabilities{}),
		portal:   accounts,
		in:       stdin,
		out:      stdout,
	}
	if err := c.dispatch(ctx, args); err != nil {
		var u usageError
		if errors.As(err, &u) {
			say(stderr, "smsctl: %s\n%s", u.msg, usage)
			return exitUsage
		}
		sayln(stderr, "smsctl:", err)
		return exitError
	}
	return exitOK
}

type usageError struct{ msg string }

func (u usageError) Error() string { return u.msg }

func (c *cli) dispatch(ctx context.Context, args []string) error {
	cmd := args[0]
	if cmd != "bootstrap" {
		if len(args) < 2 {
			return usageError{"missing subcommand"}
		}
		cmd += " " + args[1]
		args = args[1:]
	}
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "tenant display name")
	tenantFlag := fs.String("tenant", "", "tenant ID")
	sender := fs.String("sender-id", "", "sender ID")
	label := fs.String("label", "local development key", "API key label")
	scopes := fs.String("scopes", "messages:send,messages:read", "comma-separated API key scopes")
	keyID := fs.String("key-id", "", "API key ID")
	messageID := fs.String("message-id", "", "message ID (msg_...)")
	email := fs.String("email", "", "portal user email address")
	role := fs.String("role", "", "portal user role: owner, sender or viewer")
	passwordStdin := fs.Bool("password-stdin", false, "read the portal user's password from standard input")
	if err := fs.Parse(args[1:]); err != nil {
		return usageError{err.Error()}
	}
	if fs.NArg() > 0 {
		return usageError{"unexpected argument " + fs.Arg(0)}
	}
	tenantID := func() (tenancy.ID, error) {
		if *tenantFlag == "" {
			return tenancy.ID{}, usageError{"--tenant is required"}
		}
		return tenancy.ParseID(*tenantFlag)
	}

	switch cmd {
	case "bootstrap":
		if *name == "" || *sender == "" {
			return usageError{"--name and --sender-id are required"}
		}
		t, err := c.createActiveTenant(ctx, *name)
		if err != nil {
			return err
		}
		if err := c.messages.ApproveSenderID(ctx, t.ID, *sender); err != nil {
			return err
		}
		say(c.out, "sender ID approved: %s\n", *sender)
		return c.createKey(ctx, t.ID, *label, *scopes)
	case "tenant create":
		if *name == "" {
			return usageError{"--name is required"}
		}
		_, err := c.createActiveTenant(ctx, *name)
		return err
	case "tenant list":
		page, err := c.tenants.List(ctx, tenancy.ID{}, tenancy.MaxPageSize)
		for _, t := range page {
			say(c.out, "%s\t%s\t%s\n", t.ID, t.Status, t.DisplayName)
		}
		return err
	case "tenant suspend", "tenant reactivate":
		id, err := tenantID()
		if err != nil {
			return err
		}
		var t tenancy.Tenant
		if cmd == "tenant suspend" {
			t, err = c.tenants.Suspend(ctx, id)
		} else {
			t, err = c.tenants.Reactivate(ctx, id)
		}
		if err == nil {
			say(c.out, "tenant %s is %s\n", t.ID, t.Status)
		}
		return err
	case "sender-id approve":
		id, err := tenantID()
		if err != nil {
			return err
		}
		if *sender == "" {
			return usageError{"--sender-id is required"}
		}
		if err := c.messages.ApproveSenderID(ctx, id, *sender); err != nil {
			return err
		}
		say(c.out, "sender ID approved: %s\n", *sender)
		return nil
	case "api-key create":
		id, err := tenantID()
		if err != nil {
			return err
		}
		return c.createKey(ctx, id, *label, *scopes)
	case "api-key list":
		id, err := tenantID()
		if err != nil {
			return err
		}
		keys, err := c.keys.List(ctx, id)
		for _, k := range keys {
			state := "active"
			if k.RevokedAt != nil {
				state = "revoked " + k.RevokedAt.UTC().Format(time.RFC3339)
			}
			say(c.out, "%s\t%s\t%s\t%s\t%s\n", k.IDString(), k.Prefix(), scopeList(k.Scopes), state, k.Label)
		}
		return err
	case "api-key revoke":
		id, err := tenantID()
		if err != nil {
			return err
		}
		kid, err := apikey.ParseKeyID(*keyID)
		if err != nil {
			return err
		}
		k, err := c.keys.Revoke(ctx, id, kid, audit.System, smsctlName)
		if err == nil {
			say(c.out, "api key %s revoked at %s\n", k.IDString(), k.RevokedAt.UTC().Format(time.RFC3339))
		}
		return err
	case "message trace":
		id, err := tenantID()
		if err != nil {
			return err
		}
		mid, err := messaging.ParseMessageID(*messageID)
		if err != nil {
			return usageError{"--message-id must be a message ID (msg_...)"}
		}
		tr, err := c.messages.Trace(ctx, id, mid)
		if err != nil {
			return err
		}
		c.printTrace(tr)
		return nil
	case "portal-user create":
		id, err := tenantID()
		if err != nil {
			return err
		}
		if *email == "" || *role == "" {
			return usageError{"--email and --role are required"}
		}
		return c.createPortalUser(ctx, id, *email, portal.Role(*role), *passwordStdin)
	case "portal-user list":
		id, err := tenantID()
		if err != nil {
			return err
		}
		users, err := c.portal.ListUsers(ctx, id)
		for _, u := range users {
			state := "active"
			if u.LockedUntil != nil && u.LockedUntil.After(time.Now()) {
				state = "locked until " + u.LockedUntil.UTC().Format(time.RFC3339)
			}
			say(c.out, "%s\t%s\t%s\n", u.Email, u.Role, state)
		}
		return err
	case "portal-user unlock":
		id, err := tenantID()
		if err != nil {
			return err
		}
		if *email == "" {
			return usageError{"--email is required"}
		}
		if err := c.portal.Unlock(ctx, id, *email); err != nil {
			return err
		}
		say(c.out, "portal user %s unlocked\n", *email)
		return nil
	case "portal-user revoke-sessions":
		id, err := tenantID()
		if err != nil {
			return err
		}
		if *email == "" {
			return usageError{"--email is required"}
		}
		n, err := c.portal.RevokeUserSessions(ctx, id, *email)
		if err == nil {
			say(c.out, "%d session(s) of %s revoked\n", n, *email)
		}
		return err
	}
	return usageError{"unknown command " + cmd}
}

// createPortalUser creates a portal user with a generated password (printed once) or a password read
// from standard input.
func (c *cli) createPortalUser(ctx context.Context, tenantID tenancy.ID, email string, role portal.Role, fromStdin bool) error {
	var password string
	if fromStdin {
		line, err := bufio.NewReader(io.LimitReader(c.in, portal.MaxPasswordBytes+2)).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("read password from standard input: %w", err)
		}
		password = strings.TrimRight(line, "\r\n")
	} else {
		b := make([]byte, 20)
		if _, err := rand.Read(b); err != nil {
			return fmt.Errorf("generate password: %w", err)
		}
		password = generatedPasswordEncoding.EncodeToString(b)
	}
	u, err := c.portal.CreateUser(ctx, tenantID, email, role, password)
	if err != nil {
		return err
	}
	say(c.out, "portal user: %s (role %s)\n", u.Email, u.Role)
	if !fromStdin {
		say(c.out, "password (shown once; it is not stored and cannot be shown again):\n%s\n", password)
	}
	return nil
}

// generatedPasswordEncoding avoids characters that are easy to confuse (Crockford base32).
var generatedPasswordEncoding = base32.NewEncoding("0123456789abcdefghjkmnpqrstvwxyz").WithPadding(base32.NoPadding)

func (c *cli) createActiveTenant(ctx context.Context, name string) (tenancy.Tenant, error) {
	t, err := c.tenants.Create(ctx, name)
	if err != nil {
		return t, err
	}
	if t, err = c.tenants.Activate(ctx, t.ID); err != nil {
		return t, err
	}
	say(c.out, "tenant: %s (%s)\n", t.ID, t.Status)
	return t, nil
}

// smsctlName names this tool in the audit events of the API keys it creates and revokes (actor
// "system", details.via).
const smsctlName = "smsctl"

func (c *cli) createKey(ctx context.Context, tenantID tenancy.ID, label, scopes string) error {
	var list []apikey.Scope
	for _, s := range strings.Split(scopes, ",") {
		if s = strings.TrimSpace(s); s != "" {
			list = append(list, apikey.Scope(s))
		}
	}
	k, token, err := c.keys.Create(ctx, tenantID, label, list, audit.System, smsctlName)
	if err != nil {
		return err
	}
	say(c.out, "api key id: %s (scopes %s)\n", k.IDString(), scopeList(k.Scopes))
	say(c.out, "api key (shown once; it is not stored and cannot be shown again):\n%s\n", token)
	return nil
}

func (c *cli) printTrace(t messaging.Trace) {
	m := t.Message
	say(c.out, "message %s: state %s (status %s)", m.ID, m.State, m.State.Status())
	if m.Reason != nil {
		say(c.out, ", reason %s", *m.Reason)
	}
	say(c.out, "\n  encoding %s, %d segment(s), accepted %s\n", m.Encoding, m.SegmentCount, m.AcceptedAt.Format(time.RFC3339Nano))
	sayln(c.out, "status history:")
	for _, h := range t.History {
		from := "-"
		if h.From != nil {
			from = string(*h.From)
		}
		reason := ""
		if h.Reason != nil {
			reason = " (" + string(*h.Reason) + ")"
		}
		say(c.out, "  %s  %-16s -> %-16s by %s%s\n", h.OccurredAt.Format(time.RFC3339Nano), from, h.To, h.Source, reason)
	}
	sayln(c.out, "submission attempts:")
	for _, a := range t.Attempts {
		ref, code := "-", "-"
		if a.ProviderMessageRef != nil {
			ref = *a.ProviderMessageRef
		}
		if a.ProviderResultCode != nil {
			code = *a.ProviderResultCode
		}
		say(c.out, "  #%d provider %s: %s (provider code %s, provider ref %s)\n", a.Number, a.ProviderID, a.State, code, ref)
	}
	sayln(c.out, "receipts:")
	for _, r := range t.Receipts {
		say(c.out, "  %s  %s: %s\n", r.ProcessedAt.Format(time.RFC3339Nano), r.Status, r.Effect)
	}
}

func scopeList(scopes []apikey.Scope) string {
	names := make([]string, len(scopes))
	for i, s := range scopes {
		names[i] = string(s)
	}
	return strings.Join(names, ",")
}

// say and sayln write command output. A failed write to the terminal cannot be reported anywhere
// more useful, so the error is deliberately ignored.
func say(w io.Writer, format string, args ...any) { _, _ = fmt.Fprintf(w, format, args...) }

func sayln(w io.Writer, args ...any) { _, _ = fmt.Fprintln(w, args...) }
