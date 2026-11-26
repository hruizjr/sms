package database

import (
	"errors"
	"testing"
)

func TestValidateTenantID(t *testing.T) {
	t.Parallel()
	v7 := [16]byte{0x01, 0x9a, 0x1b, 0x2c, 0x3d, 0x4e, 0x7f, 0x00, 0x80, 1, 2, 3, 4, 5, 6, 7}
	v4 := [16]byte{0xaa, 0, 0, 0, 0, 0, 0x40, 0, 0xbf, 0, 0, 0, 0, 0, 0, 1}
	max := [16]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	badVariant := v7
	badVariant[8] = 0x40 // variant 01 (NCS)
	version0 := v7
	version0[6] = 0x0f
	version9 := v7
	version9[6] = 0x9f

	tests := []struct {
		name string
		id   [16]byte
		ok   bool
	}{
		{"UUIDv7", v7, true},
		{"UUIDv4", v4, true},
		{"nil UUID", [16]byte{}, false},
		{"max UUID (variant bits 11)", max, false},
		{"wrong variant", badVariant, false},
		{"version 0", version0, false},
		{"version 9", version9, false},
	}
	for _, tc := range tests {
		err := ValidateTenantID(tc.id)
		if tc.ok && err != nil {
			t.Errorf("%s: ValidateTenantID() = %v, want nil", tc.name, err)
		}
		if !tc.ok && !errors.Is(err, ErrInvalidTenantID) {
			t.Errorf("%s: ValidateTenantID() = %v, want ErrInvalidTenantID", tc.name, err)
		}
	}
}
