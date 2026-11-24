# backend/cmd/

One sub-folder per **executable** (runtime process). Each `main.go` must stay thin: load and validate
configuration, construct dependencies, start the process, and handle graceful shutdown. Business logic
belongs in `backend/internal/`, never in `cmd/`.

## Candidate executables (Proposed — not approved, not created)

| Candidate          | Plane         | Responsibility                                                                 |
| ------------------ | ------------- | ------------------------------------------------------------------------------ |
| `api`              | Control + ingestion | HTTP server for the portals and public REST API; accepts message requests |
| `dispatcher`       | Data plane    | Consumes accepted messages, applies routing and rate limits, hands to provider adapters |
| `connector-<telco>`| Data plane    | Holds persistent telco sessions (e.g. SMPP binds) for one provider; may merge into `dispatcher` |
| `scheduler`        | Control plane | Releases scheduled campaigns, runs periodic jobs (expiry, reconciliation)       |
| `migrate`          | Tooling       | Applies database migrations                                                    |

Important constraint for later design: telcos typically limit how many simultaneous connections
("binds") an aggregator may open. That means telco connector processes **cannot simply be scaled
horizontally** like a stateless web server. The real limits must come from the telcos
(see `docs/integrations/telco-discovery-checklist.md`).

The final list will be decided in the architecture phase. Do not add a new executable without an
approved task.
