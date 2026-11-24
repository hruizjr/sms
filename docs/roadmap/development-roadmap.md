# Development Roadmap

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — pending owner review | Project owner | 2026-11-24 |

This roadmap sequences the work and defines **acceptance criteria** for each milestone. It deliberately
contains **no dates**: timing depends on telco, regulatory, and business inputs that are not yet known.
Each phase starts only with the owner's explicit approval.

## Overview

```
Phase 1  Engineering foundation ...................... (this phase)
   │
Phase 2  Discovery closure & solution architecture ◀── telco, regulatory, traffic, business inputs
   │
Phase 3  Walking skeleton (tooling, CI, local stack)   (can start in parallel with late Phase 2)
   │
Phase 4  Control-plane core: tenants, identity, RBAC, audit
   │
Phase 5  Billing ledger core
   │
Phase 6  Messaging core with simulator provider
   │
Phase 7  Client & admin portal features (MVP)
   │
Phase 8  First real telco integration (telco test environment)  ◀── signed agreement + specs required
   │
Phase 9  Production readiness: security, load, operations
   │
Phase 10 Controlled pilot (limited customers, production)
   │
Phase 11 Scale-out and expansion (more telcos, analytics, broker, features)
```

## Phases

### Phase 1 — Engineering foundation *(current)*
**Goal:** repository structure, agent instructions, standards, discovery and risk documents.
**Acceptance criteria:**
- [ ] Monorepo folders with READMEs exist; no application code.
- [ ] `AGENTS.md` and all Phase 1 documents created.
- [ ] Owner has reviewed documents and the decisions requiring approval.

### Phase 2 — Discovery closure and solution architecture
**Goal:** replace unknowns with facts and agree the architecture for the MVP.
**Depends on:** owner and external parties (telcos, counsel, prospective customers).
**Work:**
- Collect traffic data (traffic capacity discovery); telco technical answers (checklist); regulatory
  guidance; business decisions (billing basis, credit model, MVP scope, launch customers).
- Write: MVP Software Requirements Specification, domain model, system architecture, threat model,
  RBAC design, data classification and retention schedule (draft).
- Decide ADRs 0002–0017 (accept/reject/amend).
**Acceptance criteria:**
- [ ] MVP scope approved in writing by the owner.
- [ ] Traffic figures for launch customers recorded with sources, or an explicit decision to proceed with
      provisional targets.
- [ ] At least one telco's technical specification received, or an explicit decision to proceed
      simulator-only.
- [ ] Regulatory path documented by counsel (even if approvals are pending).
- [ ] Architecture document and threat model approved.

### Phase 3 — Walking skeleton
**Goal:** a minimal end-to-end technical skeleton with no business features.
**Depends on:** ADR-0002/0003/0004/0013/0014/0015/0016 accepted (does not depend on telco answers).
**Work:** `go.mod`, `cmd/api` with health/readiness endpoints, config loading, logging, metrics endpoint;
frontend app(s) with design tokens and app shell; local Docker Compose with PostgreSQL; migration tooling;
CI (format, lint, test, type check, vulnerability and secret scan); OpenAPI skeleton with type generation.
**Acceptance criteria:**
- [ ] A new developer/agent can start the local stack with one documented command.
- [ ] CI runs on every PR and blocks merging on failure.
- [ ] Branch protection configured on `main`.
- [ ] Health endpoint served; frontend shell renders; tokens applied; all tests pass.

### Phase 4 — Control-plane core
**Goal:** tenants, users, authentication (with admin MFA), RBAC, audit log.
**Acceptance criteria:**
- [ ] Admin can create tenants and users; users can log in; MFA enforced for admins.
- [ ] Permission checks on every endpoint; authorization matrix tests pass.
- [ ] Cross-tenant access tests pass.
- [ ] Security-relevant actions appear in the audit log.

### Phase 5 — Billing ledger core
**Depends on:** business decisions D-080 to D-084.
**Acceptance criteria:**
- [ ] Top-up, reservation, settlement, release, refund, and adjustment entries implemented and audited.
- [ ] Concurrency tests prove no overspending; balance equals ledger sum.
- [ ] Idempotent ledger operations proven by tests.

### Phase 6 — Messaging core with simulator
**Goal:** accept → reserve → persist → queue → dispatch → simulated DLR → status → settle.
**Acceptance criteria:**
- [ ] Public API submission with idempotency keys; segmentation and cost estimate.
- [ ] Provider interface and simulator adapter; provider contract suite passing.
- [ ] Rate limiting, retry classification, ambiguous-outcome handling, DLR correlation, expiry.
- [ ] Opt-out/suppression enforced before dispatch.
- [ ] Baseline load test results documented (not production targets).
- [ ] No message lost or double-charged across crash/restart tests.

### Phase 7 — Portal features (MVP)
**Acceptance criteria:**
- [ ] MVP client portal features (per approved SRS): sending, contacts/imports, campaigns, reports, API keys, balance.
- [ ] MVP admin features: tenant management, sender ID approval, credit allocation, pricing, campaign approval, monitoring views.
- [ ] E2E tests for critical journeys pass; accessibility checks pass.

### Phase 8 — First real telco integration (test environment)
**Depends on:** signed agreement (or written test access), official technical specification, test
credentials, network connectivity (VPN/IP allowlisting).
**Acceptance criteria:**
- [ ] Adapter passes the provider contract suite against the simulator configured to the telco's documented behaviour.
- [ ] Documented test cases pass in the **telco's test environment**, with evidence recorded.
- [ ] Telco acceptance/certification (if required) completed.

### Phase 9 — Production readiness
**Acceptance criteria:**
- [ ] Load tests meet **verified** targets with margin.
- [ ] Independent penetration test completed; high/critical findings fixed.
- [ ] Backups and restore tested; RPO/RTO met.
- [ ] Monitoring, alerting, runbooks, on-call/operations model in place.
- [ ] Regulatory and privacy prerequisites confirmed by counsel.

### Phase 10 — Controlled pilot
**Acceptance criteria:**
- [ ] Limited set of approved customers live with production traffic caps.
- [ ] Reconciliation with telco records performed and discrepancies explained.
- [ ] Incident and abuse handling exercised.

### Phase 11 — Scale-out and expansion
Additional telcos, broker adoption (ADR-0006), analytics store (ADR-0008), webhooks, two-way messaging,
self-service onboarding, payments — each justified by verified need.

## Critical external dependencies

| Dependency | Blocks |
| ---------- | ------ |
| Telco technical specifications | Phase 8 (and final provider interface) |
| Telco agreements and connectivity | Phase 8–10 |
| Regulatory authorizations | Phase 10 (production) |
| Verified traffic data | Phase 9 targets, infrastructure sizing |
| Business decisions on billing | Phase 5 |
| MVP scope | Phase 4–7 detail |
