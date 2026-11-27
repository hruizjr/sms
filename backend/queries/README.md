# backend/queries/

Hand-written SQL queries compiled by [sqlc](../sqlc.yaml) into type-safe Go in
`internal/database/sqlcgen/` (ADR-0014: `pgx/v5` + `sqlc`; no ORM).

| File | Module | Tables |
| ---- | ------ | ------ |
| [`tenancy.sql`](tenancy.sql) | `internal/tenancy` (Phase 4 Group 1) | `tenants` (platform-owned, no `tenant_id`) |

To add or change a query:

1. Put it in the module's `.sql` file here (one file per module, for example `tenancy.sql`), with a
   `-- name: <QueryName> :one|:many|:exec|:execrows` comment. Comment lines directly above the name line
   become the generated method's documentation.
2. Regenerate from `backend/` with the locked sqlc version and commit the result (see
   [`backend/README.md`](../README.md), "sqlc"):

   ```powershell
   go generate ./internal/database/...
   ```

3. Follow `AGENTS.md` §7.8 and §10: parameterized queries only, no `SELECT *`, and every tenant-scoped
   query filters by `tenant_id` and runs inside `database.DB.InTenantTx`.
4. Only the owning module calls its queries (AGENTS.md §5.3). The application role must have been granted
   the privileges the query needs, in the migration that creates the table.
