# Risk Register

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — initial assessment | Project owner | 2026-11-24 |

## Scales

- **Impact:** High (threatens the business, causes major financial/legal harm, or harms recipients) ·
  Medium (significant delay, cost, or customer impact) · Low (contained, recoverable).
- **Likelihood:** High (expected without action) · Medium (plausible) · Low (unlikely).
- **Status:** Open · Mitigating · Accepted · Closed.

All risks below are **Open** at the time of writing; no mitigation has been implemented yet.

## Regulatory and commercial

| ID | Risk | Impact | Likelihood | Mitigation | Status |
| -- | ---- | ------ | ---------- | ---------- | ------ |
| R-01 | Required regulatory authorizations (NTC or other) are not obtained, delayed, or impose conditions that affect the design. | High | Medium | Engage Philippine telecom counsel early; track in D-120; avoid production commitments until confirmed; design for configurable compliance controls. | Open |
| R-02 | Telco agreements are not concluded, or terms (pricing, TPS, deposits, minimum commitments) make the business unviable. | High | Medium | Obtain term sheets early; model unit economics before heavy build; build with simulator so engineering is not blocked. | Open |
| R-03 | Telcos restrict aggregators from reselling, impose exclusivity, or classify traffic differently than expected. | High | Medium | Telco checklist §19; legal review of agreements. | Open |
| R-04 | The 5-billion monthly figure is overstated, mis-specified, or not committed, leading to over-investment. | High | High | Traffic discovery document; size only from verified data; scale incrementally. | Open |
| R-05 | Traffic materializes faster than telco capacity or platform readiness, causing failed commitments. | High | Low–Medium | Capacity negotiation tied to verified forecasts; customer onboarding gated by tested capacity. | Open |
| R-06 | Dependency on one or two telcos: an outage, policy change, or contract dispute stops service. | High | Medium | Multiple telco routes where agreements allow; provider abstraction; failover policy. | Open |
| R-07 | Messages to DITO subscribers (or ported numbers) cannot be delivered without an appropriate route. | Medium | Medium | Clarify off-net acceptance and DITO route (D-069, D-107); MNP strategy (D-070). | Open |
| R-08 | Margin loss: telco charges per submitted segment while customers are charged per delivered message (or encoding differences). | High | Medium | Align billing basis with telco terms (D-080); segment calculation shared between estimate and charge; reconciliation. | Open |
| R-09 | Customer credit risk if postpaid accounts are offered. | Medium | Medium | Prepaid by default; credit limits and approvals for postpaid. | Open |
| R-10 | Government customer requirements (procurement, accreditation, data residency) not met. | Medium | Medium | Discovery item D-125; hosting decision considers them. | Open |

## Security and compliance

| ID | Risk | Impact | Likelihood | Mitigation | Status |
| -- | ---- | ------ | ---------- | ---------- | ------ |
| R-20 | Platform used for smishing or scams (malicious customers or compromised accounts), leading to harm, telco suspension, regulatory action, and reputational damage. | High | High | KYC; sender ID vetting; campaign approval; content and URL policies; velocity limits; kill switch; monitoring. | Open |
| R-21 | Cross-tenant data leakage. | High | Medium | Tenant isolation in depth (ADR-0010); mandatory cross-tenant tests. | Open |
| R-22 | Personal data breach (phone numbers, message content, OTPs). | High | Medium | Data minimization; masking in logs; encryption; access control; retention limits; breach response plan. | Open |
| R-23 | Admin or API key account takeover. | High | Medium | MFA for admins; hashed, scoped, revocable API keys; anomaly alerts; audit logs. | Open |
| R-24 | Telco credentials leaked or misused. | High | Low–Medium | Secret manager; restricted access; rotation; network restrictions. | Open |
| R-25 | Non-compliance with Data Privacy Act obligations (registration, DPO, consent, rights). | High | Medium | Privacy counsel; privacy impact assessment; DPA-aligned processes (S-5). | Open |
| R-26 | Supply-chain compromise via dependencies or CI. | Medium | Low–Medium | Minimal dependencies with approval; lockfiles; vulnerability scanning; restricted CI secrets. | Open |

## Technical and scalability

| ID | Risk | Impact | Likelihood | Mitigation | Status |
| -- | ---- | ------ | ---------- | ---------- | ------ |
| R-30 | Duplicate SMS sent to recipients due to retries after ambiguous telco responses. | Medium | Medium | Explicit ambiguous-outcome policy per provider; idempotency; no blind retries. | Open |
| R-31 | Lost messages (accepted but never sent) due to crashes or queue errors. | High | Medium | Transactional outbox; at-least-once processing; reconciliation of stuck messages; alerting on queue age. | Open |
| R-32 | Billing inconsistency: double charges, missed charges, overspending under concurrency. | High | Medium | Append-only ledger; DB-level concurrency control; idempotent ledger entries; reconciliation. | Open |
| R-33 | Missing or delayed delivery receipts make status and billing uncertain. | Medium | High | Expiry state; unmatched DLR store; reconciliation with telco CDRs. | Open |
| R-34 | Premature complexity (microservices, Kafka, ClickHouse too early) slows delivery and raises operational cost. | Medium | Medium | ADRs 0003, 0006–0008: defer until justified. | Open |
| R-35 | Under-design: architecture cannot scale when verified traffic arrives. | High | Low–Medium | Stateless processes, partitionable data, interfaces around queue/storage, early load-test capability. | Open |
| R-36 | Telco protocol behaviour differs from assumptions (SMPP variants, DLR formats, encodings). | Medium | High | Never invent specs; adapters per telco; contract suite; telco test environment before production. | Open |
| R-37 | Limited telco binds create single points of failure in connector processes. | Medium | Medium | Active/standby design (AQ-3); reconnect logic; monitoring of bind status. | Open |
| R-38 | PostgreSQL write volume or table growth becomes a bottleneck. | Medium | Medium | Partitioning, batching, retention; load tests; broker adoption when justified. | Open |
| R-39 | Encoding/segmentation errors cause garbled messages or unexpected costs. | Medium | Medium | Exhaustively tested segmentation; cost preview; telco-confirmed encoding rules. | Open |
| R-60 | Monetary precision or rounding errors (e.g. storing centavos when rates need more precision) cause material discrepancies at volume. | High | Medium | Exact arithmetic only; representation decided from real rates (ADR-0019); one shared rounding function; reconciliation. | Open |
| R-61 | Duplicate event publication or consumption causes duplicate side effects (charges, refunds, webhooks). | High | Medium | At-least-once with idempotent consumers (ADR-0020); unique keys on side effects; test scenarios OB/DR. | Open |
| R-62 | Treating timeouts or missing receipts as failures causes wrong refunds, misleading reports, and customer resends that duplicate SMS. | Medium | High | Uncertain states (`SUBMIT_UNCERTAIN`, `UNKNOWN`); late-receipt reconciliation (ADR-0021). | Open |
| R-63 | Row-Level Security misconfigured (session-level context on pooled connections, owner/BYPASSRLS role) leaks tenant data or gives false assurance. | High | Low–Medium | Evaluation before adoption; transaction-scoped context; non-owner role; fail-closed policies; leak tests (ADR-0010). | Open |

## Operational

| ID | Risk | Impact | Likelihood | Mitigation | Status |
| -- | ---- | ------ | ---------- | ---------- | ------ |
| R-40 | No 24/7 operations capability: incidents (e.g. bind drops at night) go unnoticed, OTPs fail. | High | High | Define operations model (D-066); alerting; runbooks; on-call before production. | Open |
| R-41 | Key-person dependency on a single owner for decisions and reviews. | High | High | Documentation-first process; decision log; consider a second technical reviewer before production. | Open |
| R-42 | Infrastructure cost at scale exceeds margins. | Medium | Medium | Cost modelling from verified traffic; prefer simple, efficient components. | Open |
| R-43 | Insufficient backup/disaster recovery leads to data loss (including financial records). | High | Low–Medium | Define RPO/RTO (AQ-8); encrypted, tested backups. | Open |
| R-44 | Hosting choice incompatible with telco connectivity requirements (static IPs, VPN, cross-connects). | Medium | Medium | Obtain telco connectivity requirements before choosing hosting (AQ-7). | Open |

## AI-assisted development

| ID | Risk | Impact | Likelihood | Mitigation | Status |
| -- | ---- | ------ | ---------- | ---------- | ------ |
| R-50 | AI agents produce inconsistent code or architecture drift across tasks. | Medium | High | `AGENTS.md`; ADRs; small tasks; review checklist; linters. | Open |
| R-51 | AI agents fabricate telco specifications, regulatory facts, or test results. | High | Medium | Explicit prohibition; evidence requirements; owner verification of claims; CI as independent evidence. | Open |
| R-52 | Parallel agents create conflicting changes in shared files. | Medium | Medium | File-scope declaration; serialized hot files; contracts-first sequencing. | Open |
| R-53 | Review bottleneck: owner cannot review the volume of agent output thoroughly. | High | High | Small PRs; strong automated checks; optional reviewing agents; prioritize review effort on billing, security, data model. | Open |
| R-54 | Subtle security or concurrency bugs pass review because the owner is not a specialist. | High | Medium | Mandatory tests for critical areas; independent security review/pentest before production; consider expert consultation for ledger and SMPP design. | Open |

## Review cadence

Review this register at the end of every phase and whenever a discovery item is resolved.
