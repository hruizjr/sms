# Decision Log (Architecture Decision Records)

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — all ADRs Proposed | Project owner | 2026-11-24 |

## 1. What an ADR is and why we use them

An **Architecture Decision Record (ADR)** is a short record of one significant decision: what problem we
faced, the options, what we chose, and the consequences. ADRs let future AI agents and humans understand
*why* the system is the way it is, and stop them from re-arguing or silently reversing decisions.

## 2. Process

1. **Propose:** anyone (agent or owner) adds an ADR using the template below with status `Proposed`.
2. **Review:** the project owner reviews. Questions are added to the ADR's "Missing information".
3. **Decide:** **only the project owner** changes the status to `Accepted` or `Rejected`, recording the
   date.
4. **Change:** an accepted ADR is never rewritten. To change it, write a new ADR that *supersedes* it and
   set the old one to `Superseded by ADR-XXXX`.

**Statuses:** `Proposed` · `Accepted` · `Rejected` · `Superseded by ADR-XXXX` · `Deprecated`

**When an ADR is required:** new infrastructure components, new frameworks/major dependencies, changes to
architectural constraints, tenant/billing/ledger model changes, public API conventions, security model
changes, and anything expensive to reverse.

Short ADRs live in this file. If an ADR grows beyond ~1 page, move it to `docs/engineering/adr/NNNN-title.md`
and keep a one-line entry in the index.

## 3. Template

```markdown
### ADR-NNNN: <Decision title>

- **Status:** Proposed
- **Date proposed:** YYYY-MM-DD
- **Date decided:** —
- **Decided by:** —

**Context** — What problem or need prompted this decision? What constraints apply?

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. ... | ... | ... |
| B. ... | ... | ... |

**Recommendation / Decision** — What is recommended (or, once accepted, decided) and why.

**Consequences** — What becomes easier, what becomes harder, follow-up work.

**Operational cost / complexity** — What it costs to run and maintain.

**Missing information** — What we still need to know; what could change this decision.
```

## 4. Index

| ADR | Title | Status |
| --- | ----- | ------ |
| 0001 | Record architecture decisions as ADRs | Proposed |
| 0002 | Monorepo layout | Proposed |
| 0003 | Modular monolith with multiple executables | Proposed |
| 0004 | HTTP router: Chi vs. standard library | Proposed |
| 0005 | PostgreSQL as system of record | Proposed |
| 0006 | Message broker (Kafka/Redpanda) — defer; transactional outbox now | Proposed |
| 0007 | Redis — defer until a specific need | Proposed |
| 0008 | ClickHouse — defer until analytics volume requires it | Proposed |
| 0009 | Logical control/data plane separation | Proposed |
| 0010 | Multi-tenancy: shared schema with `tenant_id` + RLS defense-in-depth | Proposed |
| 0011 | Credit ledger model | Proposed |
| 0012 | Provider abstraction and simulator adapter | Proposed |
| 0013 | Frontend application structure and supporting libraries | Proposed |
| 0014 | Database access and migrations tooling | Proposed |
| 0015 | API contracts: OpenAPI-first | Proposed |
| 0016 | Structured logging with `log/slog` | Proposed |
| 0017 | Git workflow for one owner + multiple AI agents | Proposed |

---

## 5. Records

### ADR-0001: Record architecture decisions as ADRs

- **Status:** Proposed · **Date proposed:** 2026-11-24

**Context** — Most code will be written by AI agents that have no memory between tasks. Decisions must be
written down to be followed consistently.

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. ADRs in the repository | Versioned with code; agents read them | Requires discipline |
| B. Decisions in chat/meetings only | No overhead | Lost; agents cannot see them |
| C. External wiki | Rich editing | Drifts from code; agents may lack access |

**Recommendation** — A.

**Consequences** — Every significant change references an ADR. Slight documentation overhead.

**Operational cost** — Negligible. **Missing information** — None.

---

### ADR-0002: Monorepo layout

- **Status:** Proposed · **Date proposed:** 2026-11-24 (owner has already stated a monorepo preference;
  this ADR records the specific *layout*)

**Context** — Owner requires one repository for frontend, backend, infrastructure, contracts, and docs.

**Options considered (layout)**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. Top-level `backend/`, `frontend/`, `contracts/`, `infra/`, `scripts/`, `tests/`, `docs/` | Self-explanatory to non-specialists; clear ownership | Go module lives in a subfolder (minor tooling consideration) |
| B. Go module at repository root with `cmd/`, `internal/`, plus `web/` | Idiomatic for Go-only repos | Mixes Go and non-Go concerns at root; less clear |
| C. Polyglot build system (Bazel, Nx) | Powerful caching, cross-language builds | Significant learning and maintenance cost; unjustified now |

**Recommendation** — A (implemented in Phase 1 as folders and READMEs only).

**Consequences** — Simple, discoverable structure. CI runs Go and frontend pipelines based on changed paths.

**Operational cost** — Low. **Missing information** — Go module path (e.g. `github.com/hruizjr/sms/backend`)
to be confirmed at scaffolding.

---

### ADR-0003: Modular monolith with multiple executables

- **Status:** Proposed · **Date proposed:** 2026-11-24

**Context** — The platform has distinct capabilities (identity, billing, messaging, providers ...) and
eventually high-volume processing. Team: one owner and AI agents. Traffic: unverified.

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. Single process monolith | Simplest | Control and data plane share fate; hard to scale parts independently |
| B. Modular monolith, one Go module, several executables | Clear boundaries; independent deployment of API vs. dispatcher; simple builds; easy refactoring | Requires discipline to keep module boundaries |
| C. Microservices from the start | Independent scaling and deployment per service | High operational cost; distributed transactions; slower development; premature without traffic data |

**Recommendation** — B. Modules communicate through Go interfaces; each owns its tables. Extract a module
into a service only with evidence (new ADR).

**Consequences** — Fast development, fewer moving parts. Boundaries must be enforced in review (and
possibly by an import-lint rule later).

**Operational cost** — Low to moderate (a few deployables). **Missing information** — Final executable
list (architecture phase).

---

### ADR-0004: HTTP router — Chi vs. standard library

- **Status:** Proposed · **Date proposed:** 2026-11-24

**Context** — Owner prefers Chi, subject to evaluation. Since Go 1.22 the standard library `net/http`
`ServeMux` supports HTTP-method matching and path wildcards, which narrows the gap.

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. Chi | Route groups and sub-routers; per-group middleware composition; uses standard `http.Handler`, so handlers stay portable; mature and small | One external dependency |
| B. Standard library `ServeMux` only | Zero dependencies | Middleware grouping must be hand-built; more boilerplate for large route trees |
| C. Gin / Echo / Fiber | Batteries included | Non-standard handler signatures (lock-in); Fiber is not `net/http`-based |

**Recommendation** — A (Chi). It fits the owner's preference, keeps handlers as plain `net/http`, and its
middleware grouping is useful for tenant/auth/permission layers. Switching cost back to the standard
library remains low because handlers are standard.

**Consequences** — One small dependency. Handlers MUST use standard `http.Handler`/`http.HandlerFunc`
signatures (no Chi-specific context access outside routing glue).

**Operational cost** — Negligible. **Missing information** — Confirm Chi's current release and maintenance
status at scaffolding time.

---

### ADR-0005: PostgreSQL as system of record

- **Status:** Proposed · **Date proposed:** 2026-11-24

**Context** — Need ACID transactions for money and message state, strong constraints, mature tooling.

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. PostgreSQL | ACID; constraints; row-level security; partitioning; excellent Go support; widely known | Very high write volumes need careful design (partitioning, batching), possibly later sharding |
| B. MySQL/MariaDB | Mature, widely hosted | Weaker in some features used here (e.g. RLS) |
| C. NoSQL (e.g. Cassandra/DynamoDB) | Horizontal write scaling | Weak transactions — unsuitable for the ledger; high complexity |

**Recommendation** — A, for all transactional data. Revisit data-plane storage only if verified volume
exceeds what a well-tuned PostgreSQL deployment handles.

**Consequences** — Ledger and tenant isolation can rely on database guarantees.

**Operational cost** — Moderate: backups, replication, upgrades, monitoring (managed service reduces
this). **Missing information** — Hosting decision; version to pin; HA/DR targets.

---

### ADR-0006: Message broker — defer Kafka/Redpanda; use transactional outbox now

- **Status:** Proposed · **Date proposed:** 2026-11-24

**Context** — The data plane needs durable buffering between acceptance and dispatch, and between DLR
receipt and processing. Kafka and Redpanda are candidates. Traffic volume is unverified.

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. Kafka | Industry standard; very high throughput; durable replicated log; large ecosystem | Operationally demanding (clusters, partitions, upgrades, monitoring); significant for a small team |
| B. Redpanda | Kafka-API compatible; single binary; simpler operations | Smaller community; licensing terms should be reviewed; vendor dependence |
| C. PostgreSQL-backed queue (outbox table + workers using `FOR UPDATE SKIP LOCKED`) | No new infrastructure; transactional with business data | Throughput ceiling lower than a log broker; table bloat must be managed |
| D. Managed cloud queue/stream | No cluster operations | Cloud lock-in; depends on hosting decision |
| E. NATS JetStream / RabbitMQ | Simpler than Kafka for queues | Not in owner's preferences; different semantics |

**Recommendation** — Start with **C** for the first messaging milestones, behind an internal
publisher/consumer interface, with the transactional outbox pattern (needed regardless of broker).
Choose a Kafka-API broker (A or B, or a managed Kafka-compatible service) when verified throughput,
retention, or fan-out needs exceed what C handles — decided by load tests and traffic data. Because both
A and B speak the Kafka protocol, client code would work with either.

**Consequences** — Fewer moving parts early; a migration step later if a broker is adopted (mitigated by
the interface). Load tests must establish the PostgreSQL queue's ceiling.

**Operational cost** — C: low. A: high. B: moderate. **Missing information** — Peak TPS, retention needs,
number of consumers, hosting model, ops staffing.

---

### ADR-0007: Redis — defer until a specific need

- **Status:** Proposed · **Date proposed:** 2026-11-24

**Context** — Redis is proposed for caching and short-lived coordination (e.g. distributed rate limiting).

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. Introduce Redis now | Ready when needed | Another system to secure, monitor, back up; temptation to store authoritative data in it |
| B. Introduce when a concrete need arises (e.g. cross-instance rate limiting, hot-path caching measured as necessary) | No premature complexity | Must be added later |

**Recommendation** — B. When introduced, Redis MUST NOT be the source of truth for credits, balances,
idempotency records required for billing correctness, or message state.

**Operational cost** — Moderate when introduced. **Missing information** — Number of dispatcher instances,
rate-limit coordination design (AQ-9).

---

### ADR-0008: ClickHouse — defer until analytics volume requires it

- **Status:** Proposed · **Date proposed:** 2026-11-24

**Context** — ClickHouse is excellent for large-scale analytical queries. Reporting requirements and
volumes are unknown.

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. ClickHouse from the start | Fast analytics at scale | Data pipeline, second store, consistency questions, ops cost |
| B. PostgreSQL with pre-aggregated reporting tables first | Simple; consistent | May not scale to very large history |

**Recommendation** — B initially; adopt A when reporting volume or query latency, measured against verified
requirements, demands it.

**Operational cost** — A: moderate to high. **Missing information** — Reporting requirements (D-072),
retention (D-073), volumes (D-060).

---

### ADR-0009: Logical control/data plane separation

- **Status:** Proposed · **Date proposed:** 2026-11-24

**Context** — See architecture principle P2.

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. No separation | Simple | Shared fate between admin and messaging |
| B. Logical separation (modules) + separate processes from the same codebase | Isolation of load and failure; one codebase | Requires defined interfaces for configuration the data plane needs |
| C. Separate codebases/services and databases | Maximum isolation | High complexity; premature |

**Recommendation** — B. Data plane processes may share the PostgreSQL cluster initially; separating
databases is a later, evidence-based decision.

**Operational cost** — Low to moderate. **Missing information** — Availability targets (D-067).

---

### ADR-0010: Multi-tenancy — shared schema with `tenant_id` plus Row-Level Security

- **Status:** Proposed · **Date proposed:** 2026-11-24

**Context** — Many tenants, strict isolation, potentially large data volume.

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. Shared tables with `tenant_id` column, enforced in application | Simple; scales to many tenants; easy cross-tenant admin reporting | One missed filter leaks data |
| B. A + PostgreSQL Row-Level Security as defense-in-depth | A database-level backstop against missed filters | Requires setting tenant context per connection/transaction; must design admin and background-job access; some performance consideration |
| C. Schema per tenant | Stronger separation | Migrations × tenants; operational burden grows with tenant count |
| D. Database per tenant | Strongest isolation | Expensive; complex; only for special customers |

**Recommendation** — B for control-plane tenant data. Evaluate RLS for high-volume data-plane tables in the
architecture phase (performance testing). Keep D available as a premium/government option only if a
customer contract requires it.

**Operational cost** — Low to moderate. **Missing information** — Whether any customers require dedicated
infrastructure; sub-account/reseller hierarchy (D-064).

---

### ADR-0011: Credit ledger model

- **Status:** Proposed · **Date proposed:** 2026-11-24

**Context** — Credits must never be double-spent, lost, or silently changed, even with many concurrent
sends.

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. Single mutable `balance` column | Simple | No history; errors are invisible; hard to audit |
| B. Append-only ledger entries + balance derived (or cached and verifiable) + reservations | Full audit trail; reconcilable; concurrency-safe with row locks/conditional updates | More design work; cached balance must stay consistent |
| C. External billing system | Offloads work | Integration complexity; latency in the send path |

**Recommendation** — B:
- Amounts as `bigint` integer minor units; the unit and precision depend on business decision D-081.
- Entry types (proposed): `top_up`, `reservation`, `reservation_release`, `charge`, `refund`,
  `adjustment` (admin, requires reason and audit), each with an idempotency key (e.g. message ID + type).
- A per-account balance row updated in the same transaction as ledger inserts, with a `CHECK` that
  available balance never goes below the allowed limit; periodic verification against the ledger sum.
- Reservation at acceptance, settlement/release per business rules (D-080, D-083).

**Consequences** — Strong correctness guarantees; the balance row is a contention point for very
high-volume single tenants (mitigation options: batched reservations, sub-balances — to be evaluated
with load tests).

**Operational cost** — Low (it is application design). **Missing information** — D-080 to D-084.

---

### ADR-0012: Provider abstraction and simulator adapter

- **Status:** Proposed · **Date proposed:** 2026-11-24

**Context** — Owner requires Smart and Globe as independent adapters behind a common interface.

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. Telco logic in the dispatcher with branches per telco | Quick start | Tight coupling; every new telco touches core code |
| B. `Provider` interface; one adapter package per telco; simulator adapter; shared contract test suite | Isolation; testability; new telcos added without core changes | Interface must be designed carefully to fit real telco behaviour |
| C. Third-party SMS gateway software in front of telcos | Less protocol code | Another system to operate; may not fit; licensing |

**Recommendation** — B. The interface is drafted **after** at least one telco's technical specification is
received, so it reflects reality; the simulator is built first against a provisional interface. The SMPP
client library choice is a separate ADR made with telco specifications in hand.

**Operational cost** — Low. **Missing information** — Telco protocols and behaviours (telco checklist).

---

### ADR-0013: Frontend application structure and supporting libraries

- **Status:** Proposed · **Date proposed:** 2026-11-24

**Context** — React + TS + Vite + Tailwind are preferred. Two audiences (clients, our admins). Owner wants
no unnecessary UI or styling libraries.

**Decisions to make**
| Topic | Options | Recommendation |
| ----- | ------- | -------------- |
| App split | One app vs. two apps sharing a `ui` package | Two apps in a small workspace (security: admin on separate origin) |
| Routing | React Router vs. TanStack Router | Either; choose at scaffolding based on current docs. Not a styling library. |
| Server data | Hand-written hooks vs. TanStack Query | TanStack Query (caching, retries, invalidation are hard to hand-build correctly) |
| Form validation | Hand-written vs. a schema library (e.g. Zod) + form library | Schema library; shared schemas improve consistency |
| Accessible primitives | Build dialogs/menus/comboboxes ourselves vs. a headless (unstyled) primitives library | Owner decision — headless libraries add no styling (Tailwind remains the only styling system) but are a dependency; building accessible primitives from scratch is error-prone |
| Package manager | npm vs. pnpm | pnpm (good workspace support) — or npm for simplicity |

**Operational cost** — Low. **Missing information** — Owner's view on each dependency; current versions at
scaffolding time.

---

### ADR-0014: Database access and migration tooling

- **Status:** Proposed · **Date proposed:** 2026-11-24

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. `pgx` + hand-written SQL | Full control, performance | Boilerplate; manual scanning errors |
| B. `pgx` + `sqlc` (type-safe Go generated from SQL) | Plain SQL reviewed as SQL; compile-time type safety; no ORM magic | Code generation step |
| C. ORM (GORM, ent) | Less SQL | Hidden queries; harder performance tuning; harder to reason about transactions and locking |

Migration tools: `goose`, `golang-migrate`, `atlas` — all viable; choose one supporting SQL migration
files, timestamp versioning, and programmatic execution.

**Recommendation** — B with one SQL-file migration tool selected at scaffolding.

**Operational cost** — Low. **Missing information** — Tool versions and maintenance status at scaffolding.

---

### ADR-0015: API contracts — OpenAPI-first

- **Status:** Proposed · **Date proposed:** 2026-11-24

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. Spec-first: OpenAPI files in `contracts/`, TypeScript types generated, Go validated/generated against spec | Single source of truth; public docs come free; contract tests | Spec authoring discipline |
| B. Code-first: generate spec from Go annotations | Less duplication for backend developers | Spec quality depends on annotations; frontend waits on backend |
| C. gRPC/GraphQL | Strong typing / flexible queries | Not REST; public SMS API consumers expect REST |

**Recommendation** — A. Conventions: `/v1` versioning, `snake_case` JSON, RFC 9457 errors, cursor pagination,
idempotency keys on submission.

**Operational cost** — Low. **Missing information** — Generator tool choices at scaffolding.

---

### ADR-0016: Structured logging with `log/slog`

- **Status:** Proposed · **Date proposed:** 2026-11-24

**Options** — A. Standard library `log/slog` · B. zap / zerolog.
**Recommendation** — A: structured, part of the standard library (no dependency), adequate performance for
our needs; revisit only if profiling shows logging as a bottleneck.

**Operational cost** — None. **Missing information** — Log aggregation platform (hosting decision).

---

### ADR-0017: Git workflow for one owner + multiple AI agents

- **Status:** Proposed · **Date proposed:** 2026-11-24

**Context** — One human reviewer; several agents possibly working in parallel.

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. Trunk-based: protected `main`, short-lived task branches, PRs, squash merge | Simple; clean history; owner controls every merge | Owner review is the bottleneck |
| B. GitFlow (develop/release/hotfix branches) | Formal release management | Overhead without benefit at this stage |

**Recommendation** — A, detailed in [git-workflow.md](git-workflow.md).

**Operational cost** — Low. **Missing information** — None.
