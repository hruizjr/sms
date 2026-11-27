package rlsguard

import (
	"testing"

	"github.com/hruizjr/sms/backend/internal/database"
)

// Exactly the owner-approved exceptions exist. Adding an exception requires the project owner's
// approval, recorded in the decision log with its reason (testing standards §6); update this test
// only then. Approved: public.portal_login_directory (G3-1, decision log §20).
func TestOnlyApprovedExceptions(t *testing.T) {
	t.Parallel()
	approved := []string{"public.portal_login_directory"}
	if len(Exceptions) != len(approved) {
		t.Errorf("Exceptions = %v, want exactly %v (owner approval required, decision log)", Exceptions, approved)
	}
	for _, table := range approved {
		if _, ok := Exceptions[table]; !ok {
			t.Errorf("approved exception %s is missing", table)
		}
	}
	for table, reason := range Exceptions {
		if reason == "" {
			t.Errorf("exception %s has no reason", table)
		}
	}
}

func TestTenantSettingMatches(t *testing.T) {
	t.Parallel()
	if tenantSetting != database.TenantSetting {
		t.Errorf("tenantSetting = %q, database.TenantSetting = %q; they must be equal", tenantSetting, database.TenantSetting)
	}
}

// TestPolicyEvaluation checks the decision logic without a database: only the exact approved
// policy is accepted, and missing or unknown catalog values count against a policy (fail closed).
func TestPolicyEvaluation(t *testing.T) {
	t.Parallel()
	yes, no := true, false
	all, sel := "*", "r"
	expr, open := approvedExpression, "true"
	name := "p"
	approved := policyRow{Table: "public.t", Enabled: true, Forced: true, Name: &name,
		Command: &all, Permissive: &yes, ToPublic: &yes, Using: &expr, WithCheck: &expr}

	if got := tableProblems([]policyRow{approved}); len(got) != 0 {
		t.Errorf("approved policy: problems %q, want none", got)
	}
	cases := []struct {
		name   string
		change func(*policyRow)
	}{
		{"restrictive", func(p *policyRow) { p.Permissive = &no }},
		{"unknown permissive flag", func(p *policyRow) { p.Permissive = nil }},
		{"SELECT only", func(p *policyRow) { p.Command = &sel }},
		{"unknown command", func(p *policyRow) { p.Command = nil }},
		{"specific roles", func(p *policyRow) { p.ToPublic = &no }},
		{"unknown roles", func(p *policyRow) { p.ToPublic = nil }},
		{"open USING", func(p *policyRow) { p.Using = &open }},
		{"no USING", func(p *policyRow) { p.Using = nil }},
		{"open WITH CHECK", func(p *policyRow) { p.WithCheck = &open }},
		{"no WITH CHECK", func(p *policyRow) { p.WithCheck = nil }},
	}
	for _, tc := range cases {
		p := approved
		tc.change(&p)
		if len(policyDeviations(p)) == 0 {
			t.Errorf("%s: accepted as the approved policy", tc.name)
		}
		// Alone, it leaves the table without an approved policy; next to the approved policy, it is
		// still reported.
		if got := tableProblems([]policyRow{p}); len(got) != 2 || got[1] != "has no approved tenant-isolation policy" {
			t.Errorf("%s alone: problems %q", tc.name, got)
		}
		if got := tableProblems([]policyRow{approved, p}); len(got) != 1 {
			t.Errorf("%s next to the approved policy: problems %q, want one", tc.name, got)
		}
	}
}
