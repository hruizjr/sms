// Package migrations embeds the versioned SQL migration files so that the migrate executable does
// not depend on its working directory.
//
// Files are named <UTC timestamp YYYYMMDDHHMMSS>_<snake_case_description>.sql (AGENTS.md §10.1)
// and use goose annotations (-- +goose Up / -- +goose Down). A merged migration is immutable.
package migrations

import "embed"

// FS contains every *.sql migration file in this directory.
//
//go:embed *.sql
var FS embed.FS
