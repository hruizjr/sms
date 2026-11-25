package migrations

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

// fileName is <UTC timestamp YYYYMMDDHHMMSS>_<snake_case_description>.sql (AGENTS.md §10.1).
var fileName = regexp.MustCompile(`^[0-9]{14}_[a-z0-9]+(_[a-z0-9]+)*\.sql$`)

func TestEmbeddedMigrationFiles(t *testing.T) {
	t.Parallel()

	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no migrations embedded")
	}
	seen := map[string]string{}
	for _, e := range entries {
		name := e.Name()
		if !fileName.MatchString(name) {
			t.Errorf("%s: name must match %s", name, fileName)
		}
		version := name[:14]
		if other, dup := seen[version]; dup {
			t.Errorf("%s and %s share version %s", name, other, version)
		}
		seen[version] = name

		body, err := fs.ReadFile(FS, name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if !strings.Contains(string(body), "-- +goose Up") {
			t.Errorf("%s: missing the -- +goose Up annotation", name)
		}
	}
}
