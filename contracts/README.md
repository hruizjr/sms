# contracts/

**Shared interface contracts** — the agreements between parts of the system that are owned and
versioned independently of any one implementation.

| Planned sub-folder | Contents | Consumers |
| ------------------ | -------- | --------- |
| `openapi/` | OpenAPI 3.x specifications for the portal API and the public developer API | Go server, generated TypeScript types for the frontend, public API documentation, contract tests |
| `events/`  | Schemas for internal asynchronous events (e.g. *message accepted*, *delivery receipt received*) | Producers and consumers in the data plane |
| `webhooks/`| Schemas of the payloads we send to customer webhooks | Public documentation, contract tests |

None of these exist yet. They will be created in the architecture/scaffolding phases.

## Rules

- **Contract changes are never silent.** Any change to a file in `contracts/` must be called out in the
  pull request description under "Contract changes", with compatibility impact.
- The **public API is versioned** (e.g. `/v1`). Breaking changes to a released version are prohibited;
  introduce a new version instead.
- Specifications are the source of truth: code is generated from or validated against them, never
  the other way round (proposed in ADR-0015).
- Contracts must not contain telco-specific details invented by an agent. Telco protocol details
  belong in provider adapters and must be backed by official telco documentation.
