# Development Roadmap

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — pending owner review | Project owner | 2026-11-26 |

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
Phase 3  Walking skeleton (tooling, CI, local stack)   COMPLETE (2026-11-26)
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
**not yet done**. Two acceptance criteria below are met (MVP scope and the simulator-only decision, 2026-11-26);
**Phase 2 is not complete**.
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
- [x] MVP scope approved in writing by the owner. **Met 2026-11-26:** all 34 capabilities decided (21 MVP,
      8 Post-MVP, 5 Deferred) with production gates PG-1 to PG-8 ([MVP scope worksheet](../discovery/mvp-scope-worksheet.md#owner-approval-of-the-mvp-scope-2026-11-26);
      [decision log §14](../engineering/decision-log.md#14-owner-decisions--mvp-scope-phase-2-ex-1-2026-11-26)). Launch customers TBD until pilot discovery (EX-1 amended;
      D-061).
- [ ] Traffic figures for launch customers recorded with sources, or an explicit decision to proceed with
      provisional targets.
- [x] At least one telco's technical specification received, or an explicit decision to proceed
      simulator-only. **Met 2026-11-26 by owner decision:** simulator-only development and testing (S1–S8),
      gate PG-9, telco discovery deferred (EX-7) ([decision log §15](../engineering/decision-log.md#15-owner-decisions--simulator-only-development-phase-2-ex-3-2026-11-26)). No telco
      specification has been received.
- [ ] Regulatory path documented by counsel (even if approvals are pending).
- [ ] Architecture document and threat model approved.

### Phase 3 — Walking skeleton
**Goal:** a minimal end-to-end technical skeleton with no business features.
**Status:** **Complete (2026-11-26).** Authorized by the owner on 2026-11-25; every exit criterion below is met, and
the closure is recorded in [decision log §13](../engineering/decision-log.md#13-phase-3-closure-2026-11-26). It was delivered in eight groups, 0–7 (six
until the owner's renumbering of 2026-11-25, [decision log §9](../engineering/decision-log.md#9-owner-decisions--go-contract-integration-phase-3-group-4-2026-11-25)), each
reviewed and merged before the next starts
([decision log §7](../engineering/decision-log.md#7-owner-decisions--phase-3-walking-skeleton-2026-11-25)). The RLS evaluation, planned as a
separate spike before Phase 4, was approved by the owner as Group 7 on 2026-11-26
([decision log §12](../engineering/decision-log.md#12-owner-decisions--rls-evaluation-spike-phase-3-group-7-2026-11-26)).
**Implementation status:** all eight groups (0–7) are merged. Group 7 (RLS evaluation spike) was the last, in PR #11.

| Group | Scope | State |
| ----- | ----- | ----- |
| 0 | Record owner decisions (documentation only) | Merged (PR #3) |
| 1 | Go API skeleton, with basic CI | Merged (PR #4) |
| 2 | PostgreSQL, migrations, local database | Merged (PR #5) |
| 3 | API contract (OpenAPI) | Merged (PR #7) |
| 4 | Go API contract integration: Go types and Chi routes generated from the contract (oapi-codegen), reproducible generation and CI checks; no endpoint mounted | Merged (PR #8) |
| 5 | Frontend foundation (Group 4 before the renumbering): pnpm workspace, client portal shell (one page), shared UI package with tokens, Button and Card, lint, type check, tests, build, Linux and Windows CI ([decision log §10](../engineering/decision-log.md#10-owner-decisions--frontend-foundation-phase-3-group-5-2026-11-25)) | Merged (PR #9) |
| 6 | Local development stack and expanded CI (Group 5 before the renumbering): Docker Compose with PostgreSQL, a one-off migration service and the API; client portal development proxy; Local stack check, secret scan and dependency review in CI ([decision log §11](../engineering/decision-log.md#11-owner-decisions--local-development-stack-and-expanded-ci-phase-3-group-6-2026-11-26)) | Merged (PR #10) |
| 7 | PostgreSQL Row-Level Security evaluation spike (ADR-0010, D-141): integration-tagged tests on synthetic, test-only tables against a disposable PostgreSQL 18.7; written [evaluation](../architecture/rls-evaluation.md) with threat model, cross-tenant models, pitfalls and indicative performance; no product code. After review, the owner **accepted ADR-0010** (option B for tenant-owned control-plane tables; data-plane tables deferred) on 2026-11-26 ([decision log §12](../engineering/decision-log.md#12-owner-decisions--rls-evaluation-spike-phase-3-group-7-2026-11-26)) | Merged (PR #11) |

**Depends on:** ADR-0002/0003/0004/0005/0013/0014/0015/0016/0017 — accepted 2026-11-25; owner repository
actions — done (git-workflow §0).
**Work:** `go.mod`, `cmd/api` with health/readiness endpoints, config loading, logging, metrics endpoint;
frontend app(s) with design tokens and app shell; local Docker Compose with PostgreSQL; migration tooling;
CI (format, lint, test, type check, vulnerability and secret scan); OpenAPI skeleton with type generation.
**Acceptance criteria:**
- [x] A new developer/agent can start the local stack with one documented command. After a one-time copy of
      `infra/local/.env.example` to `infra/local/.env`, `docker compose -f infra/local/compose.yaml up -d --build --wait`
      ([`infra/local/README.md`](../../infra/local/README.md) §2). It is verified on Linux by the "Local stack check" CI job,
      and on Windows by the owner's Docker Desktop validation on 2026-11-26.
- [x] CI runs on every PR and blocks merging on failure. All five workflows run on every pull request to `main`, with
      no path filters. Ruleset R1 requires nine status checks; the owner updated it on 2026-11-26
      ([git workflow §0](../engineering/git-workflow.md#0-repository-state)).
- [x] Branch protection configured on `main` (owner-confirmed 2026-11-25). The required CI checks were added on
      2026-11-26 (see the previous item).
- [x] Health endpoint served; frontend shell renders; tokens applied; all tests pass.
      - **Health:** `/healthz` and `/readyz` are checked by the API smoke test in "Go test and build" and by "Local stack
        check".
      - **Shell and tokens:** the render, error-boundary and token-contrast tests and the build check run in "Frontend
        check".
      - **Tests:** all five workflows passed on `main` at the Group 7 merge (`63ab5de`).
- [x] RLS evaluation completed and documented (ADR-0010, D-141), before Phase 4. Done in Group 7
      ([RLS evaluation](../architecture/rls-evaluation.md)); owner decision 2026-11-26: RLS adopted for tenant-owned
      control-plane tables, data-plane tables deferred (RLS-5).

### Phase 4 — Control-plane core
**Goal:** tenants, users, authentication (with admin MFA), RBAC, audit log.
**Depends on:** database role separation (owner decision RLS-4, 2026-11-26), delivered in its own approved group
before or alongside the first tenant-owned table; the API must not use the bootstrap PostgreSQL superuser. Tenant
tables follow the adopted RLS conditions (ADR-0010).
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
- [ ] MVP client portal features (per approved SRS): single-recipient sending (one manually entered mobile number per
      send operation; no pasted lists, recipient files, or group sends), reports, API keys, balance. Bulk sending and
      campaign sending are deferred to a later release (owner decisions 2026-11-25, MVP-08, MVP-09); saved contacts,
      contact groups, and campaign scheduling are deferred as well (owner decisions 2026-11-26, MVP-11, MVP-12).
- [ ] MVP admin features: tenant management and activation, sender ID approval, credit allocation, pricing, abuse
      controls (velocity limits, alerts, kill switch), audit log, monitoring views. Campaign approval is deferred beyond
      the MVP (owner decision 2026-11-25, MVP-13).
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
