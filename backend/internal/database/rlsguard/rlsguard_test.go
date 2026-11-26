package rlsguard

import (
	"testing"

	"github.com/hruizjr/sms/backend/internal/database"
)

// No tenant-owned table is exempt. Adding an exception requires the project owner's approval,
// recorded in the decision log with its reason (testing standards §6); update this test only then.
func TestNoExceptions(t *testing.T) {
	t.Parallel()
	if len(Exceptions) != 0 {
		t.Errorf("Exceptions = %v, want none (owner approval required, decision log)", Exceptions)
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
