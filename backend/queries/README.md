# backend/queries/

Hand-written SQL queries compiled by [sqlc](../sqlc.yaml) into type-safe Go in
`internal/database/sqlcgen/` (ADR-0014: `pgx/v5` + `sqlc`; no ORM).

**This folder is intentionally empty.** The Phase 3 walking skeleton has no business tables, so it has no
queries, and sqlc refuses to generate code until at least one query exists. Do not add placeholder queries.

When the first approved business feature adds its first migration and query:

1. Add the query to a `.sql` file here, named after the module (for example `tenancy.sql`), with a
   `-- name: <QueryName> :one|:many|:exec|:execrows` comment.
2. Run `sqlc generate` from `backend/` and commit the generated files.
3. Follow `AGENTS.md` §7.8 and §10: parameterized queries only, no `SELECT *`, and every tenant-scoped
   query filters by `tenant_id`.
