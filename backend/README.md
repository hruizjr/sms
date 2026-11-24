# backend/

The Go backend. **No code exists yet** — this folder is reserved and documented in Phase 1.

## Proposed shape (see ADR-0003 and ADR-0009 in the decision log — status: Proposed)

The backend is planned as a **modular monolith**: a single Go module (one `go.mod`) that builds
**several executables**. Each executable is a separate runtime process that can be deployed and
scaled independently, but they share code, conventions, and a single build.

```
backend/
├── go.mod                 Created during the scaffolding phase (module path pending approval)
├── cmd/                   One sub-folder per executable — see cmd/README.md
├── internal/              Application modules, private to this repository — see internal/README.md
└── migrations/            Versioned PostgreSQL migrations (created when the DB layer is approved)
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
Do not create `go.mod`, packages, or migrations until the scaffolding task is approved.
