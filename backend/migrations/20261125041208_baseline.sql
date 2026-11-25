-- Baseline migration (Phase 3 Group 2). It intentionally contains NO schema changes.
--
-- Why it exists: goose refuses to run with zero migrations (goose.ErrNoMigrations), and the
-- migrations are embedded into the migrate executable with go:embed, which needs at least one
-- matching file. Applying this migration only creates goose's own version table
-- (goose_db_version) and records this version, which proves the migration pipeline end to end.
--
-- Business tables (tenants, customers, credits, messages, ...) are not part of the walking
-- skeleton (decision log §7.3). The first real migration is added by the first approved business
-- feature, with a later timestamp.
--
-- Retention class: not applicable (no tables).

-- +goose Up

-- +goose Down
