# Architecture Decision Readiness

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — Phase 2A snapshot; ADR statuses are kept in the decision log (ADR-0018 and, with a limited scope, ADR-0021 accepted 2026-11-25) | Project owner | 2026-11-25 |

This document classifies every Proposed ADR by **what it is waiting for**, so the owner can see which
decisions are possible now and which must wait for discovery. It complements the review summary in the
[decision log §5](../engineering/decision-log.md#5-review-summary-2026-11-24); the ADRs themselves remain
the authority for their content. This classification is a snapshot from 2026-11-24; **current statuses are in the
[decision log index](../engineering/decision-log.md#4-index)** (on 2026-11-25 the owner accepted ADR-0002–0005 and
0013–0017, and later that day ADR-0018 and, limited to the customer-visible status model, ADR-0021). Only the owner can accept or reject ADRs.

## 1. Classification

Legend: ● primary dependency · ○ secondary dependency.

| ADR | Topic | Can potentially be approved now | Owner preference | Business / commercial info | Telco technical specs | Regulatory / legal advice | Implementation-time benchmarking |
| --- | ----- | :-----------------------------: | :--------------: | :------------------------: | :-------------------: | :-----------------------: | :------------------------------: |
| 0001 | ADR process | ● | | | | | |
| 0002 | Monorepo layout | ● | ○ | | | | |
| 0003 | Modular monolith | ● | | | | | |
| 0004 | Chi router | ● | ● | | | | |
| 0005 | PostgreSQL | ● | | | | | |
| 0006 | Broker deferred, with criteria | ● (the deferral) | | ○ (traffic) | | | ● (eventual choice) |
| 0007 | Redis deferred | ● (the deferral) | | | | | ● (eventual need) |
| 0008 | ClickHouse deferred | ● (the deferral) | | ● (reporting needs) | | | ○ |
| 0009 | Control/data plane separation | ● | | ○ (availability targets) | | | |
| 0010 | Multi-tenancy and RLS | ○ (shared-schema baseline) | | ○ (dedicated-infrastructure demands) | | | ● (RLS evaluation) |
| 0011 | Credit ledger | | | ● (D-080–D-084) | ○ (billing basis) | ○ (tax/invoicing) | |
| 0012 | Provider abstraction | ● (the principle) | | | ● (interface shape) | | |
| 0013 | Frontend structure and components | | ● (D-091, D-140) | | | | |
| 0014 | Database tooling | | ● (dependency approval) | | | | ● (tool maintenance check) |
| 0015 | OpenAPI-first | ● | | | | | |
| 0016 | `log/slog` | ● | | | | | |
| 0017 | Git workflow | ● | | | | | |
| 0018 | API idempotency contract | | ● (D-094) | ○ (customer retry behaviour) | | | |
| 0019 | Monetary representation | | | ● (real rates, pricing model) | ● (rate precision, D-117) | ○ (tax, D-127) | |
| 0020 | Transactional outbox, at-least-once | ● | | | | | |
| 0021 | Message status model | | ○ | ● (D-096, D-099) | ● (receipt behaviour, D-104, D-114–D-116) | | |

## 2. What the owner can decide now

These do not depend on external information. Accepting them gives Phase 2B and Phase 3 a stable base:

- **ADR-0001, 0002, 0003, 0005, 0009, 0015, 0016, 0017, 0020** — foundational choices with no external
  dependency.
- **ADR-0004** — Chi is the owner's stated preference; acceptance is a preference decision.
- **ADR-0006, 0007, 0008** — accept the **deferrals and reconsideration criteria**; the eventual technology
  choices remain open until measurements justify them.
- **ADR-0012** — accept the **principle** (adapters, capabilities, simulator); the concrete interface waits
  for a telco specification.

Owner-preference decisions that should be made **before Phase 3 scaffolding**:

- **ADR-0013** — one or two portals (D-091) and whether to allow a headless UI library (D-140).
- **ADR-0014** — approve `pgx` + `sqlc` and pick a migration tool at Phase 3 start.
- **ADR-0018** — idempotency details (D-094); customer answers (CU-20) inform it.

## 3. What must wait

| ADR | Waiting for | Expected source | Register items |
| --- | ----------- | --------------- | -------------- |
| 0010 (RLS part) | Evaluation with connection pooling and performance tests | Phase 3 engineering spike | D-141 |
| 0011 | Billing basis, credit unit, payment model, refund policy | Owner decisions after telco terms | D-080–D-084 |
| 0019 | Real rate cards and pricing model; tax treatment | Telcos, owner, accountant | D-092, D-093, D-117, D-127 |
| 0021 | Customer status labels; uncertain-message billing; telco receipt behaviour. **Update 2026-11-25:** status labels decided and ADR-0021 accepted with a limited scope; uncertain-message billing, webhook behaviour, and telco receipt behaviour still pending | Owner, telcos | D-095–D-099, D-104, D-114–D-116 |
| 0012 (interface) | At least one official telco specification | Telcos | D-100, D-104–D-106, D-114, D-115 |

## 4. Decisions not yet captured as ADRs

These will need ADRs during Phase 2B or later. They are listed so they are not forgotten; **no ADR is
created for them in Phase 2A** because their inputs are missing.

| Future decision | Alternatives (illustrative) | Depends on |
| --------------- | --------------------------- | ---------- |
| Hosting and telco connectivity model | Public cloud with static egress IPs and VPN to telcos · Philippine colocation/data centre · Hybrid (connectors near telcos, rest in cloud) | D-102, D-068, D-129, D-090, D-044 |
| Direct vs. intermediary connectivity | Direct to each telco · Via intermediary · Hybrid (intermediary first). The provider abstraction (ADR-0012) supports all three technically; the trade-offs are commercial and regulatory | D-160, D-118, D-113, D-120 |
| Routing and network identification | Prefix tables · Porting/lookup data from telcos · Rely on telco off-net delivery | D-070, D-107, D-119 |
| SMPP (or other protocol) client implementation | Library vs. in-house implementation | D-100 and the official specification |
| Data retention schedule | Per data class, per legal requirement | D-124, D-073, D-121 |
| Customer-facing SLA and recovery targets | Single-site vs. multi-site; backup-only vs. replicated | D-067, D-076, D-090 |

## 5. Architecture direction that is safe regardless

Whatever discovery reveals, the Phase 1 constraints in `AGENTS.md` §6 (provider abstraction, control/data
plane separation, tenant isolation, ledger integrity, at-least-once with idempotency) remain valid. They
were chosen precisely because they do not depend on the unknowns above. Phase 2B can therefore proceed with
the domain model and threat model in parallel with commercial discovery.
