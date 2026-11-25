# Development Roadmap

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — pending owner review | Project owner | 2026-11-24 |

This roadmap sequences the work and defines **acceptance criteria** for each milestone. It deliberately
contains **no dates**: timing depends on telco, regulatory, and business inputs that are not yet known.
Each phase starts only with the owner's explicit approval.

## Overview

```
Phase 1  Engineering foundation ...................... COMPLETE
   │
Phase 2  Discovery closure & solution architecture ◀── telco, regulatory, traffic, business inputs
         (2A preparation package merged; external discovery and 2B ONGOING)
   │
Phase 3  Walking skeleton (tooling, CI, local stack)   AUTHORIZED, IN PROGRESS (Group 0 in review)
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

### Phase 1 — Engineering foundation *(complete)*
**Goal:** repository structure, agent instructions, standards, discovery and risk documents.
**Status:** Complete — reviewed and merged by the owner (PR #1).
**Acceptance criteria:**
- [x] Monorepo folders with READMEs exist; no application code.
- [x] `AGENTS.md` and all Phase 1 documents created.
- [x] Owner has reviewed documents and the decisions requiring approval.

### Phase 2 — Discovery closure and solution architecture
**Goal:** replace unknowns with facts and agree the architecture for the MVP.
**Status:** Ongoing. Phase 2A (discovery preparation package) is merged (PR #2). The actual external
discovery (telcos, counsel, customers) and Phase 2B (SRS, domain model, system architecture, threat model) are
**not yet done**; no acceptance criterion below is met.
**Depends on:** owner and external parties (telcos, counsel, prospective customers).
**Work:**
- Collect traffic data (traffic capacity discovery); telco technical answers (checklist); regulatory
  guidance; business decisions (billing basis, credit model, MVP scope, launch customers).
- Write: MVP Software Requirements Specification, domain model, system architecture, threat model,
  RBAC design, data classification and retention schedule (draft).
- Decide ADRs 0001–0021 (accept/reject/amend) — see the review summary in the decision log.
- Decide monetary representation and rounding (ADR-0019, D-092, D-093), the idempotency details (D-094),
  and the uncertain-outcome and receipt policies (D-095–D-099).
**Discovery package:** [Phase 2 discovery guide](../discovery/phase2-discovery-guide.md); progress against the
criteria below is tracked in the [Phase 2 exit checklist](../discovery/phase2-exit-checklist.md).
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
**Status:** Authorized by the owner on 2026-11-25 and **in progress**. It will be delivered in six groups, each
reviewed and merged before the next starts
([decision log §7](../engineering/decision-log.md#7-owner-decisions--phase-3-walking-skeleton-2026-11-25)).
**Application implementation has not started.**

| Group | Scope | State |
| ----- | ----- | ----- |
| 0 | Record owner decisions (documentation only) | Submitted — awaiting owner review and merge (PR #3) |
| 1 | Go API skeleton, with basic CI | Not started |
| 2 | PostgreSQL, migrations, local database | Not started |
| 3 | API contract (OpenAPI) | Not started |
| 4 | Frontend | Not started |
| 5 | Local development stack and expanded CI | Not started |

**Depends on:** ADR-0002/0003/0004/0005/0013/0014/0015/0016/0017 — accepted 2026-11-25; owner repository
actions — done (git-workflow §0).
**Work:** `go.mod`, `cmd/api` with health/readiness endpoints, config loading, logging, metrics endpoint;
frontend app(s) with design tokens and app shell; local Docker Compose with PostgreSQL; migration tooling;
CI (format, lint, test, type check, vulnerability and secret scan); OpenAPI skeleton with type generation.
**Acceptance criteria:**
- [ ] A new developer/agent can start the local stack with one documented command.
- [ ] CI runs on every PR and blocks merging on failure.
- [x] Branch protection configured on `main` (owner-confirmed 2026-11-25; required CI checks to be added once
      workflows exist).
- [ ] Health endpoint served; frontend shell renders; tokens applied; all tests pass.
- [ ] RLS evaluation completed and documented (ADR-0010, D-141), before Phase 4.

### Phase 4 — Control-plane core
**Goal:** tenants, users, authentication (with admin MFA), RBAC, audit log.
**Acceptance criteria:**
- [ ] Admin can create tenants and users; users can log in; MFA enforced for admins.
- [ ] Permission checks on every endpoint; authorization matrix tests pass.
- [ ] Cross-tenant access tests pass.
- [ ] Security-relevant actions appear in the audit log.

### Phase 5 — Billing ledger core
**Depends on:** business decisions D-080 to D-084, D-092, D-093; ADR-0011 and ADR-0019 accepted.
**Acceptance criteria:**
- [ ] Top-up, reservation, settlement, release, refund, and adjustment entries implemented and audited.
- [ ] Concurrency tests prove no overspending; balance equals ledger sum.
- [ ] Idempotent ledger operations proven by tests.
- [ ] No floating-point types in any monetary path; rounding follows the accepted rules (FI scenarios pass).

### Phase 6 — Messaging core with simulator
**Goal:** accept → reserve → persist → queue → dispatch → simulated DLR → status → settle.
**Acceptance criteria:**
- [ ] Public API submission with idempotency keys per the accepted ADR-0018; segmentation and cost estimate.
- [ ] Outbox scenarios (OB-1 to OB-9), submission scenarios (PS-1 to PS-7), and receipt scenarios
      (DR-1 to DR-10) pass, including duplicate publication and duplicate consumption.
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
