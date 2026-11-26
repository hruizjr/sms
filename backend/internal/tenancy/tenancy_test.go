package tenancy

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeDisplayName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
		ok       bool
	}{
		{"Example Tenant", "Example Tenant", true},
		{"  Padded Tenant \t", "Padded Tenant", true},
		{"Ñandú Café 東京", "Ñandú Café 東京", true},
		{strings.Repeat("é", MaxDisplayNameLength), strings.Repeat("é", MaxDisplayNameLength), true},
		{"", "", false},
		{"   ", "", false},
		{strings.Repeat("a", MaxDisplayNameLength+1), "", false},
		{"line\nbreak", "", false},
		{"tab\tinside", "", false},
		{"nul\x00byte", "", false},
		{"bad \xff utf-8", "", false},
	}
	for _, tc := range tests {
		got, err := NormalizeDisplayName(tc.in)
		if tc.ok {
			if err != nil || got != tc.want {
				t.Errorf("NormalizeDisplayName(%q) = %q, %v; want %q, nil", tc.in, got, err, tc.want)
			}
			continue
		}
		if !errors.Is(err, ErrInvalidDisplayName) {
			t.Errorf("NormalizeDisplayName(%q) error = %v, want ErrInvalidDisplayName", tc.in, err)
		}
	}
}

func TestID(t *testing.T) {
	t.Parallel()
	const text = "019abc12-3456-7def-8123-456789abcdef"
	id, err := ParseID(text)
	if err != nil {
		t.Fatalf("ParseID(%q) error: %v", text, err)
	}
	if id.String() != text || id.Version() != 7 || id.IsZero() {
		t.Errorf("ParseID(%q) = %s (version %d, zero %v)", text, id, id.Version(), id.IsZero())
	}
	upper, err := ParseID(strings.ToUpper(text))
	if err != nil || upper != id {
		t.Errorf("ParseID(upper case) = %s, %v; want %s", upper, err, id)
	}
	for _, bad := range []string{"", "not-a-uuid", text + "0", "' OR true --"} {
		if _, err := ParseID(bad); !errors.Is(err, ErrInvalidID) {
			t.Errorf("ParseID(%q) error = %v, want ErrInvalidID", bad, err)
		}
	}
	if !(ID{}).IsZero() {
		t.Error("ID{}.IsZero() = false")
	}
}
