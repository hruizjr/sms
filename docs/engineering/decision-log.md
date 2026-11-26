# Decision Log (Architecture Decision Records)

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Living document — twelve ADRs accepted (2026-11-25 and ADR-0010 on 2026-11-26); others Proposed | Project owner | 2026-11-26 |

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
**In Phase 3 Group 4 (2026-11-25) the owner approved Go code generation from the OpenAPI contract under
ADR-0015** (see [§9](#9-owner-decisions--go-contract-integration-phase-3-group-4-2026-11-25)); no ADR status
changed.
**In Phase 3 Group 5 (2026-11-25) the owner approved the frontend foundation under ADR-0013** (see
[§10](#10-owner-decisions--frontend-foundation-phase-3-group-5-2026-11-25)); no ADR status changed.
**In Phase 3 Group 6 (2026-11-26) the owner approved the local development stack and expanded CI** (see
[§11](#11-owner-decisions--local-development-stack-and-expanded-ci-phase-3-group-6-2026-11-26)); no ADR
status changed.
**In Phase 3 Group 7 (2026-11-26) the owner approved the PostgreSQL Row-Level Security evaluation spike**
(see [§12](#12-owner-decisions--rls-evaluation-spike-phase-3-group-7-2026-11-26)). **After reviewing the
evaluation, the owner accepted ADR-0010 on 2026-11-26** (decisions RLS-1 to RLS-5,
[§12.1](#121-owner-decisions-on-rls-adoption-2026-11-26)): option B for tenant-owned **control-plane** tables. The
RLS strategy for high-volume **data-plane** tables remains deferred.
**Phase 3 was closed on 2026-11-26** (see [§13](#13-phase-3-closure-2026-11-26)). No ADR status changed with
the closure.
**On 2026-11-26 the owner approved the MVP scope** (Phase 2 exit criterion EX-1, see
[§14](#14-owner-decisions--mvp-scope-phase-2-ex-1-2026-11-26)). No ADR status changed.

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
| 0010 | Multi-tenancy: shared schema with `tenant_id`; RLS as evaluated defense-in-depth | **Accepted** (owner, 2026-11-26): option B for tenant-owned control-plane tables; high-volume data-plane tables **deferred** (RLS-5) |
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

- **Status:** **Accepted** (by owner, 2026-11-26) — **option B for tenant-owned control-plane tables**; high-volume
  data-plane tables excluded and deferred · **Date proposed:** 2026-11-24 · **Revised:** 2026-11-24 (RLS role and
  evaluation requirements clarified)

**Owner decision (2026-11-26, after the Phase 3 Group 7 evaluation; decisions RLS-1 to RLS-5 in
[§12.1](#121-owner-decisions-on-rls-adoption-2026-11-26)):**

1. **Adopted (RLS-1):** option B, PostgreSQL Row-Level Security as defense-in-depth on **tenant-owned
   control-plane tables**, on top of mandatory application-level tenant filtering. Adoption is conditional on
   all requirements in [evaluation §10](../architecture/rls-evaluation.md#10-conditions-of-adoption-adopted-by-the-owner-rls-1).
2. **Policy (RLS-2):** `tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid`, with equivalent
   `USING` and `WITH CHECK` expressions. The tenant context is set **transaction-locally through one application
   helper**, which rejects missing or invalid tenant IDs before any tenant query runs. Session-level tenant
   settings are never used.
3. **Cross-tenant access (RLS-3):** a dedicated reporting role with its own credentials and connection pool,
   explicit read-only policies, and the minimum necessary table and column privileges. Cross-tenant writes need
   separate approval and narrowly scoped policies. `BYPASSRLS` is reserved for the migration role and a
   separately controlled break-glass role, never for application, reporting or worker connections.
   Cross-tenant administrative operations are audited in the application.
4. **Prerequisite (RLS-4):** database role separation is implemented in a separate, explicitly approved
   development group, before or alongside the first tenant-owned business table. The API must not use the
   bootstrap PostgreSQL superuser.
5. **Deferred (RLS-5):** this decision does **not** extend to high-volume data-plane (message) tables. Their RLS
   strategy is evaluated separately once their design and workload are known (write throughput, partitioning,
   query patterns, representative volume).

The original analysis below is kept as written.

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

**Evaluation (2026-11-26, Phase 3 Group 7):** results and the recommendation are in
[RLS evaluation](../architecture/rls-evaluation.md). The owner accepted this ADR on the same day, with the scope
and conditions above.

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
| API contract | `@redocly/cli` (OpenAPI linting and validation). Go code generation from the contract (added 2026-11-25, Group 4): `oapi-codegen` v2.8.2 (tool) and `github.com/oapi-codegen/runtime` v1.9.0 (library), see §9 |
| CI | Official `actions/*`, `golangci-lint-action`, `gitleaks-action`, GitHub dependency review — all **pinned to immutable commit SHAs** |

### 7.3 Engineering conditions

1. Basic automated CI is set up starting in **Group 1**; coverage is expanded in **Group 6** (numbered
   Group 5 before the renumbering in §9).
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

**Renumbered by the owner on 2026-11-25 (§9, decision D1): seven groups.** Group 4 is now **Go API contract
integration and generated types**; the frontend becomes **Group 5**, and the local stack, expanded CI and
developer documentation become **Group 6**. Earlier records that say "Group 4 (frontend)" or "Group 5"
(for example ADR-0013's implementation-time checks) refer to those groups under their old numbers; they are
kept unchanged as written.

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

**Further MVP scope decisions (owner, 2026-11-26, PR #7 review).** No ADR status changed.

| Item | Decision |
| ---- | -------- |
| Contact book and groups (MVP-11) | **Deferrable**: saved contact management and contact groups are deferred from the MVP. The MVP portal takes a manually entered mobile number for each single-recipient message; customers may submit several separate messages; there is no contact-group sending. Saved contacts may be introduced independently in a later release |
| Campaign scheduling (MVP-12) | **Deferrable**: campaign scheduling is deferred alongside campaign approval (MVP-13) and bulk sending (MVP-08, MVP-09). This does not permanently rule out a separate scheduled-single-message feature in a future release |

---

## 9. Owner decisions — Go contract integration (Phase 3 Group 4, 2026-11-25)

Recorded from the owner's approval of the Group 4 proposal (decisions D1–D7). They implement
**ADR-0015 (API contracts — OpenAPI-first)** on the Go side. ADR-0015's text, rationale and status
(**Accepted**) are unchanged. Its note that "any Go-side code generation or request-validation tooling is not
chosen and would need owner approval" is answered for **code generation** by D2–D3 below. **Request
validation stays unchosen** (D6). No other ADR changed.

| # | Topic | Decision |
| - | ----- | -------- |
| D1 | Group numbering | Phase 3 has **seven** groups: Group 4 is Go API contract integration and generated types; the frontend becomes Group 5; local stack, expanded CI and developer documentation become Group 6 (§7.4) |
| D2 | Go code generator | **`oapi-codegen` v2** (`github.com/oapi-codegen/oapi-codegen/v2`), pinned at **v2.8.2**, as a development tool. Generated code uses **`github.com/oapi-codegen/runtime` v1.9.0** at run time. Alternative considered, not selected: `ogen` (its own router instead of Chi, `go-faster/jx` and OpenTelemetry at run time, less documented OpenAPI 3.1 support) |
| D3 | Generated output | Models, Chi server routes with parameter binding, and the strict (typed request/response) server interface, in the package `backend/internal/publicapi`, generated from the unmodified `contracts/openapi/public-api-v1.yaml` |
| D4 | Production routes | `/v1/*` is **not mounted** in the production router during Group 4; existing production responses are unchanged. The generated routes are mounted only in tests until the first endpoint is implemented |
| D5 | Pinning the generator | A **separate tools module**, `backend/tools/go.mod` (Go `tool` directive, checksums in `tools/go.sum`), so the generator's dependencies stay out of the API module and its binaries |
| D6 | Request validation and authentication | **Deferred** to the first endpoint implementation. Generated parameter binding is not full OpenAPI request validation |
| D7 | OpenAPI 3.1 fallback | Verify generation against the real specification first, and stop and report if it falls short |

**Version policy.** Releases younger than seven days were not used, consistent with the npm
`minimum-release-age` in `contracts/.npmrc`. oapi-codegen v2.9.0 (2026-11-21) and runtime v1.9.1
(2026-11-20) were therefore not selected. Runtime v1.9.0 was preferred over v1.8.0 because v1.8.0 also pulls
the Gin, Echo and Iris web frameworks into the module graph.

**Dependencies added** (licences checked in the downloaded sources, 2026-11-25):

| Module | Version | Licence | Where | Why |
| ------ | ------- | ------- | ----- | --- |
| `github.com/oapi-codegen/oapi-codegen/v2` | v2.8.2 | Apache-2.0 | `backend/tools` only (never linked into a binary) | Generator (D2) |
| `github.com/oapi-codegen/runtime` | v1.9.0 | Apache-2.0 | `backend/go.mod` (direct) | Parameter binding and JSON body decoding used by the generated code |
| `github.com/apapsch/go-jsonmerge/v2` | v2.0.0 | MIT | `backend/go.mod` (indirect, via runtime) | — |
| `github.com/google/uuid` | v1.6.0 | BSD-3-Clause | `backend/go.mod` (indirect, via runtime) | — |
| `github.com/getkin/kin-openapi` | **v0.149.0** (security override; owner decision, PR #8 review) | MIT | `backend/tools` only | oapi-codegen v2.8.2 requires v0.142.0, which is affected by GO-2026-6789 (fixed in v0.145.0) and GO-2026-6795 (fixed in v0.148.0); govulncheck reports both as reachable in the generator. Versions before v0.149.0 are also affected by GO-2026-6800 and GO-2026-6785 (both fixed in v0.149.0). All four are crashes or panics on crafted input. v0.149.0 (released 2026-08-28) is the first release fixing all four. The generated output is byte-identical with v0.142.0, v0.148.0 and v0.149.0. This raises a transitive dependency of the approved tool. It does not change the tool, the output, or anything linked into the API |

The tools module's other transitive dependencies (YAML and JSON-path parsers, `golang.org/x/{mod,sync,text,tools}`)
are MIT, Apache-2.0 or BSD-3-Clause licensed, and locked by checksum in `backend/tools/go.sum`.

**Engineering conditions.**

1. Generated code (`backend/internal/publicapi/*.gen.go`) is **never edited by hand**. Every change comes
   from the specification or the generator configuration, through `go generate ./internal/publicapi/...`
   run from `backend/`.
2. CI regenerates the code on Linux and on Windows. It fails when the committed code is stale or
   hand-edited, when generation creates uncommitted files, or when two runs differ.
3. `govulncheck` scans both Go modules: the API module and the tools module.
4. Generated **parameter binding** (type parsing of path, query and header values, a required
   `Idempotency-Key`, JSON decoding) is not **request validation** (patterns, lengths, ranges, enums,
   required body fields, `additionalProperties: false`, null handling, `Content-Type`) and not
   **authentication or authorization**. Both are required before any `/v1` endpoint serves traffic (D6).

---

## 10. Owner decisions — frontend foundation (Phase 3 Group 5, 2026-11-25)

Recorded from the owner's approval of the Group 5 proposal (decisions F1–F8 and additional requirements).
**The approval is limited to these decisions.** No ADR status changed. ADR-0013's text is unchanged, and
these choices implement it. ADR-0013's "implementation-time checks (Group 4)" refer to this group under its
old number (§7.4).

| ID | Topic | Decision |
| -- | ----- | -------- |
| F1 | TypeScript | **6.0.3**. 7.0.2 is not supported by `typescript-eslint` 8.x (peer range below 6.1.0) |
| F2 | Packages beyond §7.2 | Approved: `@types/react`, `@types/react-dom`, `@testing-library/dom`, `@eslint/js`, `prettier-plugin-tailwindcss`. `@types/node` only if demonstrably necessary: **not added**, because type checking works without it (the build check script is plain JavaScript) |
| F3 | OpenAPI frontend types | **Deferred.** No `openapi-typescript`, no `lib/api/`, no generation scripts, no generated public API types. The public `/v1` API uses API keys, which must never be used in a browser; the portal needs a separately approved, session-authenticated portal API contract |
| F4 | Routing | **No router** in Group 5. One view at `/` (ADR-0013) |
| F5 | Workspace | A separate pnpm workspace in `frontend/` with its own lockfile, independent of `contracts/`; Node.js 26.13.1 and pnpm 10.34.9 (§7.1) |
| F6 | CI | `.github/workflows/frontend-ci.yml` with jobs **"Frontend check"** (Linux; intended to become a required status check after integration) and **"Frontend check (Windows)"**. `pnpm audit` fails on **moderate or higher** |
| F7 | Visual placeholder | Neutral placeholder theme, "Client Portal" label, system fonts, light theme only; branding and dark mode are later decisions |
| F8 | Component location | Shared design tokens and primitives in `frontend/packages/ui`; app-specific components in the application |

**Additional owner requirements.**

1. **Package age policy.** `frontend/pnpm-workspace.yaml` sets `minimumReleaseAge: 10080`, plus
   `minimumReleaseAgeStrict: true` and `minimumReleaseAgeIgnoreMissingTime: false`. Missing publication times
   fail instead of bypassing the policy. `frontend/.npmrc` sets `engine-strict` and `save-exact`.
2. **Navigation.** One page: header, skip link and main content. No empty navigation menu or menu button;
   navigation is added when more real pages exist.
3. **Frontend security.** No API calls, authentication, login placeholders, credentials, local or session
   storage, or other browser persistence. Accidental browser network calls are prevented until a separately
   approved, session-authenticated portal API contract exists.
4. **Design and accessibility.**
   - Semantic colour tokens only; Tailwind's default palette disabled.
   - WCAG AA contrast tested: 4.5:1 for text, 3:1 for focus indicators and other non-text elements.
   - Visible keyboard focus, keyboard-operable controls, accessible labels, correct landmarks.
   - The error boundary tested with a rendering failure, and its scope documented without claiming it
     catches asynchronous or event-handler errors.
   - Responsive layout checked at 360, 768 and 1280 px.
5. **Scope.** No admin portal, routing, data-fetching, form or headless UI libraries, Playwright or E2E
   infrastructure, backend changes, API clients, SMS sending or MVP functionality.

**How the requirements are met** (details in [`frontend/README.md`](../../frontend/README.md)):

- **Package age.** Verified for pnpm 10.34.9 with a local test registry and against the npm registry:
  - when versions are resolved, versions younger than 7 days are refused, for direct and transitive packages
    alike (`ERR_PNPM_NO_MATURE_MATCHING_VERSION`), with no fallback;
  - metadata without publication times fails (`ERR_PNPM_MISSING_TIME`);
  - `minimumReleaseAge` alone produces this behaviour. `minimumReleaseAgeStrict` and
    `minimumReleaseAgeIgnoreMissingTime` are pnpm 11 settings that pnpm 10.34.9 does not read. They are kept,
    as the owner requested, to keep the policy explicit for a future upgrade.

  **Limitation:** installing from an existing lockfile does not re-check the versions already recorded
  there. Every one of the 275 package versions in the committed lockfile was checked against the registry
  and found to be at least 7 days old. A from-scratch resolution with an empty store reproduced the lockfile
  byte for byte.
- **Network calls**, prevented in three layers:
  - ESLint rejects the browser network and storage APIs;
  - every test runs with those APIs replaced by functions that throw;
  - the production `index.html` carries a Content Security Policy with `connect-src 'none'`, checked after
    every build. A browser check of the built page confirmed that `fetch` and `XMLHttpRequest` are refused
    and that `sendBeacon` requests are blocked.

**Toolchain compatibility (ADR-0013 and §7.1 implementation-time check).** Every approved frontend package
supports Node.js 26 according to its declared `engines` range. With Node.js 26.13.1 and pnpm 10.34.9,
installation, lint, type check, tests and the production build passed on Linux, and on Windows in CI.

**Dependencies** (exact versions, all published at least 7 days before installation; licences read from the
installed packages, 2026-11-25):

| Package | Version | Licence | Used for | Approval |
| ------- | ------- | ------- | -------- | -------- |
| `react`, `react-dom` | 19.3.1 | MIT | UI runtime (the only packages in the production bundle, with React's `scheduler`) | §7.2 |
| `vite` | 8.3.8 | MIT | Development server and build | §7.2 |
| `@vitejs/plugin-react` | 6.3.0 | MIT | React support in Vite | §7.2 |
| `tailwindcss`, `@tailwindcss/vite` | 4.3.5 | MIT | Styling | §7.2 |
| `typescript` | 6.0.3 | Apache-2.0 | Type checking | §7.2; version F1 |
| `vitest` | 5.1.1 | MIT | Test runner | §7.2 |
| `jsdom` | 31.0.1 | MIT | Browser environment for tests | §7.2 |
| `@testing-library/react` | 16.3.3 | MIT | Component tests | §7.2 |
| `@testing-library/jest-dom` | 7.0.1 | MIT | DOM assertions (8.0.0 was younger than 7 days) | §7.2 |
| `eslint` | 10.15.0 | MIT | Lint | §7.2 |
| `typescript-eslint` | 8.73.0 | MIT | TypeScript lint rules (8.74.0 was younger than 7 days) | §7.2 |
| `eslint-plugin-react-hooks` | 7.1.1 | MIT | React hooks lint rules | §7.2 |
| `prettier` | 3.10.1 | MIT | Formatting | §7.2 |
| `@types/react`, `@types/react-dom` | 19.3.0 | MIT | React type definitions (React 19 ships none) | F2 |
| `@testing-library/dom` | 10.4.2 | MIT | Required peer of `@testing-library/react` | F2 |
| `@eslint/js` | 10.0.1 | MIT | ESLint's core recommended rules | F2 |
| `prettier-plugin-tailwindcss` | 0.8.1 | MIT | Tailwind class sorting | F2 |

**Transitive packages.** The lockfile has 275 package versions, including platform-specific binaries; 233
are installed on Linux.

| Licence | Packages |
| ------- | -------- |
| MIT | 185 |
| Apache-2.0 | 19 |
| ISC | 11 |
| BSD-2-Clause | 8 |
| BlueOak-1.0.0 (permissive) | 4: `minimatch`, `lru-cache`, `graceful-fs`, `@isaacs/nonrecursive-glob-match` |
| BSD-3-Clause | 3 |
| MPL-2.0 | 2: `lightningcss` and its binary, Tailwind's CSS processor at build time |
| CC-BY-4.0 | 1: `caniuse-lite`, browser-support data used at build time |

Only React's packages are shipped to browsers.

`pnpm audit`: no known vulnerabilities (2026-11-25).

---

## 11. Owner decisions — local development stack and expanded CI (Phase 3 Group 6, 2026-11-26)

Recorded from the owner's approval of the Group 6 proposal (decisions L1–L9 and additional requirements).
**The approval is limited to these decisions.** No ADR status changed. No new Go or npm dependency was
added, and the OpenAPI contract is unchanged.

| ID | Topic | Decision |
| -- | ----- | -------- |
| L1 | What runs in Compose | PostgreSQL, a one-off migration service and the Go API run in Docker Compose (`infra/local/compose.yaml`). The client portal runs natively |
| L2 | Image pinning | Every image is pinned by tag **and** SHA-256 digest, including PostgreSQL. The PostgreSQL image stays identical between backend CI and local Compose. The Go builder and distroless runtime use their multi-platform (index) digests |
| L3 | API container health check | A standard-library Go program, `backend/cmd/healthcheck`, shipped in the image |
| L4 | Frontend container | None |
| L5 | Frontend-to-backend connectivity | The Vite development server proxies `/healthz` and `/readyz` only. No frontend API calls |
| L6 | Migrations | Automatic: the one-off `migrate` service runs `migrate up` on every Compose start. The API depends on it with `condition: service_completed_successfully` |
| L7 | Security CI | gitleaks full-history secret scanning ("Secret scan"). Dependency review ("Dependency review") blocks moderate and higher vulnerabilities; licences are reported only |
| L8 | Windows | The stack is tested by Linux CI, plus a manual Docker Desktop checklist (`infra/local/README.md`). GitHub's Windows runners cannot run Linux containers |
| L9 | Required status checks | Inspect the real GitHub configuration before documenting it. The target list is in `docs/engineering/git-workflow.md` §0, with "Frontend check (Windows)" optional initially |

**Images** (digests recorded 2026-11-26; versions as approved in §7.1):

| Use | Image | Index digest |
| --- | ----- | ------------ |
| PostgreSQL (Compose and backend CI) | `postgres:18.7-trixie` | `sha256:c699303802144847d238196173655dbbecf7a96aeca1565f2b02eb671a1f0b6b` |
| Go builder (`infra/docker/backend.Dockerfile`) | `golang:1.27.4-trixie` | `sha256:430985542100b871e108ccc00e8dcc5eef08ca14ee5825e98ebf563d8ffec4ee` |
| Runtime (`infra/docker/backend.Dockerfile`) | `gcr.io/distroless/static-debian13:nonroot` | `sha256:8ce6dd87e9823c5b48dcfb2c8604714ccb351ce59607327bcc4957448da279b5` |

The "Local stack check" CI job fails if a pulled image is not pinned by digest, if a Dockerfile base
image is not pinned by tag and digest, or if the PostgreSQL image differs between Compose and backend CI.

**CI tools added**, all pinned. None is a new application dependency.

| Tool | Version | Licence | How it is pinned | Note |
| ---- | ------- | ------- | ---------------- | ---- |
| gitleaks | v8.30.1 (2026-02-21) | MIT | `go install github.com/zricethezav/gitleaks/v8@v8.30.1`; the Go checksum database verifies the source | §7.2 approved `gitleaks-action`. The scanner is run from its Go module instead, with the same Go toolchain as the backend, because the action downloads a release binary at run time. This is the same tool |
| `actions/dependency-review-action` | v5.0.1 (2026-11-17) | MIT | Commit SHA `bbc44d14a204704ccc493733616cd1e93a0321ec` | §7.2 approved GitHub dependency review. It fails on moderate or higher severity in all scopes (runtime, development, unknown); licence check on, with no allow or deny list, so licences are reported only |

**Secret scan of the existing history** (2026-11-26, before introducing the check):
- **Scope:** gitleaks v8.30.1, default rules, over the full history of `main` (15 commits scanned; gitleaks skips merge commits), and over the working branch.
- **Result:** one rule match on `main`, `generic-api-key`, in
  `backend/internal/httpserver/publicapi_test.go` line 218 (commit `b208db1`, PR #8). The same line also
  appears in the branch's original Group 4 commit `89b4d03`.
- **Reviewed as not a secret.** It is the `Idempotency-Key` header value in a unit test: a random UUID that
  an API client chooses as a request identifier. It is not a credential and grants no access.
- **Handling:** both exact fingerprints are listed, with this reason, in `.gitleaksignore`. Nothing else
  is allowlisted and no rule is disabled. After the exceptions, both histories scan clean.
- **Not in Git:** a git-ignored local `infra/local/.env` also matched in a working-tree scan. It is never
  committed or scanned in CI.

**Docker Compose minimum version: v2.7.0.**
- **Why:** this is the first release in which `up --wait` waits for a one-off service that another
  service depends on with `service_completed_successfully` to *complete*, instead of expecting it to be
  running.
- **How it was established:** from the Compose source history. Commit `fd8538f7` (2022-06-18), "Use
  appropriate dependency condition for one-shot containers when running `compose up --wait`", added
  `getDependencyCondition` to `pkg/compose/start.go`. It is present at tag v2.7.0 and absent at v2.6.1.
- **What was run:** older Compose releases were not executed. The stack was run with Compose v5.7.1
  (Docker Engine 29.9.3) locally and with the Compose version on GitHub's `ubuntu-24.04` runner.

**Verification of the migration lifecycle and container settings** (`scripts/local-stack-check.sh`, also run
in CI; Compose v5.7.1, 2026-11-26), all passed:
- **Fresh start on a new volume:** PostgreSQL healthy, then `migrate` exits 0 having applied the baseline,
  then the API becomes healthy. `/healthz` and `/readyz` return 200.
- **Repeated start:** `migrate` reports "no pending migrations"; the version is unchanged.
- **New migration:** a temporary migration added to `backend/migrations` was applied on the next
  `up --build --wait`, and the API stayed healthy. This was tested locally only and not committed; CI
  covers it through the failed-migration test.
- **Failed migration:** a deliberately invalid migration made `migrate` exit 1 and `up --wait` fail. The
  `api` container was created but never started, and the port did not answer. The migration was rolled
  back: version unchanged, no partial table. After it was removed, the stack started normally.
- **Persistence:** data survived `stop`/`start` and `down`/`up`. The API stopped gracefully (exit 0,
  "database pool closed").
- **Volume reset:** `down -v` removed the volume. The next start created a fresh database with only the
  baseline applied.
- **`migrate down` in a container** is refused, because `DB_HOST` is not a loopback IP.
- **API and migrate containers:** user 65532, read-only root filesystem, all capabilities dropped,
  `no-new-privileges`. The API process runs as uid 65532.
- **PostgreSQL with `no-new-privileges`:** it initialised a fresh volume, restarted and served normally,
  so the option is kept (owner requirement 7).
- **Published ports:** only 5432 and 8080, both on `127.0.0.1`. The API's metrics port is not reachable
  from the host.
- **`DB_PASSWORD`:** Compose refuses to run without it; there is no default.

**Development proxy:** through `pnpm dev`, `/readyz` and `/healthz` return the API's answers, and
`/v1/messages` is served by Vite, not proxied. The frontend CSP and lint rules are unchanged.

**Sandbox note.** The agent's environment intercepts HTTPS with its own certificate authority. Local image
builds therefore used an uncommitted Compose override that adds that certificate to the *build* stage
only. CI builds the committed Dockerfile unchanged.

---

## 12. Owner decisions — RLS evaluation spike (Phase 3 Group 7, 2026-11-26)

Recorded from the owner's approval of the Group 7 proposal, which covered the evaluation only. After
reviewing the evaluation, the owner took the adoption decisions in §12.1: **ADR-0010 is Accepted** for
tenant-owned control-plane tables, and **D-141 is resolved**. No dependency was added; no migration, business table, runtime role,
configuration, API, contract or frontend change was made.

**Approved scope:** a PostgreSQL 18.7 RLS evaluation as a Phase 3 group, with committed, reproducible
integration-tagged tests on synthetic, test-only tables. It covers:

- transaction-scoped tenant context and pgxpool connection reuse;
- fail-closed behaviour and RLS bypass conditions;
- cross-tenant access models and the documented RLS pitfalls;
- pgx statement-cache behaviour relevant to future sqlc queries, without adding sqlc product queries.

It also includes a manual, indicative performance evaluation that is never a CI gate. PgBouncer and other external poolers are documented as not tested. The outdated Group 6 status references are corrected.

**Additional requirements and how they were met:**

| ID | Requirement | Implementation |
| -- | ----------- | -------------- |
| A | Test isolation: roles are server-wide, so local runs use a separate disposable instance, never the Compose stack's persistent volume; roles are NOLOGIN, uniquely named, cleaned up and verified absent; existing roles and persistent objects are never modified | The tests need the build tags `integration,rlseval` and `RLS_EVAL_DISPOSABLE_SERVER=1`, and refuse a server whose maintenance database has been migrated. `scripts/rls-evaluation.sh` starts a throwaway, tmpfs-backed container; CI uses its ephemeral service. Roles are `rlseval_<run>_<fixture>_<purpose>`, NOLOGIN, dropped in cleanup; `TestMain`, the script and CI fail if any remain ([evaluation §2](../architecture/rls-evaluation.md#2-method-and-test-isolation)) |
| B | Threat model: the tenant GUC is controlled by trusted application code; a compromised credential or arbitrary SQL can set another tenant | [Evaluation §3](../architecture/rls-evaluation.md#3-threat-model-owner-requirement-b), with a test demonstrating it |
| C | Compare explicit cross-tenant policies with a dedicated BYPASSRLS role; recommend least privilege; document auditing; implement neither in product code | [Evaluation §6](../architecture/rls-evaluation.md#6-cross-tenant-access-owner-requirement-c) |
| D | Performance manual and indicative; no CI threshold | `BenchmarkRLSOverhead`, run only with `-bench` through the script ([evaluation §12](../architecture/rls-evaluation.md#12-performance-owner-requirement-d-indicative-only)) |

**Restrictions honoured:**

- no business tables or production migrations;
- no runtime role or configuration changes;
- no tenant, identity, authentication, RBAC, billing or messaging code;
- no API or frontend changes;
- no new infrastructure;
- no product-code `WithTenant` helper (a test-only sketch exists);
- the testing standards were not updated as though RLS were adopted while the decision was pending. After the adoption decision (§12.1) the owner asked for the adopted requirements to be recorded there, and they now are.

**CI change:** the "Go test and build" job runs the evaluation tests against its own PostgreSQL service
and checks that no test role remains. `go vet` and golangci-lint also cover the `rlseval` build tag. The
required-check names are unchanged.

**Evaluation results (summary; details in the [evaluation](../architecture/rls-evaluation.md)):**

- **Context and pooling:** transaction-scoped context with pgxpool is safe. Session-level context leaks between requests and must never be used.
- **Fail closed:** the `NULLIF` policy form fails closed consistently.
- **Who bypasses RLS:** superusers, owners of unforced tables and BYPASSRLS roles bypass it. The application role cannot.
- **The current setup gets no protection.** The API currently connects as the bootstrap superuser, so RLS would give no protection until roles are separated.
- **Pitfalls found:** unique constraints, foreign keys, views, `SECURITY DEFINER` functions and `TRUNCATE` need the mitigations listed in the evaluation.
- **Caching:** the statement and plan caches follow the context in all pgx execution modes.
- **Performance:** no measurable policy overhead in an indicative 1-million-row test; one extra statement per transaction to set the context.

### 12.1 Owner decisions on RLS adoption (2026-11-26)

Taken by the owner after reviewing the [evaluation](../architecture/rls-evaluation.md). They are recorded in
ADR-0010 above, and they resolve D-141.

| ID | Decision | Status |
| -- | -------- | ------ |
| RLS-1 | **Adopt ADR-0010 option B**: RLS as defense-in-depth for **tenant-owned control-plane tables**, on top of mandatory application-level tenant filtering. Conditional on all requirements in [evaluation §10](../architecture/rls-evaluation.md#10-conditions-of-adoption-adopted-by-the-owner-rls-1). High-volume data-plane tables are explicitly excluded | **Accepted** |
| RLS-2 | **Policy expression:** `tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid`, with equivalent `USING` and `WITH CHECK` policies. The tenant context is set transaction-locally through one application helper, which rejects missing or invalid tenant IDs before executing tenant queries. Session-level tenant settings are never used | **Accepted** |
| RLS-3 | **Explicit cross-tenant policies:** a dedicated reporting role with its own credentials and connection pool, explicit read-only policies, and the minimum necessary table and column privileges. Cross-tenant writes need separate approval and narrowly scoped policies. `BYPASSRLS` only for the migration role and a separately controlled break-glass role, never for application, reporting or worker connections. Cross-tenant administrative operations are audited in the application | **Accepted** |
| RLS-4 | **Role separation is a prerequisite.** Implemented in a separate, explicitly approved development group, before or alongside the first tenant-owned business table. The API must not use the bootstrap PostgreSQL superuser. Not implemented in Group 7 | **Accepted**, implementation pending a future approved group |
| RLS-5 | **High-volume data-plane tables deferred.** The control-plane decision is not extended to them automatically. They are evaluated separately when their design and workload are known: write throughput, partitioning, query patterns, representative volume | **Deferred** |

**Consequences recorded in the same pull request:**

- the [testing standards](../standards/testing-standards.md#32-integration-tests) now require the RLS tests for every tenant-owned control-plane table, and the catalog CI guard;
- `AGENTS.md` §10.7 and the [security baseline](../security/security-baseline.md#4-tenant-isolation) §4 summarise the adopted rules;
- the roadmap marks the Phase 3 RLS evaluation item complete.

**Nothing was implemented:** there are no business tables, migrations, runtime roles, product-code tenant helper or CI guard yet. The guard and the RLS tests apply from the first tenant-owned table, which needs the role-separation group (RLS-4) first or alongside it.

---

## 13. Phase 3 closure (2026-11-26)

The Phase 3 walking skeleton is complete. All eight groups (0–7) are merged, and every exit criterion in the
[roadmap](../roadmap/development-roadmap.md#phase-3--walking-skeleton) is met. Requested by the owner after a
read-only audit of `main` at `63ab5de`.

**This closure is documentation only.** No ADR status changed, and no code, configuration, CI workflow,
contract or migration was modified. Phase 4 is **not** authorized by it.

| Exit criterion | Evidence |
| -------------- | -------- |
| A new developer/agent can start the local stack with one documented command | `infra/local/README.md` §2: `docker compose -f infra/local/compose.yaml up -d --build --wait`, after a one-time copy of `.env.example`. Verified on Linux by "Local stack check" on every pull request. Verified on Windows by the owner's Docker Desktop validation on 2026-11-26 (below) |
| CI runs on every PR and blocks merging on failure | All five workflows (Backend, Contracts, Frontend, Local Stack, Security CI) run on `pull_request` and `push` to `main`, with no path filters. Ruleset R1 requires nine checks (below) |
| Branch protection configured on `main` | Ruleset R1 ([git workflow §0](git-workflow.md#0-repository-state)) |
| Health endpoint served; frontend shell renders; tokens applied; all tests pass | `/healthz` and `/readyz`: API smoke test in "Go test and build", and "Local stack check". Shell and tokens: "Frontend check" (render, error-boundary and WCAG contrast tests; build check). All five workflows passed on `main` at `63ab5de` |
| RLS evaluation completed and documented before Phase 4 | [RLS evaluation](../architecture/rls-evaluation.md); ADR-0010 Accepted; §12.1; PR #11 |

**Required status checks** (ruleset R1, updated by the owner on 2026-11-26, owner decision L9):

- **Required (nine):**
  - `Go test and build`
  - `Go lint`
  - `Go vulnerability check`
  - `Go generate check (Windows)`
  - `OpenAPI contract check`
  - `Frontend check`
  - `Local stack check`
  - `Secret scan`
  - `Dependency review`
- **Optional:** `Frontend check (Windows)` and `Local stack config (Windows)`.

The ruleset is a repository setting, which agents do not change. The owner made the change. The agent session
cannot read rulesets, so the change is recorded on the owner's report; the closure pull request records what it
observed of the checks gating it.

**Windows Docker Desktop validation** (owner decision L8): completed by the owner on 2026-11-26, with no failures
reported, using Docker Desktop 4.100.1, Docker Engine 29.9.3 and Docker Compose 5.7.1. Recorded in
`infra/local/README.md`.

**Carried forward, not Phase 3 exit criteria:**

- **Database role separation (RLS-4):** a separate, approved group, before or alongside the first tenant-owned
  table.
- **Data-plane RLS (RLS-5):** a separate evaluation when the message tables are designed.
- **Phase 2 exit criteria:** still open ([Phase 2 exit checklist](../discovery/phase2-exit-checklist.md)).
- **Native Windows PowerShell steps** in `backend/README.md`: not separately validated.
- **Phase 4:** needs explicit owner authorization (`AGENTS.md`, "Authorized scope").

---

## 14. Owner decisions — MVP scope (Phase 2, EX-1, 2026-11-26)

On 2026-11-26 the owner reviewed the 25 capabilities still undecided in the
[MVP scope worksheet](../discovery/mvp-scope-worksheet.md) in five batches and approved all 34 MVP capability
decisions, the related discovery-register recordings, an amendment to Phase 2 exit criterion EX-1 and the
production gates. The worksheet's
[Owner approval of the MVP scope](../discovery/mvp-scope-worksheet.md#owner-approval-of-the-mvp-scope-2026-11-26)
section is the authoritative record of the scope; this section summarizes it. **No ADR status changed.** This
is a scope decision only: it does not authorize implementation or Phase 4, and it does not close Phase 2.

| Topic | Decision |
| ----- | -------- |
| Scope | **21 MVP** (MVP-01, 03, 04, 05, 06, 07, 09, 10, 14, 16, 20, 21, 22, 25, 27, 28, 29, 30, 32, 33, 34), **8 Post-MVP** (MVP-02, 11, 12, 15, 17, 18, 19, 23), **5 Deferred** (MVP-08, 13, 24, 26, 31). MVP-13 counts as Deferred |
| Client roles (MVP-03) | Owner (manages users and API keys), Sender (sending functions), Viewer (read-only); no separate Billing role in the MVP |
| Internal roles (MVP-04) | Super Admin, Operations, Finance; least privilege; **MFA mandatory for all human internal admin accounts** (D-021, widened) |
| Public API (MVP-06, MVP-07) | Core API keys with the `messages:send` and `messages:read` scopes; single-message submission with a mandatory idempotency key; the same key with a different payload is rejected; no promise of exactly-once provider delivery |
| Money (MVP-10, 20, 21, 22) | Prepaid only with an immutable credit ledger and manual Finance top-ups; one versioned per-segment rate per client; one authoritative segment calculation for preview and billing; financial history for client Owners and authorized Finance staff. **The billing rules stay open** (D-080, D-081, D-083, D-092, D-093, D-096; ADR-0011, ADR-0019) and block implementation |
| Providers (MVP-25, 26, 27, 33) | Simulator first; Operations-controlled routing; no real telco adapter without separate approval; no automatic failover; internal usage summary, with actual telco reconciliation before production billed traffic |
| Data (MVP-16, 34) | Append-only status history, masked numbers, no message content by default; retention classes from the first table, automated expiry before production with legally approved periods |
| EX-1 amendment | Launch customers may remain **TBD until pilot discovery**; at least one validated pilot customer before production onboarding or traffic. D-061 stays open with this trigger |
| Production gates | **PG-1** no real telco adapter without approval and official documentation · **PG-2** production provider integration gated by the regulatory decisions · **PG-3** actual telco reconciliation before production billed traffic · **PG-4** automated retention expiry before production · **PG-5** no sending with a missing or unapproved rate · **PG-6** the simulator never serves production customer traffic or generates customer-billable usage · **PG-7** production-critical alert coverage and D-066 ownership resolved before production (no automatic 24/7 staffing requirement) · **PG-8** at least one validated pilot customer |
| Discovery register | Approved: D-020, D-021 (widened), D-024, D-026, D-029. Partly decided: D-064, D-065, D-073, D-078, D-082, D-084. D-089 decided for scope. Notes: D-061, D-063, D-066, D-088, D-091, D-099 |

**Still open** (not decided here): the billing rules above; D-095 and D-097; webhook semantics (D-099); legal
questions (D-122, D-124, D-130, D-088, D-113; EX-4); launch-customer identity (D-061); D-066 and D-078 staffing.
Phase 2 remains open: EX-1 and EX-13 are satisfied
([Phase 2 exit checklist](../discovery/phase2-exit-checklist.md)).
