# Decision Log (Architecture Decision Records)

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Living document — nine ADRs accepted 2026-11-25; others Proposed | Project owner | 2026-11-25 |

## 1. What an ADR is and why we use them

An **Architecture Decision Record (ADR)** is a short record of one significant decision: what problem we
faced, the options, what we chose, and the consequences. ADRs let future AI agents and humans understand
*why* the system is the way it is, and stop them from re-arguing or silently reversing decisions.

**Authority** ([`AGENTS.md` §1.1](../../AGENTS.md#11-document-authority-and-conflicts)): **Accepted** ADRs
are authoritative for approved architectural decisions. **Proposed** ADRs are the current direction but are
not binding. No ADR grants permissions or widens task scope beyond `AGENTS.md`. If an ADR genuinely
conflicts with another document, stop the affected work, report the conflict, and request owner
resolution.

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

**On 2026-11-25 the owner accepted ADR-0002, 0003, 0004, 0005, 0013, 0014, 0015, 0016, and 0017.** All other
ADRs remain `Proposed`; the owner's approval is limited to those decisions and the Phase 3 walking skeleton.
**Later on 2026-11-25 (PR #7 review) the owner also accepted ADR-0018, and ADR-0021 with a limited scope**
(see [§8](#8-owner-decisions--public-api-contract-2026-11-25)).

| ADR | Title | Status |
| --- | ----- | ------ |
| 0001 | Record architecture decisions as ADRs | Proposed |
| 0002 | Monorepo layout | **Accepted** (owner, 2026-11-25) |
| 0003 | Modular monolith with multiple executables | **Accepted** (owner, 2026-11-25) |
| 0004 | HTTP router: Chi vs. standard library | **Accepted** (owner, 2026-11-25) |
| 0005 | PostgreSQL as system of record | **Accepted** (owner, 2026-11-25) |
| 0006 | Message broker (Kafka/Redpanda) — defer, with reconsideration criteria | Proposed (revised 2026-11-24) |
| 0007 | Redis — defer, with reconsideration criteria | Proposed (revised 2026-11-24) |
| 0008 | ClickHouse — defer, with reconsideration criteria | Proposed (revised 2026-11-24) |
| 0009 | Logical control/data plane separation | Proposed |
| 0010 | Multi-tenancy: shared schema with `tenant_id`; RLS as evaluated defense-in-depth | Proposed (revised 2026-11-24) |
| 0011 | Credit ledger model | Proposed (revised 2026-11-24) |
| 0012 | Provider abstraction, capabilities, and simulator adapter | Proposed (revised 2026-11-24) |
| 0013 | Frontend application structure and component strategy | **Accepted** (owner, 2026-11-25) |
| 0014 | Database access and migration tooling | **Accepted** (owner, 2026-11-25) |
| 0015 | API contracts: OpenAPI-first | **Accepted** (owner, 2026-11-25) |
| 0016 | Structured logging with `log/slog` | **Accepted** (owner, 2026-11-25) |
| 0017 | Git workflow for one owner + multiple AI agents | **Accepted** (owner, 2026-11-25) |
| 0018 | API idempotency contract | **Accepted** (owner, 2026-11-25) |
| 0019 | Monetary representation, scale, and rounding | Proposed — **open** (new 2026-11-24) |
| 0020 | Internal event delivery: transactional outbox, at-least-once | Proposed (new 2026-11-24) |
| 0021 | Message status model and uncertain outcomes | **Accepted — limited scope** (owner, 2026-11-25): eight customer-visible statuses and initial reason codes only |

## 5. Review summary (2026-11-24)

This table is a **reviewer recommendation to help the owner decide**, as written on 2026-11-24. It does not
change any status; the index above shows current statuses.

| ADR | Sufficiently defined? | Depends on external (telco / regulatory / commercial) info? | Recommendation | Outstanding conditions |
| --- | --------------------- | ----------------------------------------------------------- | -------------- | ---------------------- |
| 0001 | Yes | No | Can be accepted now | — |
| 0002 | Yes | No | Can be accepted now | Go module path confirmed at scaffolding |
| 0003 | Yes | No | Can be accepted now | Final executable list in architecture phase |
| 0004 | Yes | No | Can be accepted now | Verify Chi maintenance status at scaffolding |
| 0005 | Yes | No (hosting affects operations, not the choice) | Can be accepted now | Version and HA/DR targets later |
| 0006 | Yes (as a deferral) | Traffic data for the eventual broker choice | Accept the **deferral and criteria** now; broker choice stays open | Measurements per criteria |
| 0007 | Yes (as a deferral) | No | Accept the deferral now | — |
| 0008 | Yes (as a deferral) | Reporting requirements | Accept the deferral now | — |
| 0009 | Yes | No | Can be accepted now | Availability targets refine physical separation |
| 0010 | Partly — RLS needs evaluation | No | Keep Proposed; decide after the RLS evaluation in Phase 3 | RLS evaluation (pooling, transaction-scoped context, roles, performance) |
| 0011 | Structure yes; flows depend on business rules | **Commercial** (billing basis, credit unit, refunds) | Keep Proposed until D-080–D-084 decided | Phase 2 business decisions; ADR-0019 |
| 0012 | Principle yes; interface shape no | **Telco** specifications | Accept the **principle** (abstraction + simulator + capabilities) now; interface shape after a telco spec | Telco technical documentation |
| 0013 | Options defined; choices open | No | Keep Proposed until owner decides D-091 and D-140 | Owner decisions before Phase 3 |
| 0014 | Direction yes; tool choice open | No | Keep Proposed; accept at Phase 3 start after verifying tool maintenance | Dependency approval |
| 0015 | Yes | No | Can be accepted now | Generator tools at scaffolding |
| 0016 | Yes | No | Can be accepted now | — |
| 0017 | Yes | No | Can be accepted now | Owner actions in git-workflow §0 |
| 0018 | Decision-ready; options listed | Client retry behaviour; volume | Keep Proposed; decide D-094 in Phase 2 | D-094a–e |
| 0019 | Options defined; **no selection** | **Commercial** (actual rates, pricing model) | Keep Proposed (open) | D-081, D-092, D-093 |
| 0020 | Yes | No | Can be accepted now | Dedupe retention set when broker/queue chosen |
| 0021 | States defined; policies open | **Commercial and telco** | Keep Proposed until D-095–D-099 decided | Business decisions; telco receipt behaviour |

---

## 6. Records

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

- **Status:** **Accepted** (by owner, 2026-11-25) · **Date proposed:** 2026-11-24 (owner has already stated a monorepo preference;
  this ADR records the specific *layout*)

**Owner decision (2026-11-25):** accepted. Layout as in option A. Go module path: `github.com/hruizjr/sms/backend`.

**Context** — Owner requires one repository for frontend, backend, infrastructure, contracts, and docs.

**Options considered (layout)**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. Top-level `backend/`, `frontend/`, `contracts/`, `infra/`, `scripts/`, `tests/`, `docs/` | Self-explanatory to non-specialists; clear ownership | Go module lives in a subfolder (minor tooling consideration) |
| B. Go module at repository root with `cmd/`, `internal/`, plus `web/` | Idiomatic for Go-only repos | Mixes Go and non-Go concerns at root; less clear |
| C. Polyglot build system (Bazel, Nx) | Powerful caching, cross-language builds | Significant learning and maintenance cost; unjustified now |

**Recommendation** — A (implemented in Phase 1 as folders and READMEs only).

**Consequences** — Simple, discoverable structure. CI runs Go and frontend pipelines based on changed paths.

**Operational cost** — Low. **Missing information** — None. The Go module path
`github.com/hruizjr/sms/backend` is accepted (owner decision above).

---

### ADR-0003: Modular monolith with multiple executables

- **Status:** **Accepted** (by owner, 2026-11-25) · **Date proposed:** 2026-11-24

**Owner decision (2026-11-25):** accepted. Phase 3 creates only the `cmd/api` and `cmd/migrate` executables; further executables need an approved task.

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

**Operational cost** — Low to moderate (a few deployables). **Missing information** — None for Phase 3: the
initial executables `cmd/api` and `cmd/migrate` are accepted. Any further executable (e.g. dispatcher,
telco connectors, scheduler) requires separate owner authorization.

---

### ADR-0004: HTTP router — Chi vs. standard library

- **Status:** **Accepted** (by owner, 2026-11-25) · **Date proposed:** 2026-11-24

**Owner decision (2026-11-25):** accepted. Router: Chi (`github.com/go-chi/chi/v5`), with handlers kept as standard `net/http` handlers.

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

- **Status:** **Accepted** (by owner, 2026-11-25) · **Date proposed:** 2026-11-24

**Owner decision (2026-11-25):** accepted. PostgreSQL **18**, using the latest available 18.x patch release at the time of pinning. PostgreSQL 19 compatibility must be reviewed again before production deployment.

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
this). **Missing information** — Hosting decision; HA/DR targets. (Version decided: PostgreSQL 18, exact
latest 18.x patch pinned at implementation; PostgreSQL 19 reviewed before production.)

---

### ADR-0006: Message broker — defer Kafka/Redpanda, with reconsideration criteria

- **Status:** Proposed · **Date proposed:** 2026-11-24 · **Revised:** 2026-11-24 (outbox semantics moved
  to ADR-0020; reconsideration criteria added)

**Context** — The data plane needs durable buffering between acceptance and dispatch, and between receipt
ingestion and processing. Kafka and Redpanda are candidates. Traffic volume is unverified.

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. Kafka | Industry standard; very high throughput; durable replicated log; large ecosystem | Operationally demanding (clusters, partitions, upgrades, monitoring); significant for a small team |
| B. Redpanda | Kafka-API compatible; single binary; simpler operations | Smaller community; licensing terms must be reviewed; vendor dependence |
| C. PostgreSQL-backed queue (outbox table + workers claiming rows with `FOR UPDATE SKIP LOCKED`) | No new infrastructure; transactional with business data | Lower throughput ceiling than a log broker; table bloat and vacuum must be managed |
| D. Managed cloud queue/stream | No cluster operations | Cloud lock-in; depends on hosting decision |
| E. NATS JetStream / RabbitMQ | Simpler than Kafka for queues | Not in owner's preferences; different semantics |

**Recommendation** — Start with **C** for early messaging milestones, behind an internal publisher/consumer
interface, using the transactional outbox (ADR-0020, needed regardless of broker). Because A and B both
speak the Kafka protocol, client code would work with either if a broker is later adopted.

**Reconsideration criteria (evidence, not arbitrary thresholds).** Reopen this decision when any of the
following is **measured** against **verified** requirements and service-level objectives (SLOs) — the
numeric targets come from the traffic discovery and SLOs once they are set, not from this ADR:
1. Load tests show the PostgreSQL-backed queue cannot sustain the verified peak rate with agreed headroom
   while keeping database utilization within agreed limits.
2. Queue lag (age of the oldest unprocessed item) exceeds the latency objective (e.g. for OTPs) under
   verified load, and tuning (batching, indexing, partitioning, more workers) has been tried.
3. Queue-table churn (bloat, vacuum, lock contention) measurably degrades the transactional workload.
4. Approved requirements need capabilities a log broker provides natively: multiple independent consumer
   groups, long event retention, or replay of history (e.g. for an analytics pipeline).
5. Operational capacity exists to run a broker (staffing, monitoring) or a managed Kafka-compatible service
   is available in the chosen hosting.

**Consequences** — Fewer moving parts early; a migration step later if a broker is adopted (mitigated by the
interface). Load tests must establish the PostgreSQL queue's measured ceiling.

**Operational cost** — C: low. A: high. B: moderate. **Missing information** — Peak rate, retention needs,
number of consumers, hosting model, ops staffing, SLOs.

---

### ADR-0007: Redis — defer, with reconsideration criteria

- **Status:** Proposed · **Date proposed:** 2026-11-24 · **Revised:** 2026-11-24 (criteria added)

**Context** — Redis is proposed for caching and short-lived coordination (e.g. distributed rate limiting).

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. Introduce Redis now | Ready when needed | Another system to secure, monitor, and operate; temptation to store authoritative data in it |
| B. Introduce when a concrete, measured need arises | No premature complexity | Must be added later |

**Recommendation** — B.

**Reconsideration criteria** — Reopen when one of these is measured or required:
1. Rate limits must be shared across several dispatcher or API instances (e.g. one telco throughput limit
   shared by multiple processes), and a database-based or single-owner approach has been shown inadequate
   (accuracy, latency, or load).
2. Profiling shows hot, repeated reads causing measurable database load or latency after query and index
   optimization, and an in-process cache is insufficient (e.g. because of multiple instances needing
   consistent invalidation).
3. A specific approved feature needs short-lived shared state with expiry that PostgreSQL serves poorly.

**Constraints if introduced** — Redis MUST NOT be the source of truth for credits, balances, ledger
entries, idempotency records required for correctness (ADR-0018), or message state. Losing Redis data must
never cause financial or message-state errors.

**Operational cost** — Moderate when introduced. **Missing information** — Number of instances; rate-limit
coordination design (architecture question AQ-9).

---

### ADR-0008: ClickHouse — defer, with reconsideration criteria

- **Status:** Proposed · **Date proposed:** 2026-11-24 · **Revised:** 2026-11-24 (criteria added)

**Context** — ClickHouse is strong for large-scale analytical queries. Reporting requirements and volumes
are unknown.

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. ClickHouse from the start | Fast analytics at scale | Data pipeline, second store, consistency questions, operational cost |
| B. PostgreSQL with pre-aggregated reporting tables first; a read replica for reporting as an intermediate step | Simple; consistent | May not scale to very large history or ad hoc analysis |

**Recommendation** — B initially.

**Reconsideration criteria** — Reopen when one of these is measured or required:
1. Reporting queries miss their approved latency objectives after aggregation tables, partitioning, and
   indexing have been applied.
2. Reporting load measurably degrades the transactional workload even with a read replica.
3. Approved analytics requirements need ad hoc queries over long histories at a volume PostgreSQL cannot
   serve within objectives (requires D-072 and D-073 answers).

ClickHouse, if adopted, holds **derived** data only; PostgreSQL remains authoritative.

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

### ADR-0010: Multi-tenancy — shared schema with `tenant_id`; RLS as evaluated defense-in-depth

- **Status:** Proposed · **Date proposed:** 2026-11-24 · **Revised:** 2026-11-24 (RLS role and evaluation
  requirements clarified)

**Context** — Many tenants, strict isolation, potentially large data volume.

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. Shared tables with `tenant_id`, enforced in application | Simple; scales to many tenants; easy cross-tenant admin reporting | One missed filter leaks data |
| B. A + PostgreSQL Row-Level Security (RLS) as defense-in-depth | Database-level backstop against missed tenant filters | Tenant context must be set safely per transaction; admin and background access must be designed; performance must be measured |
| C. Schema per tenant | Stronger separation | Migrations multiplied by tenants; operational burden grows |
| D. Database per tenant | Strongest isolation | Expensive and complex; only for special contractual cases |

**Recommendation** — A as the baseline for all tenant data, with B **adopted only after evaluation** (Phase 3
spike). D remains an option only if a customer contract requires dedicated infrastructure.

**What RLS is and is not**
- RLS restricts **which rows** a database session can see or change. It is **not a substitute for
  application authorization**: it does not know user roles or permissions, which actions are allowed, or
  business rules. Application-level tenant scoping and permission checks remain mandatory with or without RLS.
- RLS protects only if the database role used by the application is actually subject to it. PostgreSQL
  documentation states that **superusers and roles with `BYPASSRLS` always bypass** row security, and that
  **table owners normally bypass it** unless the table uses `ALTER TABLE ... FORCE ROW LEVEL SECURITY`.
  The application must therefore connect as a non-owner, non-superuser role without `BYPASSRLS`
  (or tables must force RLS).
- With RLS enabled and no applicable policy, PostgreSQL applies **default deny** (no rows visible).

**Evaluation required before adoption**
1. **Transaction-scoped tenant context.** The tenant identifier must be set per transaction (e.g.
   `set_config('app.tenant_id', $1, true)`, where `true` limits it to the current transaction), never as
   a session-level setting that could survive on a pooled connection and leak into another tenant's request.
2. **Connection pooling.** Confirm behaviour with the application pool (pgx) and any external pooler
   (e.g. transaction-mode pooling, where session state is unreliable). Every tenant-scoped query must run
   inside a transaction that sets the context.
3. **Fail closed.** Policies must return no rows when the context is missing (e.g. reading the setting with
   `current_setting('app.tenant_id', true)`, which returns NULL when unset, so comparisons match nothing).
4. **Cross-tenant access.** Admin reporting, reconciliation, and background jobs that legitimately span
   tenants use a separate, minimal, audited path (dedicated role or explicit policies) — never by
   disabling RLS ad hoc.
5. **Performance.** Measure policy overhead on hot queries and high-volume data-plane tables; indexes must
   lead with `tenant_id`.
6. **Tests.** Missing context returns no rows; a pooled connection reused by another tenant cannot see the
   previous tenant's data; the application role cannot bypass RLS.

**Operational cost** — Low to moderate. **Missing information** — RLS evaluation results; dedicated
infrastructure requirements; sub-account/reseller hierarchy (D-064).

---

### ADR-0011: Credit ledger model

- **Status:** Proposed · **Date proposed:** 2026-11-24 · **Revised:** 2026-11-24 (monetary representation
  moved to ADR-0019; rules reference [financial integrity](../architecture/financial-integrity.md))

**Context** — Credits must never be double-spent, lost, or silently changed, even with many concurrent
sends, duplicate events, or duplicate receipts.

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. Single mutable `balance` column | Simple | No history; errors invisible; hard to audit |
| B. Append-only ledger entries + balance row maintained in the same transaction and verifiable against the ledger + reservations | Full audit trail; reconcilable; concurrency-safe with row locks/conditional updates | More design work |
| C. External billing system | Offloads work | Integration complexity; latency in the send path |

**Recommendation** — B, following the rules F1–F8 and the entry types in
[financial integrity](../architecture/financial-integrity.md):
- entry types `top_up`, `reservation`, `reservation_release`, `charge`, `refund`, `reversal`, `adjustment`;
- each entry carries a unique idempotency key so duplicates are impossible;
- corrections are compensating entries, never edits or deletes;
- balance invariants enforced by the database in the same transaction;
- amount representation and scale per **ADR-0019 (open)** — this ADR does not choose one.

**Consequences** — Strong correctness guarantees. The per-account balance row is a contention point for very
high-volume tenants (mitigations such as batched reservations to be evaluated with load tests).

**Operational cost** — Low. **Missing information** — D-080 to D-084 (settlement point, credit unit,
payment model, refunds, pricing structure); ADR-0019.

---

### ADR-0012: Provider abstraction, capabilities, and simulator adapter

- **Status:** Proposed · **Date proposed:** 2026-11-24 · **Revised:** 2026-11-24 (provider capabilities and
  receipt normalization added)

**Context** — Owner requires Smart and Globe as independent adapters behind a common interface. Providers
will differ in protocol, receipt behaviour, encodings, limits, and extra features.

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. Telco logic in the dispatcher with branches per telco | Quick start | Tight coupling; every new telco touches core code |
| B. `Provider` interface; one adapter package per telco; simulator adapter; capability description; shared contract test suite | Isolation; testability; new telcos added without core changes | Interface must fit real telco behaviour |
| C. Third-party SMS gateway software in front of telcos | Less protocol code | Another system to operate; fit and licensing unknown |

**Recommendation** — B, with these elements:
- **Capabilities, not telco names.** Each adapter declares its capabilities (for example: receipts
  supported, per-segment or per-message receipts, inbound messages, supported encodings, maximum segments,
  status-query support, provider-side duplicate detection, throughput limits from configuration). Core logic
  consults capabilities and never branches on a provider's identity. Every declared capability must cite
  telco documentation; unknown capabilities are declared as unknown and treated conservatively.
- **Normalization inside adapters.** Adapters map provider responses and receipts to normalized results
  and statuses ([messaging reliability §5.1](../architecture/messaging-reliability.md#51-adapter-boundary));
  raw codes are kept as opaque data for audit and reconciliation.
- **Outcome classification inside adapters.** Each adapter classifies responses as success, retryable,
  permanent, or ambiguous, from documented error codes.
- **Simulator** configurable to emulate capability profiles and failure modes.
- The concrete interface is drafted **after** at least one telco specification is received; the simulator is
  built against a provisional interface first. The SMPP client library is a separate ADR made with specs in hand.

**Operational cost** — Low. **Missing information** — Telco protocols and behaviours (telco checklist).

---

### ADR-0013: Frontend application structure and component strategy

- **Status:** **Accepted** (by owner, 2026-11-25) · **Date proposed:** 2026-11-24 · **Revised:** 2026-11-24 (component strategy
  direction proposed by the owner recorded)

**Owner decision (2026-11-25):** accepted with these choices:
- React, TypeScript (strict), Vite, Tailwind CSS **v4**; pnpm **10** with an exact version pinned.
- One application initially, `client-portal`, plus a shared UI package, `packages/ui`. An `admin-portal`
  application is added later, when admin functionality begins (D-091).
- Router, data-fetching, and form libraries are **deferred to Phase 4** and need separate approval.
- The headless UI library decision (D-140) is **deferred**.

**Context** — The selected stack is React, TypeScript, Vite, and Tailwind CSS. Two audiences (clients, our
admins). The owner wants no unnecessary UI or styling libraries.

**Component strategy (accepted direction; library choice deferred)**
- **Tailwind CSS remains the only styling system.**
- Carefully selected, maintained, accessible **headless** (unstyled) UI primitives **may** be used for
  complex interactive behaviours: dialogs, menus, dropdowns, popovers, comboboxes, and similar. These
  provide keyboard interaction and focus management, which are difficult to build correctly from scratch.
- Simple components (buttons, inputs, cards, badges, tables) are built in-house with Tailwind.
- All components must be keyboard accessible with appropriate focus management (WCAG 2.2 AA target).
- A large, opinionated UI framework (pre-styled component suites) is **not** introduced for convenience.
- **Selection criteria** before choosing any library: maintenance activity and release history;
  documented accessibility behaviour (keyboard, focus, screen readers); licence; compatibility with the
  pinned React version and Tailwind; dependency footprint and bundle size; security history; ability to
  adopt per-component.
- **The library choice (D-140) is deferred by the owner.** No package is added until separately approved.

**Decisions (accepted 2026-11-25)**
| Topic | Decision | Alternatives considered, not selected |
| ----- | -------- | ------------------------------------- |
| Application structure (D-091) | pnpm workspace with **one application initially, `client-portal`**, and a shared **`packages/ui`**; `admin-portal` is added when admin functionality begins | One application serving both clients and admins with role-based routes (admin code would ship to customer browsers); starting with two applications immediately |
| Package manager | **pnpm 10**, exact version pinned | npm |
| Routing | **Deferred to Phase 4**; requires separate owner approval | — (candidates such as React Router or TanStack Router are evaluated then) |
| Server data fetching | **Deferred to Phase 4**; requires separate owner approval; until then the typed API client is used directly | — (candidates such as TanStack Query are evaluated then) |
| Form validation | **Deferred to Phase 4**; requires separate owner approval | — (candidates such as a schema library plus a form library are evaluated then) |
| Headless UI primitives (D-140) | **Deferred** | — (selection criteria above apply when proposed) |

Any deferred library is a dependency requiring owner approval under `AGENTS.md` §2 rule 5 when it is proposed.

**Operational cost** — Low. **Missing information** — None for the Phase 3 scaffold. Later decisions: routing,
data fetching, and forms (Phase 4); D-140 when complex widgets are needed. Implementation-time checks: exact
versions and Node.js 26 LTS compatibility (Group 4).

---

### ADR-0014: Database access and migration tooling

- **Status:** **Accepted** (by owner, 2026-11-25) · **Date proposed:** 2026-11-24 · **Revised:** 2026-11-24 (selection criteria added)

**Owner decision (2026-11-25):** accepted. Data access: `pgx/v5` with `sqlc`. Migration tool: **goose**. During the Phase 3 walking skeleton, migrations contain no business-specific tables.

**Options considered (data access)**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. `pgx` + hand-written SQL | Full control, performance | Boilerplate; manual scanning errors |
| B. `pgx` + `sqlc` (type-safe Go generated from SQL) | Plain SQL reviewed as SQL; compile-time type safety; no ORM magic | Code generation step |
| C. ORM (GORM, ent) | Less SQL | Hidden queries; harder tuning; harder to reason about transactions and locking |

**Migration tool** — **goose** (selected by the owner, 2026-11-25). Alternatives considered, not selected:
`golang-migrate`, `atlas`.

**Selection criteria (used to choose goose; confirmed against the pinned version at implementation)**
- Plain SQL migration files reviewed like code; timestamp-based versions (parallel agents).
- Each migration in a transaction by default, with an explicit opt-out for statements that cannot run in
  a transaction (e.g. `CREATE INDEX CONCURRENTLY`).
- Locking so two deployments cannot run migrations concurrently.
- Programmatic execution from the `migrate` executable; clear status reporting.
- Forward-only production use (down migrations optional, not relied upon).
- Active maintenance and compatible licence.

**Decision** — B for data access (`pgx/v5` + `sqlc`) and goose for migrations. The data access layer must
support transaction-scoped tenant context if RLS is adopted (ADR-0010), and exact monetary types per ADR-0019.

**Operational cost** — Low. **Missing information** — None for the tool choice. Implementation-time checks:
exact versions and current maintenance status of `pgx/v5`, `sqlc`, and goose, and confirmation that the pinned
goose version meets the selection criteria (recorded in the Group 2 PR). ADR-0010 and ADR-0019 remain Proposed.

---

### ADR-0015: API contracts — OpenAPI-first

- **Status:** **Accepted** (by owner, 2026-11-25) · **Date proposed:** 2026-11-24 · **Revised:** 2026-11-24 (idempotency and money
  details delegated to ADR-0018 and ADR-0019)

**Owner decision (2026-11-25):** accepted. OpenAPI specification in `contracts/openapi/` as the source of truth, with TypeScript types generated from it. This does **not** accept ADR-0018 (idempotency) or ADR-0019 (monetary representation), which remain Proposed.

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. Spec-first: OpenAPI files in `contracts/`, TypeScript types generated, Go validated/generated against spec | Single source of truth; public docs come free; contract tests | Spec authoring discipline |
| B. Code-first: generate spec from Go annotations | Less duplication for backend developers | Spec quality depends on annotations; frontend waits on backend |
| C. gRPC/GraphQL | Strong typing / flexible queries | Not REST; public SMS API consumers expect REST |

**Recommendation** — A. Conventions: `/v1` versioning, `snake_case` JSON, RFC 9457 errors, cursor pagination.
Idempotency keys follow ADR-0018; monetary amounts are transmitted as strings with documented scale per
ADR-0019 (proposed).

**Operational cost** — Low. **Missing information** — TypeScript type generation (`openapi-typescript`) and
specification linting (`@redocly/cli`) are approved (decision log §7). Any Go-side code generation or
request-validation tooling is not chosen and would need owner approval.

---

### ADR-0016: Structured logging with `log/slog`

- **Status:** **Accepted** (by owner, 2026-11-25) · **Date proposed:** 2026-11-24

**Owner decision (2026-11-25):** accepted. Structured logging with the standard library `log/slog`.

**Options** — A. Standard library `log/slog` · B. zap / zerolog.
**Recommendation** — A: structured, part of the standard library (no dependency), adequate performance for
our needs; revisit only if profiling shows logging as a bottleneck.

**Operational cost** — None. **Missing information** — Log aggregation platform (hosting decision).

---

### ADR-0017: Git workflow for one owner + multiple AI agents

- **Status:** **Accepted** (by owner, 2026-11-25) · **Date proposed:** 2026-11-24 · **Revised:** 2026-11-24 (current repository state noted)

**Owner decision (2026-11-25):** accepted. `main` is the default branch and is protected (owner-confirmed): pull requests required, zero required approvals, conversation resolution required, force pushes and deletion blocked. Required CI status checks are added once workflows exist. Phase 3 is delivered as one Draft PR per implementation group, each reviewed and merged by the owner before the next starts.

**Context** — One human reviewer; several agents possibly working in parallel. At the time of proposal
(2026-11-24) the repository had no `main` branch and no branch protection; those owner actions were completed
before acceptance — see [git-workflow.md §0](git-workflow.md#0-repository-state).

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. Trunk-based: protected `main`, short-lived task branches, PRs, squash merge | Simple; clean history; owner controls every merge | Owner review is the bottleneck |
| B. GitFlow (develop/release/hotfix branches) | Formal release management | Overhead without benefit at this stage |

**Recommendation** — A, detailed in [git-workflow.md](git-workflow.md).

**Operational cost** — Low. **Missing information** — None.

---

### ADR-0018: API idempotency contract

- **Status:** **Accepted** (by owner, 2026-11-25) · **Date proposed:** 2026-11-24

**Owner decision (2026-11-25, PR #7 review):** accepted, with decisions D-094a–e and these clarifications:
- The key is **required** on message submission; scope is **tenant (account) + operation**; business
  rejections such as insufficient credit are **not recorded**; retention is **seven days**; bulk sending
  is deferred to a later release (D-094e).
- For the same tenant, operation, and key: a **different normalized request body** returns
  `422 idempotency-key-reused`, **whether the original request is still processing or has completed**; an
  **identical body** returns `409 idempotency-request-in-progress` while the original is still processing,
  and a replay of the original `202` response and body with `Idempotent-Replayed: true` once it has completed.
- `X-Request-ID` and other tracing or transport headers are **not** part of the comparison; the comparison
  is based on the normalized request body.
- The replay header name (`Idempotent-Replayed`) remains provisional.

**Context** — Clients retry requests after timeouts. Without idempotency, a retry can create a second
message, a second charge, and a second SMS to a real person.

**Options considered** — key scope per tenant + operation, per API credential + operation, or per tenant
across operations; required vs. optional keys; which outcomes are recorded; retention periods. Full option
tables with advantages and disadvantages are in [API idempotency](../architecture/api-idempotency.md).

**Recommendation** — The contract in [API idempotency](../architecture/api-idempotency.md): `Idempotency-Key`
header (aligned with the IETF Internet-Draft, which is not yet a standard); scope tenant + operation;
request fingerprint; replay of the original response for the same request; `422` for a reused key with a
different request; `409` while the original is in progress; concurrency resolved by a database unique
constraint in the acceptance transaction.

**Consequences** — Prevents duplicate operations **inside our platform**. It does **not** prevent duplicate
submissions to a telco; that is governed by ADR-0021 and
[messaging reliability §4](../architecture/messaging-reliability.md#4-provider-submission-and-ambiguous-outcomes).

**Operational cost** — Low (one table, one constraint, retention cleanup). **Missing information** — Owner
decisions D-094a–e (required vs. optional, scope, recorded outcomes, retention, bulk semantics).

---

### ADR-0019: Monetary representation, scale, and rounding

- **Status:** Proposed — **open; no option selected** · **Date proposed:** 2026-11-24

**Context** — Ledger amounts and rates must be exact. The customer credit unit, currency, the precision of
actual telco and retail rates, and rounding rules are not yet known. Centavos may not be sufficient for
wholesale rates.

**Options considered** — (A) integer centavos; (B) integer in a finer fixed sub-unit; (C) exact decimal
(`numeric` + a Go decimal type); (D) hybrid: decimal rates, scaled-integer ledger amounts. Trade-offs are
in [financial integrity §3](../architecture/financial-integrity.md#3-representation-options-decision-deferred-to-adr-0019).

**Fixed regardless of option** — no IEEE floating point anywhere in monetary paths; documented currency
and scale per field; one shared rounding function; amounts transmitted in APIs as strings (proposed);
frontend never performs monetary arithmetic.

**Decision** — Deferred until actual telco rates and the commercial pricing model are available (Phase 2).

**Operational cost** — Depends on option. **Missing information** — D-081, D-092, D-093, D-127; telco
checklist §15.

---

### ADR-0020: Internal event delivery — transactional outbox, at-least-once

- **Status:** Proposed · **Date proposed:** 2026-11-24 (split out of ADR-0006 so it can be decided
  independently of broker choice)

**Context** — State changes must reliably produce events (e.g. "message accepted" → dispatch), without
publishing events for rolled-back changes and without losing events for committed ones.

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. Publish directly to the queue inside request handling (dual write) | Simple | A crash between the database commit and the publish loses the event, or an event is published for a rolled-back change |
| B. Transactional outbox + publisher, at-least-once, idempotent consumers | Atomic with the business change; no loss of committed events; broker-agnostic | Duplicates are possible and every consumer must handle them; publication delay |
| C. Change data capture from the database log | No outbox writes in application code | Additional infrastructure and operational complexity |

**Recommendation** — B, with the guarantees, non-guarantees, publisher and consumer requirements, and
ordering rules in [messaging reliability §2](../architecture/messaging-reliability.md#2-internal-events-and-the-transactional-outbox).
The outbox does **not** provide exactly-once publication.

**Operational cost** — Low. **Missing information** — Dedupe record retention depends on the queue/broker
redelivery window (ADR-0006).

---

### ADR-0021: Message status model and uncertain outcomes

- **Status:** **Accepted — limited scope** (by owner, 2026-11-25) · **Date proposed:** 2026-11-24

**Owner decision (2026-11-25, PR #7 review):** accepted, **limited to** the approved customer-visible status
model: the eight statuses `accepted`, `provider_accepted`, `submission_uncertain`, `delivered`,
`undelivered`, `unknown`, `rejected`, `failed`, and the initial reason-code vocabulary (ten codes,
extensible), as defined in the [public API contract](../../contracts/openapi/README.md#3-status-reference-customer-visible-statuses).
This acceptance does **not** settle, and these remain **separate open decisions**:
- the **billing treatment of `unknown`** and other uncertain messages, and the effect of late receipts after
  a release or refund (D-096);
- **webhook behaviour** (D-099, webhook part);
- the ambiguous-submission policy (D-095), segment roll-up (D-097), and conflicting final receipts (D-098).

**Context** — API acceptance, queueing, provider submission, provider acknowledgment, final receipts, and
customer-visible status are different milestones. Timeouts and missing receipts do not prove
non-delivery.

**Options considered**
| Option | Advantages | Disadvantages |
| ------ | ---------- | ------------- |
| A. Simple sent/failed/delivered model; timeouts and missing receipts treated as failures | Simple for customers | Misreports delivered messages as failed; wrong refunds; invites duplicate resends |
| B. Explicit milestones with uncertain states (`SUBMIT_UNCERTAIN`, `UNKNOWN`), precedence rules, and auditable history | Accurate; reconcilable; late receipts handled | More states to explain to customers |

**Recommendation** — B, as described in [messaging reliability §3–§5](../architecture/messaging-reliability.md#3-message-lifecycle-milestones).
The customer-visible status set is a mapping decided separately (D-099).

**Operational cost** — Low. **Missing information** — D-095 to D-099; telco receipt behaviour (D-104,
D-114–D-116).

---

## 7. Owner decisions — Phase 3 walking skeleton (2026-11-25)

The owner approved the Phase 3 recommendations with the adjustments below. **The approval is limited to these
decisions and the Phase 3 walking skeleton scope.** All other ADRs and later-phase functionality remain
subject to separate owner approval.

### 7.1 Toolchain

| Item | Decision | Note |
| ---- | -------- | ---- |
| Go | **1.27.4** | Pinned exactly; same version locally and in CI. Availability verified before pinning in Group 1; if unavailable, report to the owner — no substitution. |
| Node.js | **26 LTS** | Adopted after verifying compatibility of the approved frontend dependencies. Exact version pinned; same locally and in CI. |
| pnpm | **10**, exact version pinned | Same locally and in CI. |
| PostgreSQL | **18**, latest available 18.x patch | Review PostgreSQL 19 compatibility before production deployment (ADR-0005). |

### 7.2 Dependencies approved for Phase 3

Each dependency is still justified in the pull request that introduces it (`AGENTS.md` §2 rule 5: purpose,
licence, maintenance status, security record), with exact versions pinned at that time.

| Area | Approved |
| ---- | -------- |
| Go libraries | `github.com/go-chi/chi/v5`, `github.com/jackc/pgx/v5`, `github.com/pressly/goose/v3`, `github.com/prometheus/client_golang`. Configuration uses the standard library only. |
| Go tools | `sqlc`, `golangci-lint`, `govulncheck` |
| Frontend | `react`, `react-dom`, `vite`, `@vitejs/plugin-react`, `typescript`, `tailwindcss` with `@tailwindcss/vite` (v4), `vitest`, `@testing-library/react`, `@testing-library/jest-dom`, `jsdom`, `eslint`, `typescript-eslint`, `eslint-plugin-react-hooks`, `prettier`, `openapi-typescript` |
| API contract | `@redocly/cli` (OpenAPI linting and validation) |
| CI | Official `actions/*`, `golangci-lint-action`, `gitleaks-action`, GitHub dependency review — all **pinned to immutable commit SHAs** |

### 7.3 Engineering conditions

1. Basic automated CI is set up starting in **Group 1**; coverage is expanded in **Group 5**.
2. The Prometheus metrics endpoint is **not publicly exposed**.
3. GitHub Actions are pinned to immutable commit SHAs.
4. Database migrations contain **no business-specific tables** during the walking skeleton.
5. No Kafka, Redis, ClickHouse, SMPP, or actual SMS sending.
6. No customer authentication, billing, tenant management, or other business functionality.

### 7.4 Implementation workflow

Six groups, one Draft PR each; the owner reviews and merges each group before the next begins:
Group 0 (record decisions), 1 (Go API skeleton with basic CI), 2 (database), 3 (API contract), 4 (frontend),
5 (local stack, expanded CI, developer documentation). The PostgreSQL Row-Level Security evaluation
(ADR-0010, still Proposed) is a separate spike after the skeleton and before Phase 4.

---

## 8. Owner decisions — public API contract (2026-11-25)

Recorded from the owner's review of PR #7 (Phase 3 Group 3, OpenAPI contract). **The approval is limited to
these decisions**; no other ADR status changed.

| Item | Decision |
| ---- | -------- |
| ADR-0018 (API idempotency) | **Accepted**, with D-094a–e and the outcome clarifications recorded in ADR-0018 above |
| ADR-0021 (status model) | **Accepted — limited** to the eight customer-visible statuses and the initial reason-code vocabulary; D-096 billing of uncertain messages and D-099 webhook behaviour remain open |
| Bulk sending (D-094e; MVP-08, MVP-09) | Bulk sending through the public API and through the client portal (uploaded recipient file) is **deferred to a later release**; the MVP supports single-message sending through the portal and the public API |
| Portal recipient scope (MVP-09) | One portal send operation accepts **exactly one recipient**: no pasted recipient lists, recipient-file uploads, contact-group sends, or multi-recipient selection. Separate single-recipient sends remain subject to rate limits and abuse controls |
| Campaign approval (MVP-13; D-025, D-087) | MVP-13 changed from Essential to **Deferrable**: dedicated campaign-level approval workflows are deferred beyond the MVP (no bulk sending or campaign-sending workflow in the MVP). Essential abuse controls stay in the MVP: admin-controlled tenant onboarding and activation, sender ID approval before use, tenant/API-key velocity limits, abuse alerts and an operational kill switch, and audit logging of administrative and security actions. The approval policy is **not** decided: thresholds, first-campaign policies, and turnaround remain open |
| Public API contract details | D-143 (Bearer API key; `messages:send`, `messages:read`), D-144 (1,600-character maximum; segment count deferred to product/provider policy), D-145 (domain stays a placeholder), D-146 (`402` for insufficient credit), D-147 (explicit `sender_id`), recorded in the [discovery register](../requirements/discovery-register.md) |
