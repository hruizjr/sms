# backend/cmd/

One sub-folder per **executable** (runtime process). Each `main.go` must stay thin: load and validate
configuration, construct dependencies, start the process, and handle graceful shutdown. Business logic
belongs in `backend/internal/`, never in `cmd/`.

## Executables

Approved for Phase 3 (ADR-0003, accepted 2026-11-25): **`api`** (created in Group 1) and **`migrate`**
(created in Group 2) only. The other rows are candidates that need a separately approved task.

| Executable | Status | Run (from `backend/`) |
| ---------- | ------ | --------------------- |
| `api` | Implemented | `go run ./cmd/api` — serves `/healthz`, `/readyz` (checks PostgreSQL), and metrics on a loopback listener |
| `migrate` | Implemented | `go run ./cmd/migrate up\|status\|version\|down` — applies the embedded goose migrations; `down` is refused in production |

Both read the same `DB_*` configuration (see [`backend/README.md`](../README.md)). The API never runs
migrations; `migrate` is always run explicitly.

| Candidate          | Plane         | Responsibility                                                                 |
| ------------------ | ------------- | ------------------------------------------------------------------------------ |
| `api`              | Control + ingestion | HTTP server for the portals and public REST API; accepts message requests |
| `dispatcher`       | Data plane    | Consumes accepted messages, applies routing and rate limits, hands to provider adapters |
| `connector-<telco>`| Data plane    | Holds persistent telco sessions (e.g. SMPP binds) for one provider; may merge into `dispatcher` |
| `scheduler`        | Control plane | Releases scheduled campaigns, runs periodic jobs (expiry, reconciliation)       |
| `migrate`          | Tooling       | Applies database migrations (implemented in Group 2)                           |

Important constraint for later design: telcos typically limit how many simultaneous connections
("binds") an aggregator may open. That means telco connector processes **cannot simply be scaled
horizontally** like a stateless web server. The real limits must come from the telcos
(see `docs/integrations/telco-discovery-checklist.md`).

The final list will be decided in the architecture phase. Do not add a new executable without an
approved task.
