# Phase 2 Exit Checklist

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — three rows satisfied (EX-13, 2026-11-25; EX-1, 2026-11-26, amended criterion; EX-3, 2026-11-26, simulator-only decision) and EX-7 deferred; Phase 2 not complete | Project owner | 2026-11-26 |

This checklist maps each Phase 2 acceptance criterion and work item in the
[development roadmap](../roadmap/development-roadmap.md) to the evidence or owner approval needed. The
roadmap remains the authority for the criteria; this file tracks progress against them. **Phase 2 is not
complete** until every row is satisfied with recorded evidence (see the [evidence register](evidence-register.md))
or an explicit, recorded owner decision.

## 1. Acceptance criteria

| # | Roadmap criterion | Evidence or approval required | Register items | Where prepared | Status | Evidence IDs |
| - | ----------------- | ----------------------------- | -------------- | -------------- | ------ | ------------ |
| EX-1 | MVP scope approved in writing by the owner. | Owner decision column completed in the [MVP scope worksheet](mvp-scope-worksheet.md) and recorded approval (e.g. merged PR or signed note). ~~Launch customers identified.~~ **Amended by the owner 2026-11-26:** launch customers may remain TBD until pilot discovery; at least one validated pilot customer is required before production onboarding or traffic (gate PG-8; D-061 stays open with this trigger). | D-089, D-020, D-061, D-007 | MVP worksheet; [customer questionnaire](questionnaires/enterprise-customer.md) | **Satisfied (amended criterion)** — all 34 capabilities decided and the scope approved by the owner on 2026-11-26 (21 MVP, 8 Post-MVP, 5 Deferred; production gates PG-1 to PG-8), recorded in the [MVP scope worksheet](mvp-scope-worksheet.md#owner-approval-of-the-mvp-scope-2026-11-26) and [decision log §14](../engineering/decision-log.md#14-owner-decisions--mvp-scope-phase-2-ex-1-2026-11-26); written approval is the merge of the pull request that records it. Launch customers TBD (D-061) | — (owner decision; no external evidence) |
| EX-2 | Traffic figures for launch customers recorded with sources, **or** an explicit decision to proceed with provisional targets. | [Traffic evidence worksheet](traffic-evidence-worksheet.md) completed with evidence IDs; unit of the 5-billion figure established; **or** a recorded owner decision naming the provisional targets and their basis. | D-004, D-060, D-061, D-062, D-074, D-075 | Traffic worksheet; [partner](questionnaires/business-partner.md) and customer questionnaires | Not started | — |
| EX-3 | At least one telco's technical specification received, **or** an explicit decision to proceed simulator-only. | Official specification logged as evidence (stored outside Git); **or** a recorded owner decision to proceed simulator-only, with the risk acknowledged. | D-100, D-101, D-104–D-106, D-114, D-115 | [Telco questionnaires](questionnaires/telco-core.md); [telco checklist](../integrations/telco-discovery-checklist.md) | **Satisfied (simulator-only decision)** — the owner decided on 2026-11-26 to proceed simulator-only without a telco specification, with the risks acknowledged (decisions S1–S8, gate PG-9, [decision log §15](../engineering/decision-log.md#15-owner-decisions--simulator-only-development-phase-2-ex-3-2026-11-26)). No telco specification has been received; every telco question stays unanswered | — (owner decision; no external evidence) |
| EX-4 | Regulatory path documented by counsel (even if approvals are pending). | Counsel's written opinion covering classification and registrations (LC-01–LC-03), privacy role and registration (LC-20, LC-21); logged as evidence (restricted, outside Git). | D-120, D-121, D-113, D-160 | [Legal counsel questionnaire](questionnaires/legal-counsel.md); [research log](legal-regulatory-research.md) | Not started | — |
| EX-5 | Architecture document and threat model approved. | Phase 2B deliverables written and approved by the owner. | D-142 and Phase 2B work | Not yet prepared (Phase 2B) | Not started | — |

## 2. Roadmap work items

| # | Work item | Evidence or approval required | Register items / ADRs | Status |
| - | --------- | ----------------------------- | --------------------- | ------ |
| EX-6 | Collect traffic data | As EX-2 | D-004, D-060–D-062 | Not started |
| EX-7 | Collect telco technical answers | Telco questionnaire answers logged per telco. **Amended by the owner 2026-11-26:** telco technical discovery is formally deferred during simulator-only development (EX-3); unanswered questions remain production prerequisites (gates PG-1, PG-9) | D-100–D-119 | **Deferred (owner decision 2026-11-26)** — no telco answers have been collected and none is marked collected ([decision log §15](../engineering/decision-log.md#15-owner-decisions--simulator-only-development-phase-2-ex-3-2026-11-26)) |
| EX-8 | Obtain regulatory guidance | As EX-4 | D-120–D-131 | Not started |
| EX-9 | Business decisions: billing basis, credit model, MVP scope, launch customers | Recorded owner decisions | D-080–D-084, D-089, D-160–D-163 | In progress — **MVP scope decided** (2026-11-25 and 2026-11-26; see EX-1). Payment model and pricing structure partly decided for the MVP only: prepaid only (D-082) and one flat per-segment rate per client (D-084). Still open: D-080, D-081, D-083 (charge point, credit unit, refunds), D-160–D-163 (business model), launch customers (D-061, TBD until pilot discovery) |
| EX-10 | Write MVP SRS, domain model, system architecture, threat model, RBAC design, data classification and retention schedule (draft) | Documents approved by owner | — (Phase 2B) | Not started |
| EX-11 | Decide ADRs 0001–0021 | Owner records Accepted / Rejected / amended for each; see [decision readiness](decision-readiness.md) | D-142 | In progress — accepted on 2026-11-25: 0002–0005, 0013–0018, and 0021 (limited scope); accepted on 2026-11-26: 0010 (tenant-owned control-plane tables) and 0012 (limited scope: provisional interface, capability declarations, adapter abstraction, simulator); still Proposed: 0001, 0006–0009, 0011, 0019, 0020; see the [decision log index](../engineering/decision-log.md#4-index) |
| EX-12 | Decide monetary representation and rounding | ADR-0019 decided; D-092, D-093 resolved | ADR-0019 | Not started |
| EX-13 | Decide idempotency details | D-094 resolved; ADR-0018 decided | ADR-0018 | **Satisfied** — D-094a–e decided and ADR-0018 accepted by the owner, 2026-11-25 ([decision log §8](../engineering/decision-log.md#8-owner-decisions--public-api-contract-2026-11-25)) |
| EX-14 | Decide uncertain-outcome and receipt policies | D-095–D-099 resolved; ADR-0021 decided | ADR-0021 | In progress — ADR-0021 accepted with a limited scope and customer labels decided (2026-11-25); D-095, D-096 (billing), D-097, D-098, D-099 (webhooks) open. **2026-11-26:** D-095 interim rule — uncertain submissions remain uncertain and are never automatically resubmitted or failed |

## 3. Readiness summary

| Question | Answer today |
| -------- | ------------ |
| Is Phase 2A preparation done? | Yes — merged (PR #2) |
| Has external discovery started? | No evidence recorded; telco technical discovery formally deferred during simulator-only development (EX-7) |
| Can Phase 2 exit? | **No** — EX-1 (amended), EX-3 (simulator-only) and EX-13 are satisfied and EX-7 is deferred; EX-2, EX-4 to EX-6, EX-8 to EX-12 and EX-14 are not |
