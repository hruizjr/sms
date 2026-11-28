# Decision Log (Architecture Decision Records)

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Living document — thirteen ADRs accepted (2026-11-25; ADR-0010 and ADR-0012 (limited scope) on 2026-11-26); others Proposed | Project owner | 2026-11-28 |

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
   date. A decision affecting **pricing, product scope or launch** also requires **business-partner approval**,
   recorded with its date before the decision is labelled approved (governance rule G1, [§17.1](#17-owner-decisions--business-decisions-review-phase-2-ex-9-2026-11-26)).
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
**On 2026-11-26 the owner decided to proceed simulator-only** (Phase 2 exit criterion EX-3, see
[§15](#15-owner-decisions--simulator-only-development-phase-2-ex-3-2026-11-26)) and **accepted ADR-0012 with a
limited scope**: the provisional provider interface, capability declarations, adapter abstraction and simulator.
**On 2026-11-26 the owner approved provisional traffic targets** (Phase 2 exit criterion EX-2, see
[§16](#16-owner-decisions--provisional-traffic-targets-phase-2-ex-2-2026-11-26)). No ADR status changed.
**On 2026-11-26 the owner completed the EX-9 business-decisions review** (see
[§17](#17-owner-decisions--business-decisions-review-phase-2-ex-9-2026-11-26)): governance rules G1–G4, owner proposals pending partner approval, and earlier
product-scope and pricing decisions relabelled as pending partner ratification. No ADR status changed.

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
| 0012 | Provider abstraction, capabilities, and simulator adapter | **Accepted — limited scope** (owner, 2026-11-26): option B limited to the provisional interface, capability declarations, adapter abstraction and simulator; no SMPP library or real adapter |
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
   *Amended 2026-11-26 (owner decision G1-9, [§18](#18-owner-decisions--database-role-separation-and-tenant-registry-phase-4-group-1-2026-11-26)):
   the migration role has `NOBYPASSRLS`; no blanket bypass is granted; cross-tenant data migrations need a
   separately designed, controlled procedure. The rest of this item is unchanged.*
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

- **Status:** **Accepted — limited scope** (by owner, 2026-11-26) — **option B, limited to the provisional
  provider interface, capability declarations, adapter abstraction and simulator** · **Date proposed:** 2026-11-24
  · **Revised:** 2026-11-24 (provider capabilities and receipt normalization added)

**Owner decision (2026-11-26, Phase 2 EX-3 simulator-only decision; see
[§15](#15-owner-decisions--simulator-only-development-phase-2-ex-3-2026-11-26)):** option B is accepted, **limited
to**:
- a **provisional** provider interface, built against the simulator and expected to change when the first
  official telco specification is received;
- **capability declarations**: core logic consults declared capabilities, never a provider's identity; every
  capability of a real provider cites its official documentation, and unknown capabilities are declared unknown
  and treated conservatively;
- the **adapter abstraction**: one adapter per provider, with normalization and outcome classification inside
  the adapter;
- the **simulator** adapter, with generic synthetic settings only (S2).

This acceptance does **not** cover, and these remain **separate decisions**: the **SMPP client library**
selection (a separate ADR, made with telco specifications in hand); **any real telco adapter**, its
configuration or credentials (gates PG-1, PG-2 and PG-9); the final provider interface; and the
ambiguous-submission policy (D-095). Accepting the ADR authorizes no implementation: simulator and adapter code
need an approved Phase 4 group.

The original analysis below is kept as written.

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

> **Status update (2026-11-27, owner decision):** the **"Frontend check (Windows)"** job of F6 is removed from
> `.github/workflows/frontend-ci.yml`. The Linux job **"Frontend check"** keeps every check (generated API types,
> formatting, lint, type check, tests, build and build verification, security audit). Windows compatibility of the
> frontend is validated by the owner's local Windows testing. The Windows job was optional, never a required status
> check (§13). Windows support of the application and the Windows jobs of Backend CI and Local Stack CI are unchanged.

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
| RLS-3 | **Explicit cross-tenant policies:** a dedicated reporting role with its own credentials and connection pool, explicit read-only policies, and the minimum necessary table and column privileges. Cross-tenant writes need separate approval and narrowly scoped policies. `BYPASSRLS` only for the migration role and a separately controlled break-glass role, never for application, reporting or worker connections. Cross-tenant administrative operations are audited in the application. *Migration-role part superseded by G1-9 (§18, 2026-11-26): `NOBYPASSRLS`* | **Accepted** |
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

> **Status update (2026-11-26, EX-9 review, [§17](#17-owner-decisions--business-decisions-review-phase-2-ex-9-2026-11-26)):** the scope and pricing decisions in
> this section are **owner-approved, pending partner ratification (PR)** under governance rule G1. The production
> gates are owner engineering decisions (G4). Where this section says "one versioned per-segment rate per client",
> read "one versioned rate per client **per billable unit**"; the billable unit (segment count) is an owner
> proposal pending partner approval (§17.4).

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

---

## 15. Owner decisions — simulator-only development (Phase 2, EX-3, 2026-11-26)

On 2026-11-26 the owner decided to proceed with **simulator-only development and testing without a telco
specification**, satisfying Phase 2 exit criterion EX-3 by recorded owner decision with the risks acknowledged.
No telco information has been received: every question in the
[telco discovery checklist](../integrations/telco-discovery-checklist.md) stays **unanswered**. This decision
authorizes **no implementation** (no simulator code, real adapter, migration or product feature), does not close
Phase 2, and does not authorize Phase 4.

### 15.1 Decisions S1–S8

| # | Decision |
| - | -------- |
| S1 | **Simulator-only.** Development and testing use the simulator as the only provider until at least one official telco specification is received and a real adapter is separately approved. |
| S2 | **Synthetic settings only.** Simulator settings have generic names (for example `synthetic-default`); none is named after, or claims to emulate, Smart, Globe, PLDT Enterprise or DITO. Every capability is labelled *synthetic* or *unknown*. Settings for a named provider require that provider's official documentation ([testing standards §3.3](../standards/testing-standards.md#33-contract-tests)). |
| S3 | **Nine failure scenarios.** The simulator covers acceptance, rejection, timeout (ambiguous outcome), delayed receipts, duplicate receipts, out-of-order receipts, **missing receipts**, **throttling**, and **session loss and reconnect**. |
| S4 | **Conservative capability defaults.** No provider-side duplicate detection, no status-query capability, no guaranteed receipts and no guarantee that a receipt follows the acknowledgment. Ambiguous outcomes are never automatically resubmitted (interim rule below). |
| S5 | **Provisional provider interface.** Built under ADR-0012 option B (accepted with a limited scope, below), labelled provisional, and expected to change when the first telco specification arrives. The SMPP client library remains a separate ADR made with specifications in hand. |
| S6 | **No route to real networks.** The simulator runs locally or in-process, makes no outbound network connection, uses only synthetic numbers, holds no real telco credentials, and cannot be configured as a production provider (gate PG-6). |
| S7 | **Honest reporting.** Results are described as "verified against the simulator". Nothing claims telco compatibility, throughput, delivery rates or receipt timing; demos are labelled simulated. |
| S8 | **Telco discovery continues** outside simulator-only development (checklist questions, PLDT Enterprise and Smart (D-112), direct or intermediary connectivity (D-118), whether to approach DITO). The answers are production prerequisites under PG-1 and PG-9. **Update 2026-11-26 (EX-9, §17.3):** D-118 resolved as direct aggregation only and DITO included in the coverage objective (Smart, Globe, DITO), both pending partner ratification (PR); PLDT Enterprise is a possible commercial counterpart, not a separate network. |

### 15.2 Related owner decisions

| Topic | Decision |
| ----- | -------- |
| **PG-9** (new production gate) | Every real adapter must pass the provider contract suite using the provider's **official specifications**, plus validation in the provider's **test environment** where one is available. If no test environment is available, the gap is documented and an alternative validation is explicitly approved by the owner. Recorded with PG-1 to PG-8 in the [MVP scope worksheet](../discovery/mvp-scope-worksheet.md#production-gates). |
| **EX-7** (amended) | Telco technical discovery is **formally deferred** during simulator-only development. Unanswered questions remain **production prerequisites**; they are not marked as collected. |
| **ADR-0012** | **Accepted — limited scope:** option B, limited to the provisional provider interface, capability declarations, adapter abstraction and simulator. No SMPP library selection and no real adapter implementation. |
| **D-095** | **Remains open.** **Interim rule:** uncertain submissions remain uncertain and are **never automatically resubmitted or failed**. |

### 15.3 Accepted risks and limitations

The owner accepted these risks of building before any telco specification exists:

- **Rework (R-36):** protocol, receipt format, message ID behaviour (D-114), error codes and retry rules (D-106),
  and encoding and segment counting (D-105, D-117) may differ from the provisional interface and tests.
- **Commercial uncertainty (R-02, R-08):** billing basis, rates, deposits and throughput terms are unknown.
- **Segment counting:** the platform's own calculation may differ from a telco's (D-117) until confirmed
  (`TODO(telco-confirm)`).
- **Unresolved uncertain outcomes:** without provider capabilities (D-115), uncertain messages can only be held
  (D-095, D-096 open).
- **Hosting and connectivity:** VPN, static IP and allowlisting requirements are unknown (checklist §5, §6).
- **No capacity evidence:** load results against the simulator measure only the platform; no capacity target is
  derived from them (AGENTS.md §11.4).

Limitations while simulator-only: no real SMS and no real telco connection in any environment; no telco-specific
code, configuration, credentials, hostnames, ports, system IDs, encoding values or error mappings; no SMPP
library; routing (MVP-25) points only at simulated providers and the usage summary (MVP-27) shows simulated usage
only; network detection for ported numbers (D-070) and off-net routing (D-107) stay unknown.

**Still open:** D-095 (final policy), D-096, D-097, D-098; all telco-dependent items (D-100 to D-119); EX-4
(counsel); Phase 4 authorization and database role separation (RLS-4). Phase 2 remains open: EX-1, EX-3 and
EX-13 are satisfied and EX-7 is deferred ([Phase 2 exit checklist](../discovery/phase2-exit-checklist.md)).

---

## 16. Owner decisions — provisional traffic targets (Phase 2, EX-2, 2026-11-26)

On 2026-11-26 the owner decided to proceed with **provisional platform design targets**, satisfying Phase 2 exit
criterion EX-2 by recorded owner decision. The targets, test workloads and distributions are recorded in full in
the [traffic evidence worksheet §7](../discovery/traffic-evidence-worksheet.md#7-provisional-design-targets-owner-decision-2026-11-26-ex-2),
which is the authoritative record; this section summarizes the decisions. **No ADR status changed.** The decision
authorizes **no implementation**, does not close Phase 2 and does not authorize Phase 4.

They are **platform capacity targets**, not evidence, not a demand forecast and not telco throughput. **No traffic
evidence has been collected.** D-004 and D-060 remain **open and unused**: the targets are not derived from,
compared with or checked against the unverified monthly figure.

| # | Decision |
| - | -------- |
| TR-1 | **Tenants:** pilot 1–3; initial (first 6 months of production) up to 10; growth (12 months) up to 50. One tenant may generate up to 50 % of traffic |
| TR-2 | **Messages per day:** 10,000 / 100,000 / 1,000,000 (about 300,000 / 3,000,000 / 30,000,000 per 30-day month) |
| TR-3 | **Design peak submission rate**, sustained 15 minutes: 10 / 50 / 250 messages/s, which is **8× / 4× / 2×** headroom over the calculated rate (busiest hour 15 % of the day, bursts 3× the busiest-hour average). **Stress test:** 500 messages/s for 15 minutes |
| TR-4 | **Segments:** 1.3 per message on average, 10 % UCS-2, peak segment rate 1.5 × the submission rate; counted by the platform's own calculation (D-117 open) |
| TR-5 | **Distributions** (synthetic or provisional, never telco facts or demand forecasts): use case OTP 60 % / alerts 35 % / other 5 %; channel API 90 % / portal 10 %; 80 % of traffic 07:00–22:00 Asia/Manila; destination network split synthetic across two simulated providers; Philippines only; zero inbound messages |
| TR-6 | **Scenarios and test workloads:** pilot, initial, growth and stress, with **message submissions, segments and receipt events defined and reported as separate workloads** (receipt events a synthetic upper bound of one per segment plus 10 % duplicates); revision triggers recorded in the worksheet §7.6 |
| TR-7 | **Permitted uses:** test design, queue and partitioning evaluation (ADR-0006, ADR-0008), configured limits, placeholder alert thresholds, development and test environment sizing. **Not** telco negotiations, customer commitments, financial models or production infrastructure sizing |
| TR-8 | **EX-2 satisfied** by the recorded provisional targets. D-062, D-074 and D-075 carry provisional answers, not confirmed ones; D-004 and D-060 remain open and unused |
| TR-9 | **EX-6 deferred.** Traffic evidence collection continues during pilot discovery; the worksheet's evidence sections stay empty and nothing is marked collected |
| PG-10 | **Gate:** before general production rollout, the targets are re-baselined against validated pilot-customer volume information (PG-8) and contracted telco limits (PG-1). Production infrastructure sizing requires this re-baseline |

**Platform capacity versus telco throughput:** the targets are what the platform must accept, persist, queue,
dispatch to the simulator and process. Telco throughput comes only from signed agreements and official
documentation (D-101) and is unknown. In production, dispatch runs at the lower of the two, with excess queued
under back-pressure.

**Still open:** D-004, D-060, D-061, D-062, D-074, D-075 (for evidence), D-101, D-117; an OTP latency objective;
storage sizing (D-073, D-124). Phase 2 remains open: EX-1, EX-2, EX-3 and EX-13 are satisfied and EX-6 and EX-7
are deferred ([Phase 2 exit checklist](../discovery/phase2-exit-checklist.md)).

---

## 17. Owner decisions — business decisions review (Phase 2, EX-9, 2026-11-26)

On 2026-11-26 the owner completed the review of the Phase 2 business decisions (EX-9: D-080 to D-084 and D-160
to D-163) in three batches, and recorded a governance rule for decisions reserved to the business partners. This
section is the authoritative record of that review. **No ADR status changed.** **Partner approval has been
established only where this section says PD.** Nothing here is approved merely because the owner reviewed it,
because time has passed, or because the partners have not objected. The review authorizes no implementation and
does not close Phase 2. **EX-9 status: owner review complete, partner approval pending.**

**Labels used in this section and in the records that cite it:**

| Label | Meaning |
| ----- | ------- |
| **OD** | Owner decision on a day-to-day project matter (the owner has authority) |
| **PD** | Partner-approved decision, confirmed in writing by the owner |
| **OP** | Owner proposal, **pending partner approval** |
| **PR** | Owner-approved, **pending partner ratification** (recorded earlier as an owner decision) |
| **F** | Fact provided by the owner |
| **P** | Provisional planning estimate |
| **D** | Deferred, with a revisit trigger |

### 17.1 Governance (G1–G4)

| # | Decision | Label |
| - | -------- | ----- |
| G1 | "The project owner has decision authority over day-to-day project matters. Decisions affecting pricing, product scope, or launch require business-partner approval in addition to the project owner's approval. Both approvals, including their dates, must be recorded before a decision is labelled approved. Supporting approval records are kept outside Git." | OD |
| G2 | **Evidence of partner approval:** the owner records written confirmation that the relevant business partners approved the decision, with the approval date, its scope and, where applicable, a non-sensitive reference to the meeting record or written confirmation. Minutes, messages and other private documents stay outside Git. **Silence or lack of objection is not approval.** No quorum or voting mechanism is defined here: approval follows the partners' agreed governance arrangement. | OD |
| G3 | Product-scope and pricing decisions recorded earlier as owner decisions are relabelled **PR** (list in §17.2). Development may proceed against the provisional scope; partner ratification is required for final approval of those decisions and before production launch. | OD |
| G4 | **Production gates and launch.** The technical definitions and acceptance criteria of production gates PG-1 to PG-10 are **engineering decisions under the owner's authority (OD)**. The **decision to authorize production launch** requires partner approval in addition to satisfaction of the gates. The partners' launch right does **not** require separate partner approval of each ordinary technical readiness criterion or implementation choice. | OD |

### 17.2 Classification of earlier records

| Record | Label |
| ------ | ----- |
| EX-1: all 34 MVP capability decisions ([MVP scope worksheet](../discovery/mvp-scope-worksheet.md)), including the decisions of 2026-11-25 (MVP-01, 05, 08, 09, 13, 29, 30) and 2026-11-26 (MVP-11, 12) | **PR** |
| Pricing records: MVP-20 (prepaid), MVP-21 (client rate), the D-082 and D-084 partial decisions | **PR** |
| D-021, D-024, D-026 and D-029 approvals, as product-scope decisions | **PR** |
| D-160 (operating model) and its dependants: D-118 resolution, intermediary questionnaire status; the DITO coverage change | **PR** |
| Production gates PG-1 to PG-10, including the validated-pilot gate (PG-8): definitions and acceptance criteria | **OD** (G4) |
| EX-2 traffic and test-planning decisions TR-1 to TR-9 | **OD** |
| EX-3 development approach S1 to S8 | **OD** |
| ADR-0010, ADR-0012 (limited scope) and the RLS decisions RLS-1 to RLS-5 | **OD** |

### 17.3 Operating model and business facts

| Item | Decision or fact | Label |
| ---- | ---------------- | ----- |
| **D-160** | **Direct SMS aggregation only**, through the business's own commercial agreements and direct technical integrations with telcos. **No third-party SMS aggregator or intermediary may route messages**, for the first live connection or any fallback route. **No reseller or white-label operating model.** | **PR** |
| D-160 | **Coverage objective:** direct integration with the Philippine mobile network operators **Smart, Globe and DITO**. PLDT Enterprise is **not a separate mobile network operator**; it may be a commercial or enterprise contracting counterpart (its relationship to Smart is D-112). The objective is **not a promise of any network at launch**; each integration is subject to that operator's technical, commercial and regulatory requirements. Including DITO is a product-scope change. | **PR** |
| D-118 | Resolved as **direct aggregation only** (depends on D-160) | **PR** |
| D-161 | One registered legal entity is intended (confirms D-047) | F |
| D-161 | The business partners hold formal approval rights over **pricing, product scope and launch** | F |
| D-161 | Legal-entity identifiers stay out of Git; the appropriate legal structure is subject to counsel's advice (EX-4) | OD; D (counsel) |
| D-162 | **Clients are charged per message** (general direction) | **PD** |
| D-162 | No additional fee is decided; setup, sender ID, platform or other fees are not assumed | F |
| D-162 | Numeric gross-margin target deferred until actual telco rate cards are available (D-108) | D |
| D-162 | The technical model distinguishes a submitted message from its segments and keeps segment-level usage records | OD |
| D-163 | No binding launch-date, pricing, volume, exclusivity or service-level commitment to customers, investors or telcos has been reported | F |
| D-163 | About **5,000,000 messages per day**: an owner-provided, **unverified planning estimate**, not verified demand, a commitment or an initial-launch requirement; recorded separately from D-004 and D-060, which remain open and unused | P |
| D-163 | Traffic targets TR-1 to TR-6 unchanged. The initial production build is designed and load-tested for the existing **1,000,000 messages/day growth capacity** (250 messages/s design peak, 500 messages/s stress). Scaling toward the estimate follows actual usage and performance evidence and the PG-10 re-baseline | OD |

### 17.4 Charging, refunds, balance and pricing (all pending partner approval)

| Item | Proposal | Label |
| ---- | -------- | ----- |
| **D-080** | The charge is **settled on provider acceptance** (`provider_accepted`). A platform-side rejection before provider acceptance never produces a settled charge. Recognizing a charge does not depend on final delivery receipts. The charge event is separate from the billable unit. Provisional: revisit when the telco billing basis is known (D-108) | **OP** |
| **D-083** 1 | Rejected or failed before provider acceptance: release the reservation; no charge | **OP** |
| D-083 2 | Provider-accepted, then `undelivered`: no automatic refund while the telco still charges us. Where the telco does not charge us or credits us for the message, the benefit is passed to the client as a balance credit; **the client credit is not assumed to equal the telco's wholesale credit** — its amount follows the partner-approved pricing and refund policy and the partial-delivery rules (D-097). Exceptional manual Finance adjustments with a reason and audit record. Revisit under D-108 | **OP** |
| D-083 3 | `submission_uncertain`: the reservation stays held; never automatically settled, released, failed or resubmitted; final treatment under D-096 (EX-14) | **OP** |
| D-083 4 | Provider-accepted with no final receipt (`unknown`): refund treatment under D-096 (EX-14) | D |
| D-083 5 | Confirmed platform-caused duplicate: manual Finance credit through an auditable compensating ledger entry with a reason; the client does not bear the cost | **OP** |
| D-083 6 | Duplicate or late receipts never cause a duplicate charge or refund (existing rule) | Existing rule |
| D-083 7 | Cash refund of unused prepaid balance: **undecided**, outside D-083; neither allowed nor prohibited; needs a separate partner decision | Open |
| **D-162 / B3-1** | **Billable unit:** one submitted message, with **billable quantity equal to its SMS segment count** (one segment = 1; three segments = 3). The client sees and manages one message; the pre-send estimate shows the segment count and the estimated charge. Provisional: revisit when telco rate cards and billing rules are known (D-108, D-117) | **OP** |
| **D-081** | The client's prepaid service balance is denominated in **Philippine pesos (PHP)**, not SMS credits. Precision, rounding and storage representation are not settled (EX-12). The balance is **not** described as cash-withdrawable, transferable or a general-purpose e-wallet; those matters are undecided | **OP** |
| **D-082** | Prepaid only for the MVP; postpaid is Post-MVP and not in the initial implementation; revisit when a credible enterprise or government customer requirement creates a business case for monthly invoicing; postpaid would need an approved credit-risk policy, tax-compliant invoicing and partner approval | **OP** (the existing prepaid-only record is **PR**) |
| **D-084** 1 | One versioned rate per client **per billable unit**; no customer pricing dimensions by destination network, encoding, message type or volume tier (deferred until telco rate cards and operating costs justify them). The platform may record actual telco costs internally by network, route, encoding and segment for cost attribution and margin analysis (OD) | **OP** |
| D-084 2 | Rates are versioned and effective-dated and apply to new messages only; each accepted message keeps a reference or snapshot of the rate version assigned at acceptance, so a later change never alters the price of an accepted or queued message. Any client-notice period is a contractual matter for the partners | **OP** |
| D-084 3 | Client-rate approval, staged: until a partner-approved pricing policy exists, each client rate needs owner and partner approval; afterwards Finance may set rates within the policy's boundaries, and rates outside it need owner and partner approval. **Entering a rate in the platform is not approving it.** Evidence follows G1 and G2. "Approved rate" in gate PG-5 means approved under this rule | **OP** |
| D-084 4 | Standard list rate versus negotiated rates: deferred until the partners approve the initial go-to-market pricing approach; no public list price or negotiated-only policy is assumed; does not block the pricing data model | D |

### 17.5 Dependencies and status

- **Partner approval** of every OP item and ratification of every PR item, evidenced under G2.
- **Separate partner decisions:** cash refunds of balance; client-notice period for rate changes; list versus
  negotiated rates; additional fees; balance transferability or withdrawal.
- **EX-4** (counsel): legal structure and regulatory path. **EX-12** (ADR-0019, D-092, D-093): currency handling,
  precision, rounding and storage. **EX-11** (ADR-0011, ledger). **EX-14** (D-095 final policy, D-096, D-097,
  D-098, D-099). **D-108 and D-117** (telco billing basis, rate cards, segment counting) as revisit triggers.
- **Phase 2 status:** EX-1 is **owner-approved, pending partner ratification**; EX-9 is **owner review complete,
  partner approval pending**; EX-2, EX-3 and EX-13 are satisfied; EX-6 and EX-7 are deferred. Phase 2 is not
  complete. Non-production engineering work may proceed against the provisional scope once the owner authorizes
  it; partner approval is not a blanket prohibition on such work.

---

## 18. Owner decisions — database role separation and tenant registry (Phase 4 Group 1, 2026-11-26)

On 2026-11-26 the owner approved **Phase 4 Group 1** in full, as one implementation group and one Draft pull
request. It is the first Phase 4 group: it establishes the database foundation for the customer and messaging
groups that follow. **No ADR status changed.** No later Phase 4 group is authorized; the next one starts only
after this one is reviewed and merged.

**Approved scope:**

| Part | Scope | Implementation |
| ---- | ----- | -------------- |
| A | Database role separation (owner decision RLS-4) | Roles `sms_migrator` and `sms_app`, defined by one idempotent script, [`dbroles/provision.sql`](../../backend/internal/database/dbroles/provision.sql), used by the local stack, CI and the test helpers. The API refuses an unsafe role on every new connection (`database.RequireApplicationRole`); `migrate` refuses a privileged role (`database.CheckMigrationRole`) |
| B | Transaction-local tenant-context helper (RLS-2) | `database.DB.InTenantTx`: validates a non-nil RFC 9562 tenant UUID before any SQL, then `set_config('app.tenant_id', …, true)` as the first statement; never a session-level setting |
| C | Platform-owned tenant registry | Migration `20261126175603_create_tenants.sql`, sqlc queries (`backend/queries/tenancy.sql`; sqlc v1.31.1 locked in `backend/tools`), and the `internal/tenancy` repository: create, get, list, activate, suspend, reactivate. No HTTP endpoints |
| D | Row-Level Security catalog guard in CI (RLS-1; testing standards §6) | `internal/database/rlsguard`: run against a freshly migrated database; detects missing RLS, missing `FORCE`, missing policies and policies that do not cover all commands with the tenant setting; a negative test with deliberately unprotected tables. **No exceptions** |

**Decisions:**

| # | Decision |
| - | -------- |
| G1-1 | **Primary keys: UUIDv7**, generated by PostgreSQL 18's built-in `uuidv7()` as the column default. No UUID-generation dependency. *Scope clarified by the owner on 2026-11-26:* UUIDv7 is the default for newly introduced generated entity primary keys; natural and composite primary keys remain valid where appropriate; other exceptions need a concrete technical justification |
| G1-2 | **`tenants` retention class: *account*.** The retention period remains **undefined**, subject to legal review; none is invented |
| G1-3 | **`created_by` / `updated_by` deferred** until identity and authenticated actors exist |
| G1-4 | **`tenants` is platform-owned and global:** no `tenant_id` column, no RLS policy. Display names are **not** required to be unique |
| G1-5 | **`sms_migrator`:** owns application tables and runs migrations; not a superuser; no `BYPASSRLS`; cannot create roles; only the database and schema privileges needed to migrate (`CONNECT`; `USAGE`, `CREATE` on `public`) |
| G1-6 | **`sms_app`:** not a superuser; no `BYPASSRLS`; owns no tables or schemas; cannot create objects; no `TRUNCATE`; only the table privileges the API needs; inherits nothing from the migration role. The API refuses to start with an unsafe role or a table owner |
| G1-7 | **Existing local volumes are never deleted automatically.** Fresh volumes are provisioned by the official image initialisation (`/docker-entrypoint-initdb.d`); existing volumes by a documented, idempotent upgrade command ([`infra/local/README.md` §9](../../infra/local/README.md#9-database-roles-and-upgrading-an-existing-database-volume)). A volume reset is offered only as an explicit, user-typed alternative with its data-loss warning |
| G1-9 | **Confirmed by the owner on 2026-11-26, superseding the RLS evaluation's recommendation** (RLS-3's migration-role clause, evaluation §6 and §10): `sms_migrator` has **`NOBYPASSRLS`**. Both `sms_migrator` and `sms_app` are non-superuser and have no `BYPASSRLS`. `sms_migrator` may own application tables and run migrations, but forced RLS policies still apply to its ordinary data queries. Future cross-tenant data migrations must use a **separately designed, controlled procedure**; **no blanket RLS bypass** is granted. History and reasons: [RLS evaluation §11.1](../architecture/rls-evaluation.md#111-superseded-bypassrls-for-the-migration-role-2026-11-26) |
| G1-8 | **Out of scope:** user identity, authentication, sessions, MFA, RBAC; customer or administrator HTTP endpoints; audit logging needing authenticated actors; message submission or delivery; billing, monetary types, balances, pricing, ledger; telco adapters, routing, simulator; production deployment, provisioning or real credentials |

**Engineering details recorded with this group** (within the approved scope; listed so they can be reviewed):

- **Column privileges on `tenants`:** `sms_app` may `SELECT`, `INSERT (display_name)` and `UPDATE (status,
  updated_at)` only, so the database itself prevents the API from choosing a tenant's ID, initial status or
  timestamps. No `DELETE`.
- **Status transitions:** `pending → active` (activate), `active → suspended` (suspend), `suspended → active`
  (reactivate), as one conditional `UPDATE`, so concurrent transitions cannot both succeed.
- **Display name:** trimmed, 1–200 characters, no control characters, enforced in Go and by a `CHECK`
  constraint. The 200-character bound is an engineering limit, not a business rule.
- **Provisioning hardening:** `PUBLIC` loses `CONNECT` and `TEMPORARY` on the application database and `USAGE`
  on `public`; functions created later by `sms_migrator` are not executable by `PUBLIC` by default. The script
  refuses (changing nothing) if a role has memberships, owns the database or a schema, or if `public` contains
  objects owned by another role. It transfers only goose's version table on old volumes.
- **sqlc pin:** sqlc v1.31.1 in `backend/tools/go.mod` (builds without cgo, so the generation command also runs
  on Windows). As with kin-openapi (decision log §9), four of its indirect dependencies are raised to their
  first fixed releases so that `govulncheck` and the "Dependency review" job report nothing:
  `golang.org/x/net` v0.62.0, `google.golang.org/grpc` v1.83.2, `github.com/google/cel-go` v0.30.0 and
  `golang.org/x/text` v0.43.0 (advisories listed in `backend/tools/go.mod`). The oapi-codegen output is
  byte-identical. Nothing in `backend/tools` is linked into the API or migrate binaries.
- **CI:** role provisioning (run twice), sqlc regeneration and determinism check (Linux and Windows), the RLS
  catalog guard as a named step, `migrate` smoke-tested as `sms_migrator` and refused as the superuser, the API
  smoke-tested as `sms_app` and refused as the superuser and as `sms_migrator`, a govulncheck scan of sqlc, and
  the local stack check extended with role checks and the existing-volume upgrade. The required status-check
  names are unchanged.

**Windows validation (owner, 2026-11-26):** run with the checklist
[`infra/local/windows-validation.md`](../../infra/local/windows-validation.md#validation-record) on commit
`1ce253a`.

- **Environment:** Windows PowerShell 5.1, Docker Desktop 4.98.0, Engine 29.9.2, Compose v5.6.1.
- **Part A (fresh database): passed.** Every step from 0.1 to A11 passed:
  - role separation and role attributes;
  - the tenants migration and its privileges;
  - permission denials;
  - unsafe-role refusal and repair by re-provisioning;
  - health and readiness;
  - restart and persistence.
- **Part B (existing-volume upgrade): not tested on Windows.** No existing volume was present. It is verified on
  Linux only, by the local stack check in CI.

**Open follow-ups (not decided here):** the reporting role and the break-glass role (neither may receive a
blanket `BYPASSRLS` without its own approved design); production role provisioning and credential management;
the separately designed, controlled procedure for any future cross-tenant **data** migration (G1-9); the
retention period for the *account* class (legal review).

---

## 19. Owner decisions — messaging backend and provider integration foundation (Phase 4 Group 2, 2026-11-27)

On 2026-11-27 the owner approved **Phase 4 Group 2**: a working messaging backend with a provider-independent
integration boundary, as one Draft pull request. **No ADR status changed.** ADR-0012 stays *accepted — limited
scope*: the provider interface remains **provisional** and the simulator the only provider (§15).

**Approved scope (implemented):**

| Part | Implementation |
| ---- | -------------- |
| Public API v1 | `POST /v1/messages`, `GET /v1/messages/{message_id}`, `GET /v1/messages`, exactly as the contract defines them (statuses, reason codes, problem types, idempotency, cursor pagination) |
| Authentication | Bearer API keys (D-143): random 256-bit secret, only its SHA-256 hash stored, shown once, scopes `messages:send` / `messages:read`, revocable, tenant derived from the key; inactive tenants refused (`403`) |
| Persistence | Migration `20261127150000_create_messaging.sql`: tenant-owned tables with forced RLS and the approved policy; column-level `sms_app` grants; no `DELETE` or `TRUNCATE` |
| Durable dispatch | Transactional outbox (`dispatch_jobs` committed with the message), per-tenant claiming with `FOR UPDATE SKIP LOCKED` and a lease, attempts recorded before the provider call, recovery of abandoned attempts as uncertain |
| Provider boundary | `internal/messaging/provider`: submission, capability declaration (unknown is never support), normalized outcomes (accepted with reference, rejected, not sent, uncertain), normalized receipts |
| Simulator | The first and only adapter, in-process, deterministic scenarios by synthetic destination (backend README, "Messaging") |
| Receipts | Platform-owned inbox matched inside each tenant's context; duplicates and conflicts recorded without overwriting (D-098 open) |
| Local administration | `smsctl` (development and test only): tenants, sender IDs, API keys, message traces. No public management endpoint |

**Engineering decisions recorded with this group** (within the approved scope; listed so they can be reviewed):

| # | Decision |
| - | -------- |
| G2-1 | **No cross-tenant processing.** The dispatcher works tenant by tenant in each tenant's transaction-local context. Receipts, which carry only a provider reference, are stored in the platform-owned `provider_receipts` table (no `tenant_id`, no RLS) and correlated per tenant through RLS-protected submission attempts. No role bypasses RLS |
| G2-2 | **API key format** `smsk_<tenant>_<key id>_<secret>`: the visible part names the tenant, so authentication sets the tenant context before the key lookup, which RLS therefore also protects |
| G2-3 | **Uncertain outcomes** (timeout, adapter failure, an acceptance without a provider reference, an attempt abandoned by a crashed dispatcher) become `submit_uncertain` and are never resubmitted or failed (D-095 interim rule). Only definite "not sent" outcomes are retried, with backoff, an attempt limit and a maximum message age |
| G2-4 | **Destinations:** Philippine mobile numbers only (`+639` and nine digits, D-043); others get `422 destination-not-supported` |
| G2-5 | **Messages of a pending or suspended tenant stay queued**; they are neither sent nor failed |
| G2-6 | **Segment limit and reporting window** come from the configured provider; both are unknown/disabled by default (no universal default, messaging reliability §5.3). The simulator's values are synthetic |
| G2-7 | **The simulator is refused in production**, and production has no default provider (S6, PG-6) |
| G2-8 | **Not implemented** (no decided values or features): rate limits (`429`), credit reservation and `402`, suppression lists, sender ID and API key management endpoints, webhooks, retention expiry |

**Out of scope (unchanged):** real SMPP or telco adapters, any SMPP library, real credentials or endpoints,
third-party aggregators, fallback or multi-provider routing, billing, pricing, production deployment, frontend
changes, Kafka, RabbitMQ and Redis.

## 20. Owner decisions — first working client portal (Phase 4 Group 3, 2026-11-27)

On 2026-11-27 the owner approved **Phase 4 Group 3**: the first working version of the client portal (sign in,
view the organization's messages, compose and submit one SMS, see its status and details), connected to the Group 2
messaging backend, as one Draft pull request. Before implementation the agent identified four decisions that
blocked a secure implementation; the owner chose the recommended option for each on 2026-11-27 (G3-1 to G3-3 and
G3-6). **No ADR status changed.**

**Owner decisions:**

| # | Decision |
| - | -------- |
| G3-1 | **Login directory — the first approved exception to the RLS catalog guard.** Signing in with only an email address and a password needs one lookup before any tenant is known. The platform-owned table `portal_login_directory` maps a normalized email address to its tenant and user and holds **nothing else** (columns `email`, `tenant_id`, `user_id`, `created_at`; a test enforces the list). Portal users, password hashes, roles, lockout state and sessions stay in tenant-owned tables under forced RLS and are read only inside the tenant's transaction-local context. An email address is unique across the platform. Recorded in `rlsguard.Exceptions`; `TestOnlyApprovedExceptions` allows exactly this one |
| G3-2 | **Passwords are hashed with Argon2id** (OWASP parameters: 19 MiB memory, 2 iterations, 1 lane; 16-byte salt; PHC string format), using `golang.org/x/crypto` v0.59.0 (maintained by the Go project, BSD-3-Clause; already present in the module graph indirectly). At most four hashes run at once. Resolves the "Argon2id (proposed)" item of the security baseline for portal users |
| G3-3 | **Account and session policy:** roles Owner and Sender may read and send, Viewer may only read (Owner management functions are not in this group); **no MFA for client users** in this group (MFA stays mandatory for internal admin accounts, D-021); sessions end after **30 minutes without a request**, **12 hours after sign-in**, at sign-out or on revocation; **five consecutive failed sign-ins lock the account for 15 minutes**; passwords have **at least 12 characters**; local provisioning through `smsctl` (development and test only) |
| G3-6 | **Frontend:** a session-authenticated portal API contract (`contracts/openapi/portal-api-v1.yaml`), Go server code generated with the existing oapi-codegen, TypeScript types generated with **`openapi-typescript` 7.13.0** (development dependency only, MIT, types only; it declares a peer range of TypeScript 5 and is verified to work with the pinned TypeScript 6.0.3), one typed API client in `src/lib/api/`, and **no router or data-fetching library** (a small History-API router). Supersedes F3 and F4 of §10 for the client portal |

**Engineering decisions recorded with this group** (within the approved scope; listed so they can be reviewed):

| # | Decision |
| - | -------- |
| G3-4 | **No self-registration, no email.** `smsctl portal-user create` generates a random password and prints it once, or reads one from standard input (`--password-stdin`), never from a command-line flag. `portal-user list`, `unlock` and `revoke-sessions` complete the local administration |
| G3-5 | **Session and CSRF design.** The session cookie `__Host-sms_session` is `HttpOnly`, `Secure`, `SameSite=Strict`, `Path=/`; its value is `<tenant>.<session id>.<secret>` (the tenant part lets authentication set the tenant context first, like API keys, G2-2); only the SHA-256 hash of the 256-bit secret is stored. The CSRF token is derived from the secret (synchronizer token, not stored) and required in `X-CSRF-Token` on every state-changing request; sign-in accepts only `application/json`. Unknown addresses, wrong passwords and locked accounts get the same `401` after the same hashing work. The last-seen time is written at most once a minute. Sign-in and sign-out are logged with tenant and user IDs, never the email address or password |
| G3-7 | **One messaging engine.** Portal submissions go through `messaging.Service.Accept` with the same idempotency records (scope: organization + `create_message`), dispatcher, provider and status history as `POST /v1/messages`. The portal generates one idempotency key per intended send and reuses it on retries. The portal shows customer-visible statuses and history only, never provider details or the message text |
| G3-8 | **Narrowed frontend guardrails.** `fetch` is allowed only in `src/lib/api/client.ts`; the production CSP allows `connect-src 'self'`; the development proxy forwards `/portal` (never `/v1`); browser storage stays forbidden; tests keep the network guard and use an injected fake transport |
| G3-9 | **Not implemented** (out of scope or not decided): billing and cost estimates (the confirmation step says cost is not available), bulk sending, user and API key management, registration, password recovery, inbound SMS, webhooks, admin portal, analytics; per-IP sign-in rate limiting; an audit log table (S-8); a retention job for expired sessions (§10.8); serving the portal from a production host (D-145) |

**Out of scope (unchanged):** real SMPP or telco adapters, real credentials, third-party aggregators, billing,
pricing, production deployment, Kafka, RabbitMQ and Redis. The simulator remains the only provider.

## 21. Phase 4 Group 4 — enterprise client portal and API key management (2026-11-27)

On 2026-11-27 the owner approved **Phase 4 Group 4**: an enterprise-quality redesign of the client portal and
customer-facing API key management (list, create, revoke) for the public messaging API, as one Draft pull request.
No blocking decision was found: the existing approved decisions cover the authorization question. **No ADR status
changed.** G4-1 to G4-5 are the existing decisions applied and the engineering decisions made within the approved
scope, listed so they can be reviewed. G4-6 to G4-9 are **owner decisions of 2026-11-28** on the questions the
pull request left open.

| # | Decision |
| - | -------- |
| G4-1 | **Only owners manage API keys**, applying MVP-03 (§14: "Owner (manages users and API keys)"). The portal permission `api_keys:manage` belongs to the Owner role only; Sender and Viewer receive `403`. The tenant comes from the session; a key of another organization is reported as `404` |
| G4-2 | **One key service.** The portal endpoints (`GET`/`POST /portal/v1/api-keys`, `DELETE /portal/v1/api-keys/{id}`) call the same `apikey.Service` as `smsctl` and the public API's Bearer authentication, which is unchanged. Only the SHA-256 hash of a key's secret is stored; the full key is returned once, in the `201` response (`Cache-Control: no-store`), and never logged (creation and revocation are logged with tenant, user and key IDs). Revocation takes effect for the next public API request; revoking an already revoked key returns it unchanged. Key names are validated against the existing column constraint (1–100 characters, no control characters) |
| G4-3 | **No schema change.** The `api_keys` table has no last-used time, so the portal says that last use is not recorded rather than showing an invented value |
| G4-4 | **Design system extended, no UI library.** New semantic tokens (subtle surfaces and status tints, code-sample colours) with contrast tests; new shared primitives `Badge`, `Icon` (hand-drawn SVG line icons, decorative only), `Dialog` (native `<dialog>` with `showModal()`), and refined `Button` and `Card`. Navigation: sidebar on large screens, menu button below 1024 px; tables become stacked rows below 1280 px |
| G4-5 | **Not implemented** (out of scope): user management and invitations; MFA for client users; billing, pricing and costs; bulk sending; analytics. Last-used tracking, the audit log and key limits are decided in G4-6, G4-8 and G4-9 |

**Owner decisions (2026-11-28):**

| # | Decision |
| - | -------- |
| G4-6 | **API key last-used tracking — deferred** to a later group. When implemented, prefer **throttled** database updates (for example at most one write per key per interval, like the portal's last-seen time, G3-5) rather than a write on every public API request |
| G4-7 | **Password re-authentication before creating an API key — approved, required.** The signed-in owner re-enters their current password (`current_password` in `POST /portal/v1/api-keys`). Enforced by the backend (`portal.Service.Reauthenticate`), checked against the user's tenant-owned Argon2id credentials under the tenant context. Failed attempts count toward the existing sign-in lockout (five consecutive failures lock the account for 15 minutes, G3-3); while locked every attempt fails; a success clears the count. A wrong password or locked account receives `403` (`reauthentication-failed`), no key is created and nothing secret is returned. The password is never logged, stored or echoed; the frontend clears it after every attempt. The one-time display of a new key is unchanged |
| G4-8 | **Durable security audit log — required before production.** API key creation and revocation (with the other events of the security baseline, §12 of AGENTS.md) must be recorded as auditable security events in a durable, append-only audit trail. Ordinary application logs are **not** a substitute. The implementation (S-8) may follow in a subsequent group; until then the structured log lines (tenant, user and key IDs) remain operational logging only |
| G4-9 | **No fixed limit on API keys per organization** for now. No arbitrary limit is introduced; configurable limits are revisited when customer usage and operational requirements justify them |

**Out of scope (unchanged):** real SMPP or telco adapters, real credentials, third-party aggregators, billing,
pricing, production deployment, Kafka, RabbitMQ and Redis. The simulator remains the only provider.

---

## 22. Phase 4 Group 5 — enterprise client portal UI (2026-11-28)

On 2026-11-28, after Phase 4 Group 4 was merged, the owner approved **Phase 4 Group 5**: a refinement of the existing
client portal into an enterprise-grade SaaS console (application shell and navigation, page headers, typography,
spacing and surfaces, the messages list, message details, the compose and review flow, API key management, and
shared components and tokens), as one Draft pull request. It is a **frontend-only** group: routes, API contracts,
authentication, CSRF protection, role permissions, backend behaviour and database schemas are unchanged, and no
dependency or UI library is added. **No ADR status changed.** G5-1 to G5-5 are the engineering decisions made within
the approved scope, listed so they can be reviewed.

| # | Decision |
| - | -------- |
| G5-1 | **Tokens refined, not rebranded.** Cooler neutral surfaces and borders, a clearer primary blue, and navigation tokens (`nav-*`) for a dark sidebar, all `#rrggbb` with WCAG 2.2 AA contrast tests (4.5:1 text, 3:1 focus and boundaries), including a separate focus colour on the dark navigation. Still the light theme of owner decision F7; branding remains open |
| G5-2 | **Application shell.** A fixed dark sidebar from 1024 px (organization, navigation by permission, account and sign-out). Below 1024 px a top bar opens the same navigation in a modal panel on the native `<dialog>` (focus kept inside, Escape and outside click close it, focus returns to the menu button, closes after navigation). Pages use breadcrumbs instead of a back link |
| G5-3 | **Messages list density.** The status filter and refresh sit in a toolbar inside the table card, the loaded count and "Load more" in its footer. The table is used from 768 px (stacked rows below), with segments and last status update from 1280 px; the whole row opens the message (the message ID remains the row's link). No summary counts or charts are shown, because the API returns pages, not totals |
| G5-4 | **Details and compose.** Message details show a status panel (label, explanation, reason), a label–value list, a copy action for the message ID and a status history with status icons and the current status marked. Compose shows its two steps (compose, review and send) and the review includes the message text as entered; the idempotency, confirmation and error handling of Group 3 are unchanged |
| G5-5 | **Fixes found in browser checks.** After a key is revoked, focus moves to "Create API key" instead of being lost with the removed row button; table rows draw their own separators so that a row moved after a revoke cannot show a dark border; the API key permission and password re-authentication controls are grouped and labelled as before |

**Not implemented (out of scope, unchanged):** new features or endpoints, billing or cost display, analytics, user
management, admin portal, dark mode, new dependencies. G4-6 (last-used tracking) and G4-8 (durable audit log) remain
open (implemented in Phase 4 Group 6, §23).

---

## 23. Phase 4 Group 6 — security auditability and operational controls (2026-11-28)

On 2026-11-28, after Phase 4 Group 5 was merged, the owner approved **Phase 4 Group 6**: the outstanding security and
production-readiness requirements — a durable security audit log (G4-8), API key last-used tracking (G4-6) and per-IP
rate limiting of portal sign-in (G3-9) — with the database changes, tests and portal UI they need, as one Draft pull
request. Earlier decisions are not reopened: the account lockout (G3-3), password re-authentication (G4-7), the RLS
policy (RLS-2), role separation (RLS-4) and the one RLS guard exception (G3-1) are unchanged, and no RLS exception is
added. No dependency is added and Redis is not introduced. **No ADR status changed.** G6-1 to G6-9 are the
engineering decisions made within the approved scope, listed so they can be reviewed.

| # | Decision |
| - | -------- |
| G6-1 | **Two audit tables.** Events of an organization go to the tenant-owned `security_audit_events` (RLS enabled and forced, the approved policy). Events that identify no organization — a sign-in for an address that is not registered, malformed sign-in input, a rate-limited sign-in — go to the platform-owned `platform_security_events`, which has **no tenant column** (so the RLS guard needs no exception) and which the application role may insert into but **not read**. No platform-wide record is shown to any portal user and no internal operator role is introduced; database administrators read platform events until an approved admin function exists |
| G6-2 | **Append-only under the application's permissions.** `sms_app` has `INSERT` (and `SELECT` on its tenant's events) only — no `UPDATE`, `DELETE` or `TRUNCATE`; triggers also reject those statements by any role, including the table owner. Superusers, or the owner after disabling the triggers, can still alter rows: the log is **not** described as tamper-proof. Tamper evidence remains open (S-8) |
| G6-3 | **Same transaction.** API key creation and revocation, session creation (sign-in), sign-out, and each failed sign-in or re-authentication (with its failure count and any lock) are committed together with their audit event; when the event cannot be written the operation fails (`500`) instead of succeeding unaudited. Refusals by role are refused regardless, and a failure to record them is logged |
| G6-4 | **Events and contents.** Recorded: sign-in (success, failure, denied for an inactive organization), sign-out, account lock, API key creation (success, failed password confirmation, denied), revocation (success, denied), and refused API key list and audit log requests. Each event has a UUIDv7 ID, UTC time, tenant, actor type and ID, action, target, outcome, request ID, client IP, sanitized user agent and fixed non-secret details. Never passwords, tokens, API keys or hashes, authorization headers or message text; the address tried in a sign-in for an unregistered account is not stored. Ordinary message operations are not audited |
| G6-5 | **Owners read the audit log** (`audit_log:read`, Owner role only, as for API keys under MVP-03): `GET /portal/v1/audit-events` with action, outcome and time filters and cursor pages, and a read-only portal page. No export and no bulk operations. No automatic deletion: the **retention period remains an open decision** (S-4, §10.8) |
| G6-6 | **API key last use** (G4-6): after a key is authenticated **and authorized** for a public API request, `last_used_at` is written at most once per key per minute (a conditional update, safe across instances); the shown time can be up to a minute older than the latest request. Invalid, revoked or wrongly scoped requests never update it. A failure to record it does not fail the request |
| G6-7 | **Superseded by G6-11 (§23.1).** *Original:* **Per-source sign-in limit in PostgreSQL.** Default **50 attempts per source per 15 minutes** (`SIGNIN_RATE_LIMIT_MAX_ATTEMPTS`, `SIGNIN_RATE_LIMIT_WINDOW`); a source is an IPv4 address or an IPv6 `/64`. One atomic upsert per attempt, shared by all instances and surviving restarts; `429 too-many-sign-in-attempts` with `Retry-After`, before any credential check, so it never touches an account's lockout and reveals nothing about accounts. Ended windows are deleted every 5 minutes. Only the first refusal per source and window is audited |
| G6-8 | **Client IP.** The direct peer address, unless the peer is a configured trusted proxy (`TRUSTED_PROXIES`, none by default); then `X-Forwarded-For` is read right to left, skipping trusted proxies. Other forwarding headers are ignored. **A production deployment behind a load balancer must configure it** (hosting is undecided, D-145) |
| G6-9 | **Contract changes (portal API only, additive).** New `GET /portal/v1/audit-events` and schemas, permission `audit_log:read`, `ApiKey.last_used_at` (nullable, required), and `429` on sign-in. The public API is unchanged |

### 23.1 Security review remediation (2026-11-28)

On 2026-11-28 an independent security review of the Draft PR found: the Local Stack CI check expected a stale
migration version (B1); a signed-in Sender or Viewer could write one audit event per refused request without limit
(B2); the account lock was stored with the outcome `success` and relabelled "Locked" by the portal (B3); every
attempt on a locked account and every unknown-address sign-in wrote an event, and IPv6 address rotation escaped the
per-/64 limit (N1); 50 attempts per address and 15 minutes, counting successes, blocked shared addresses (N2); an
unset `TRUSTED_PROXIES` behind a load balancer was accepted silently in production (N3); combinations of broad
prefixes could trust every address (N4); the application supplied the audit time (back-dating possible); the
migration altered `api_keys` without `lock_timeout`; and `smsctl` user administration was not audited. The owner
authorized the remediation below (G6-10 to G6-18 are **owner decisions**; G6-19 and G6-20 record a requirement and
the contract changes). G6-7 is **superseded** by G6-11; G6-2, G6-4 and G6-8 are amended by G6-13 to G6-16.

| # | Decision |
| - | -------- |
| G6-10 | **Account lockout unchanged:** five failed attempts per account lock it for 15 minutes (G3-3). Nothing below weakens it |
| G6-11 | **Layered sign-in limits (supersedes G6-7).** (1) Failed-attempt budgets per IPv4 `/32` (100), IPv6 `/64` (100) and IPv6 `/48` aggregate (500) per 15 minutes; each attempt reserves a slot in every budget of its source atomically before the password is checked, and a successful sign-in refunds **only its own** slot, so earlier failures stay counted. (2) An all-attempt token bucket per IPv4 address or IPv6 `/64`: 10 requests per second, burst 100. Both live in PostgreSQL (atomic upserts; correct across instances and restarts). All values are initial, configurable defaults, not hard-coded |
| G6-12 | **Bounded Argon2 concurrency.** Password hashing runs with a configured concurrency, wait queue and maximum wait; beyond them `503 service-busy` with `Retry-After`, without checking the password and refunding the attempt's slot. Initial defaults (4, 32, 1 s) were measured on the development container only; measuring them on the production hardware is a production gate (S-12) |
| G6-13 | **Account lock outcome `enforced`.** `portal.account_locked` is stored with the outcome `enforced` (a database check makes `enforced` the outcome of this action and of nothing else); the API contract, the generated types and the portal filters and badges show the stored value; the portal's "Locked" override is removed |
| G6-14 | **Deduplicated refusals.** A refused Sender or Viewer request is recorded at most once per actor, action and five-minute interval; an attempt on a locked account at most once per account, action and lock. A unique `(tenant_id, dedup_key)` index and `INSERT … ON CONFLICT DO NOTHING`: no existing event is modified and no auxiliary record exists |
| G6-15 | **Trusted-proxy policy (amends G6-8).** In production an unset or empty `TRUSTED_PROXIES` prevents the API from starting; `none` declares that there is no proxy; otherwise the proxies' actual addresses or CIDRs. Forwarded headers are read only when the direct peer is a configured proxy. Prefixes broader than an IPv4 `/8` or an IPv6 `/32` are refused unless `TRUSTED_PROXIES_ALLOW_BROAD_RANGES=true` (startup warning); malformed entries, and entries that together cover a whole address family, are always refused |
| G6-16 | **Database-assigned audit identity and time (amends G6-4).** PostgreSQL assigns each event's `id` and `occurred_at`; the application role has no `INSERT` privilege on them, so events cannot be back-dated, collide with or overwrite existing ones. UTC throughout |
| G6-17 | **`smsctl` user administration audited.** Creating a portal user, clearing a lockout and revoking a user's sessions record `portal_user.create`, `portal_user.unlock` and `portal_user.sessions_revoke` (system actor, `via: smsctl`) in the same transaction as the change, as `sms_app` inside the tenant's context: role separation, RLS and the platform-event restrictions are unchanged. Tenant registry commands remain unaudited (platform-level; no platform audit design yet) |
| G6-18 | **Wording (amends G6-2).** The portal and the contract say that application permissions and triggers prevent ordinary application modification, and that a database superuser can still alter records; nothing describes the log as tamper-proof. Platform-wide event visibility (G6-1) and the open retention period (S-4) are unchanged; no retention period or automatic deletion is introduced |
| G6-19 | **Production requirement:** an external immutable audit export or externally anchored integrity evidence (security baseline S-8). An internal PostgreSQL hash chain is not claimed to protect against a database superuser |
| G6-20 | **Engineering corrections and contract changes.** The migration sets `lock_timeout` (5 s; tested); the local stack check derives the expected migration versions from the migration files and checks `migrate status`. Portal API contract: `AuditOutcome` gains `enforced` (a stored outcome changes from `success` to `enforced` for one action — the portal API is not yet published, and the Group 6 PR is unmerged, so no client depends on the old value); `AuditAction` gains the three `portal_user.*` actions; `AuditDetails` gains `dedup_interval_seconds`; sign-in and API key creation gain `503 service-busy`. The public API is unchanged |

**Open (owner decisions or later groups):** audit retention period and expiry (S-4); the external immutable audit
export (S-8, G6-19); edge rate limiting and production capacity measurements (S-12); an internal, MFA-protected view of platform security events (admin portal, D-021); rate limits of the
public API per tenant and key (security baseline §7); trusted-proxy configuration of the production hosting
(D-145). **Out of scope (unchanged):** real SMPP or telco adapters, billing, pricing, user management, admin portal,
Kafka, RabbitMQ and Redis.
