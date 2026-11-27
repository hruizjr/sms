# contracts/

**Shared interface contracts** — the agreements between parts of the system that are owned and
versioned independently of any one implementation.

| Planned sub-folder | Contents | Consumers |
| ------------------ | -------- | --------- |
| `openapi/` | OpenAPI 3.x specifications for the portal API and the public developer API | Go server (types and routes generated into `backend/internal/publicapi`), generated TypeScript types for the frontend, public API documentation, contract tests |
| `events/`  | Schemas for internal asynchronous events (e.g. *message accepted*, *delivery receipt received*) | Producers and consumers in the data plane |
| `webhooks/`| Schemas of the payloads we send to customer webhooks | Public documentation, contract tests |

Current contents (Phase 3 Group 3):

| Path | Contents |
| ---- | -------- |
| `openapi/public-api-v1.yaml` | **Draft** public developer API v1: message submission and status (implemented in Phase 4 Group 2, simulator provider only) |
| `openapi/README.md` | Semantics of the public API, status reference, problem types, and open owner decisions |
| `tests/` | Contract tests (Node.js built-in test runner, no extra dependencies) |
| `redocly.yaml` | OpenAPI lint rules (`@redocly/cli`, approved in decision log §7.2) |
| `package.json`, `pnpm-lock.yaml`, `.npmrc` | Pinned tooling: Node.js 26.13.1, pnpm 10.34.9, `@redocly/cli` 2.77.0 |

`events/` and `webhooks/` do not exist yet. The portal API is added with the portal features.

Validate locally from this directory (the same commands run in CI, job **OpenAPI contract check**):

```bash
pnpm install --frozen-lockfile
pnpm run check
```

## Rules

- **Contract changes are never silent.** Any change to a file in `contracts/` must be called out in the
  pull request description under "Contract changes", with compatibility impact.
- The **public API is versioned** (e.g. `/v1`). Breaking changes to a released version are prohibited;
  introduce a new version instead.
- Specifications are the source of truth: code is generated from or validated against them, never
  the other way round (ADR-0015, accepted).
- Contracts must not contain telco-specific details invented by an agent. Telco protocol details
  belong in provider adapters and must be backed by official telco documentation.
