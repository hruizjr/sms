# Architecture Principles

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — pending owner review | Project owner | 2026-11-24 |

These principles guide the architecture phase. They are **not** the final system architecture, which
will be defined after discovery (traffic, telco specifications, MVP scope). Individual decisions are
recorded as ADRs in the [decision log](../engineering/decision-log.md).

---

## P1. Start as a modular monolith; split only with evidence

**What it means:** One Go codebase with strict internal module boundaries, built into a few executables
(e.g. API server and message dispatcher), rather than many independently deployed microservices.

**Why:** Microservices add network failures, distributed transactions, deployment pipelines, and
operational overhead for every service. With one human owner and AI agents implementing, that overhead
would slow us down and increase failure modes. A well-structured modular monolith can be split later.

**When to split a module into its own service:** when there is evidence — independent scaling needs,
different availability requirements, or isolation of failure — recorded in an ADR.

## P2. Separate the control plane from the data plane

**Control plane:** tenants, users, authentication, configuration, pricing, billing, administration,
reporting. Lower volume, rich business logic, human-facing.

**Data plane:** message acceptance, queueing, routing, rate limiting, telco submission, delivery receipt
processing. High volume, latency-sensitive, machine-facing.

**Why separate:** they have different scaling and reliability needs. A slow admin report must never
delay OTPs; a campaign burst must never take down the portals.

**How (proposed):** separate modules from day one; separate processes (`api`, `dispatcher`, and possibly
per-telco connectors) as soon as messaging exists. The data plane reads the configuration it needs
(routes, rate limits, tenant status, prices) through explicit interfaces, with caching where justified,
so it can keep working briefly if the control plane is degraded. Whether message acceptance itself
belongs to the API process or a separate ingestion service is an architecture-phase decision.

## P3. Telcos are adapters behind one provider interface

- Business logic depends on a `Provider` abstraction (send, receive receipts, report health), never on
  Smart- or Globe-specific code.
- Each telco adapter translates between our internal model and that telco's protocol.
- Adapter differences (encoding, DLR format, error codes, TPS) are captured in the adapter and its
  configuration — backed by telco documentation, never assumed.
- Adapters declare **capabilities** (e.g. receipt support, per-segment receipts, status query, provider-side
  duplicate detection); core logic consults capabilities, never provider names (ADR-0012).
- A **simulator adapter** behaves like a configurable telco (latency, failures, DLR delays) so the whole
  system is testable without telco access.
- All adapters must pass one shared **provider contract test suite**.

## P4. The database is the source of truth; events are derived

- PostgreSQL holds the authoritative state of tenants, messages, and money.
- A state change and the outbox record announcing it are written in the same transaction (transactional
  outbox, ADR-0020). Rolled-back changes publish nothing; committed events are published **at least once**
  and may be published more than once, so consumers process events idempotently. Details:
  [messaging reliability §2](messaging-reliability.md#2-internal-events-and-the-transactional-outbox).
- Caches and analytics stores are derived and rebuildable; they are never authoritative for money or
  message state.

## P5. Assume at-least-once; design for idempotency and reconciliation

- Any message or event may be processed more than once; every handler must be safe to repeat.
- Exactly-once **SMS delivery cannot be guaranteed**: if a telco submission times out, we cannot know
  whether the telco accepted it. Policies for these ambiguous cases are explicit per provider.
- API acceptance, provider acknowledgment, and final receipts are different milestones; a missing receipt
  is an **uncertain** outcome, not proof of non-delivery
  ([messaging reliability §3–§5](messaging-reliability.md#3-message-lifecycle-milestones)).
- Reconciliation jobs compare independent records (our states vs. DLRs; our ledger vs. telco CDRs) and
  surface discrepancies.

## P6. Money is exact, append-only, and auditable

- Exact arithmetic only, never floating point. The representation (scaled integer vs. exact decimal),
  scale, and rounding rules are **open** until real rates and the pricing model are known (ADR-0019,
  [financial integrity](financial-integrity.md)).
- Every balance change is a ledger entry linked to its cause (message, top-up, admin adjustment).
- Concurrency control at the database level prevents overspending under parallel requests.
- Balances are derivable from the ledger; any cached balance is verifiable against it.

## P7. Tenant isolation is enforced in depth

- Application layer: tenant derived from identity; every query scoped.
- Database layer: `tenant_id` on all tenant data; composite constraints; Row-Level Security as an extra
  guard on tenant-owned control-plane tables (adopted 2026-11-26, ADR-0010; data-plane tables deferred).
- Test layer: automated cross-tenant access tests for every tenant-scoped feature.

## P8. Explicit contracts at every boundary

- Public and portal APIs: OpenAPI specifications in `contracts/`.
- Internal events: versioned schemas.
- Provider interface: Go interface plus contract test suite.
- Contracts change only deliberately, with compatibility analysis.

## P9. Scale by design, provision by evidence

- Stateless application processes; state in the database/broker.
- Partitionable data (by time and/or tenant) for high-volume tables.
- Throughput limits and concurrency as configuration.
- Infrastructure added only when verified traffic, telco capacity, and measurements justify it.
- Telco contracted throughput is likely the real ceiling; buffering bursts matters more than unlimited
  intake speed.

## P10. Observability is a feature

- Every process exposes metrics (Prometheus format proposed), structured logs, and health endpoints.
- Key business metrics are first-class: accepted/submitted/delivered/failed per provider and tenant,
  queue depth and age, DLR latency, bind status, credit reservation backlog.
- Alerts are defined alongside the features they protect.

## P11. Secure and private by default

- Deny-by-default authorization, least privilege, MFA for administrators.
- Minimize and protect personal data (phone numbers, message content) under the Data Privacy Act.
- Abuse prevention (sender ID approval, content policies, campaign approval) is part of the core design.

## P12. Fail safely and visibly

- Prefer rejecting or queueing work over silently dropping it.
- Back-pressure instead of unbounded buffering.
- Circuit breakers for failing providers; operator-visible failure states; dead-letter storage.
- Graceful shutdown that does not lose accepted work.

## P13. Keep it simple and reversible

- Prefer standard library and boring, well-documented technology.
- Prefer decisions that are cheap to reverse; spend design effort on those that are not (data model,
  ledger, public API, tenant model).

---

## Open architecture questions (for the architecture phase)

| # | Question | Depends on |
| - | -------- | ---------- |
| AQ-1 | Which queue/broker, and when? (Postgres-backed queue, Kafka, Redpanda, other) | Traffic discovery, operations capacity |
| AQ-2 | Is message acceptance part of the API process or a separate ingestion service? | Peak throughput, availability targets |
| AQ-3 | How are telco connector processes deployed given limited bind counts (active/standby, leader election)? | Telco bind limits |
| AQ-4 | How is the recipient's network determined (prefix, MNP database, HLR, telco off-net acceptance)? | Telco and regulatory answers |
| AQ-5 | Message table partitioning strategy and retention. | Retention requirements, volume |
| AQ-6 | When is ClickHouse (or another analytics store) needed vs. PostgreSQL aggregates? | Reporting requirements, volume |
| AQ-7 | Where is the platform hosted, and how does it connect to telcos (VPN, private link, static IPs)? | Telco connectivity requirements, data residency |
| AQ-8 | High availability and disaster recovery targets (RPO/RTO). | Business SLAs, budget |
| AQ-9 | How are rate limits coordinated across multiple dispatcher instances? | Instance count, telco limits |
| AQ-10 | Billing settlement point (submission vs. delivery) and its effect on the ledger flow. | Business decision D-080 |
| AQ-11 | Should PostgreSQL Row-Level Security be adopted, given pooling and performance? | **Answered 2026-11-26:** adopted for tenant-owned control-plane tables (ADR-0010 accepted); high-volume data-plane tables need their own evaluation (RLS-5) |
| AQ-12 | Monetary representation, scale, and rounding. | Actual rates and pricing model (ADR-0019) |
| AQ-13 | Policy for uncertain submissions and `UNKNOWN` messages. | Business decisions D-095, D-096; telco capabilities |
