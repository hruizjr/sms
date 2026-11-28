//go:build integration

package messaging

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/hruizjr/sms/backend/internal/tenancy"
)

// A sender ID approval is recorded in the organization's audit log in the same transaction, as the
// system actor with the administration tool named (no operator identity is invented); an approval
// that changes nothing records nothing; another organization sees nothing; and without its event
// the approval does not happen.
func TestIntegrationSenderIDApprovalIsAudited(t *testing.T) {
	t.Parallel()
	e := newEnv(t, nil)
	repo := tenancy.NewRepository(e.app.Pool())
	newTenant := func(name string) tenancy.ID {
		tn, err := repo.Create(e.ctx, name)
		if err != nil {
			t.Fatal(err)
		}
		return tn.ID
	}
	a, b := newTenant("Approved Co"), newTenant("Other Co")
	events := func(id tenancy.ID) []string {
		t.Helper()
		rows, err := e.admin.Pool().Query(e.ctx, `SELECT action || ' ' || outcome || ' ' || actor_type || ' ' || coalesce(actor_id::text, '-') || ' ' ||
			coalesce(target_type, '-') || ' ' || coalesce(details->>'sender_id', '-') || ' ' || coalesce(details->>'via', '-')
			FROM security_audit_events WHERE tenant_id = $1 ORDER BY occurred_at, id`, id)
		if err != nil {
			t.Fatal(err)
		}
		got, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			t.Fatal(err)
		}
		return got
	}

	if err := e.svc.ApproveSenderID(e.ctx, a, "EXAMPLECO", "smsctl"); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.ApproveSenderID(e.ctx, a, "EXAMPLECO", "smsctl"); err != nil { // already approved
		t.Fatal(err)
	}
	want := "sender_id.approve success system - - EXAMPLECO smsctl"
	if got := events(a); strings.Join(got, "\n") != want {
		t.Errorf("events of the organization:\n%s\nwant one:\n%s", strings.Join(got, "\n"), want)
	}
	if got := events(b); len(got) != 0 {
		t.Errorf("another organization has events: %v", got)
	}

	// With the audit insert impossible, the approval is rolled back with it.
	if _, err := e.admin.Pool().Exec(e.ctx, `REVOKE INSERT ON security_audit_events FROM sms_app`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = e.admin.Pool().Exec(e.ctx, `GRANT INSERT (tenant_id, action, outcome, actor_type, actor_id, target_type,
			target_id, request_id, source_ip, user_agent, details, dedup_key) ON security_audit_events TO sms_app`) //nolint:errcheck // test teardown
	})
	if err := e.svc.ApproveSenderID(e.ctx, b, "UNAUDITED", "smsctl"); err == nil {
		t.Fatal("approval without its audit event succeeded")
	}
	var n int
	if err := e.admin.Pool().QueryRow(e.ctx, `SELECT count(*) FROM sender_ids WHERE sender_id = 'UNAUDITED'`).Scan(&n); err != nil || n != 0 {
		t.Errorf("unaudited approval stored: %d, %v", n, err)
	}
}
