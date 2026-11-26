# Enterprise A2P SMS Platform

A multi-tenant platform for business-to-consumer ("Application-to-Person", A2P) SMS messaging in the
Philippines, intended to connect business, government, and enterprise customers to mobile networks
through direct telco connectivity (subject to agreements and regulatory approval).

> **Project status: Phase 3 — Walking skeleton (technical foundation only).**
> The repository contains:
> - documentation;
> - a Go API skeleton (health and readiness endpoints, metrics, PostgreSQL connection, migrations);
> - the draft OpenAPI contract for the public messages API, and Go code generated from it (not served);
> - a client portal shell;
> - a local Docker Compose development stack;
> - CI.
>
> There is **no business functionality**: no SMS sending, no telco integration, no business database
> tables, no authentication or billing.
> No telco agreement, regulatory authorization, or traffic projection has been verified.
> Phase 2 discovery is ongoing ([roadmap](docs/roadmap/development-roadmap.md)).

## Start here

| If you are...                 | Read                                                                                          |
| ----------------------------- | --------------------------------------------------------------------------------------------- |
| An AI coding agent            | [`AGENTS.md`](AGENTS.md) — mandatory before any work                                          |
| The project owner             | [`docs/README.md`](docs/README.md), then the [discovery register](docs/requirements/discovery-register.md) |
| New to SMS / telco terms      | [`docs/glossary.md`](docs/glossary.md)                                                        |
| Reviewing a pull request      | [`docs/engineering/git-workflow.md`](docs/engineering/git-workflow.md)                        |

## Repository layout

```
.
├── AGENTS.md            Instructions for all AI coding agents (source of truth)
├── CLAUDE.md            Pointer that makes Claude Code load AGENTS.md
├── backend/             Go backend: one module, several runtime processes
│   ├── cmd/             Executables: api, migrate, healthcheck
│   └── internal/        Technical packages (config, database, httpserver, ...); business modules later
├── frontend/            React + TypeScript + Vite + Tailwind CSS (pnpm workspace) — client portal shell
├── contracts/           Shared API contracts (OpenAPI) and event schemas
├── infra/               Docker image and local Docker Compose stack (PostgreSQL, migrations, API)
├── scripts/             Developer and CI helper scripts (local stack check)
├── tests/               Cross-cutting tests: end-to-end, load, provider contract suites
├── docs/                Product, requirements, architecture, standards, security, risks, roadmap
└── .github/             CI workflows, pull request and task templates
```

Each top-level folder has its own `README.md` explaining what belongs there.

## Run it locally

Start PostgreSQL, the migrations and the API with one command, from the repository root (first time:
copy `infra/local/.env.example` to `infra/local/.env` and set `DB_PASSWORD`):

```bash
docker compose -f infra/local/compose.yaml up -d --build --wait
```

Then, optionally, run the client portal from `frontend/` with `pnpm dev`. Details, reset and Windows
notes: [`infra/local/README.md`](infra/local/README.md).

## Technology (see the decision log for the status of each)

- **Frontend:** React, TypeScript (strict), Vite, Tailwind CSS (accepted, ADR-0013)
- **Backend:** Go, Chi router, REST + OpenAPI (accepted, ADR-0004, ADR-0015)
- **Data:** PostgreSQL (primary). Kafka/Redpanda, Redis, ClickHouse are *candidates*, to be introduced only when justified.
- **Operations:** Docker, Prometheus, Grafana

See [`docs/engineering/decision-log.md`](docs/engineering/decision-log.md) for the status of every decision.
