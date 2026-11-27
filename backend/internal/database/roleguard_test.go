package database

import (
	"errors"
	"fmt"
	"testing"
)

func TestUnsafeRoleError(t *testing.T) {
	t.Parallel()
	var err error = &UnsafeRoleError{Purpose: "application", Role: "sms", Problems: []string{"is a superuser", "has BYPASSRLS"}}
	err = fmt.Errorf("ping database: %w", err)
	if !errors.Is(err, ErrUnsafeRole) {
		t.Errorf("errors.Is(%v, ErrUnsafeRole) = false", err)
	}
	var unsafe *UnsafeRoleError
	if !errors.As(err, &unsafe) || len(unsafe.Problems) != 2 {
		t.Fatalf("errors.As() did not find the UnsafeRoleError in %v", err)
	}
	want := `ping database: refusing to use database role "sms" as the application role: is a superuser; has BYPASSRLS`
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}
