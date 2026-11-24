# Enterprise A2P SMS Platform

A multi-tenant platform for business-to-consumer ("Application-to-Person", A2P) SMS messaging in the
Philippines, intended to connect business, government, and enterprise customers to mobile networks
through direct telco connectivity (subject to agreements and regulatory approval).

> **Project status: Phase 1 — Engineering Foundation.**
> This repository currently contains **documentation and repository structure only**.
> There is no application code, no database schema, and no telco integration.
> No telco agreement, regulatory authorization, or traffic projection has been verified.
> Phase 2 has **not** been authorized.

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
│   ├── cmd/             One folder per executable (API server, workers, ...) — none yet
│   └── internal/        Private application modules (billing, messaging, ...) — none yet
├── frontend/            React + TypeScript + Vite + Tailwind CSS web application — not yet scaffolded
├── contracts/           Shared API contracts (OpenAPI) and event schemas
├── infra/               Local development and deployment configuration (Docker, monitoring)
├── scripts/             Developer and CI helper scripts
├── tests/               Cross-cutting tests: end-to-end, load, provider contract suites
├── docs/                Product, requirements, architecture, standards, security, risks, roadmap
└── .github/             Pull request and task templates
```

Each top-level folder has its own `README.md` explaining what belongs there.

## Technology preferences (not all approved — see the decision log)

- **Frontend:** React, TypeScript (strict), Vite, Tailwind CSS
- **Backend:** Go, Chi router (under evaluation), REST
- **Data:** PostgreSQL (primary). Kafka/Redpanda, Redis, ClickHouse are *candidates*, to be introduced only when justified.
- **Operations:** Docker, Prometheus, Grafana

See [`docs/engineering/decision-log.md`](docs/engineering/decision-log.md) for the status of every decision.
