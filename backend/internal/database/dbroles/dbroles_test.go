package dbroles

import (
	"regexp"
	"strings"
	"testing"
)

// The provisioning script is the single definition of the roles. These checks keep it free of
// secrets and of anything destructive, and keep it in step with the role names used in Go.
func TestProvisionSQL(t *testing.T) {
	t.Parallel()
	sql := ProvisionSQL
	for _, want := range []string{Migrator, App, "NOSUPERUSER", "NOBYPASSRLS", "NOCREATEROLE", "NOCREATEDB",
		"REVOKE ALL ON SCHEMA public FROM PUBLIC", "GRANT USAGE ON SCHEMA public TO sms_app"} {
		if !strings.Contains(sql, want) {
			t.Errorf("provision.sql does not contain %q", want)
		}
	}
	// Code only (comments removed): no password, no destructive statement, no grant of CREATE,
	// ownership or TRUNCATE to the application role, no role membership.
	var code strings.Builder
	for line := range strings.SplitSeq(sql, "\n") {
		if i := strings.Index(line, "--"); i >= 0 {
			line = line[:i]
		}
		code.WriteString(line + "\n")
	}
	forbidden := regexp.MustCompile(`(?i)\bPASSWORD\b|\bDROP\b|\bTRUNCATE\b|\bDELETE\b|OWNER TO sms_app|` +
		`GRANT\s+[^;]*\bCREATE\b[^;]*\bTO\s+sms_app|GRANT\s+sms_\w+\s+TO`)
	for _, m := range forbidden.FindAllString(code.String(), -1) {
		t.Errorf("provision.sql contains forbidden code %q", m)
	}
	// Every CREATE ROLE and ALTER ROLE sets exactly the unprivileged attributes.
	const attributes = "LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS"
	roleStatements := regexp.MustCompile(`(?i)'(CREATE|ALTER) ROLE %I ([^']*)'\s*'([^']*)'`).FindAllStringSubmatch(code.String(), -1)
	if len(roleStatements) != 2 {
		t.Fatalf("found %d CREATE/ALTER ROLE statements, want 2", len(roleStatements))
	}
	for _, m := range roleStatements {
		if got := strings.Join(strings.Fields(m[2]+" "+m[3]), " "); got != attributes {
			t.Errorf("%s ROLE attributes = %q, want %q", m[1], got, attributes)
		}
	}
}
