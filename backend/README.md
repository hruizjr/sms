# backend/

The Go backend. **No code exists yet** — it is created during the Phase 3 walking skeleton, one reviewed
group at a time (see the [decision log §7](../docs/engineering/decision-log.md#7-owner-decisions--phase-3-walking-skeleton-2026-11-25)).

## Shape (ADR-0002 and ADR-0003 accepted; ADR-0009 still Proposed)

The backend is planned as a **modular monolith**: a single Go module (one `go.mod`) that builds
**several executables**. Each executable is a separate runtime process that can be deployed and
scaled independently, but they share code, conventions, and a single build.

```
backend/
├── go.mod                 Module github.com/hruizjr/sms/backend, Go 1.27.4 (created in Phase 3)
├── cmd/                   One sub-folder per executable — see cmd/README.md
├── internal/              Application modules, private to this repository — see internal/README.md
└── migrations/            Versioned PostgreSQL migrations, managed with goose (created in Phase 3)
```

Why this shape:

- **One module, many binaries** lets us separate the *control plane* (accounts, billing, admin) from
  the *data plane* (message dispatch, delivery receipts) at deployment time, without the overhead of
  separate repositories, separate versioning, or network calls between every module.
- **`internal/`** is enforced by the Go toolchain: code there cannot be imported by other modules, so
  we keep freedom to refactor.
- We can extract a module into its own service later if verified load or team structure requires it.

## Rules

Follow [`AGENTS.md`](../AGENTS.md) §7 (Go standards) and
[`docs/standards/coding-standards.md`](../docs/standards/coding-standards.md).
Create `go.mod`, packages, and migrations only within an approved Phase 3 group task. Walking-skeleton
migrations contain no business tables.
