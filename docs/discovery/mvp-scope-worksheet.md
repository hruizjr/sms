# MVP Scope Worksheet

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — proposed tiers; **owner decisions recorded** for MVP-08 and MVP-09 (2026-11-25) | Project owner | 2026-11-25 |

This worksheet helps the owner decide what launches first (D-089, D-020). Every capability from the
original product expectations (D-007) is listed and classified:

1. **Essential** — required for the initial MVP.
2. **Deferrable** — important, but can follow shortly after launch.
3. **Future** — not needed for the initial platform.

The "Proposed tier" column is a **recommendation** based on a prepaid, admin-onboarded, Philippines-only
MVP serving a small set of launch customers. It is not a decision. Launch-customer answers
([customer questionnaire](questionnaires/enterprise-customer.md) CU-10, CU-11) and telco answers may change
it. The owner records the final tier in the "Owner decision" column.

## Summary

| ID | Capability | Primary user | Proposed tier | Owner decision |
| -- | ---------- | ------------ | ------------- | -------------- |
| MVP-01 | Client account (tenant) creation by admin | Admin | Essential | |
| MVP-02 | Self-service registration with document upload (KYC) | Prospective client | Deferrable | |
| MVP-03 | Client users and basic roles | Client admin | Essential | |
| MVP-04 | Admin users, roles, and mandatory MFA | Our staff | Essential | |
| MVP-05 | Sender ID request and approval workflow | Client, admin | Essential | |
| MVP-06 | API keys (create, scope, revoke) | Developer | Essential | |
| MVP-07 | Send a single message via public API (with idempotency key) | Developer | Essential | |
| MVP-08 | Send to many recipients via API (batch) | Developer | Essential* | **Deferred to a later release** — not in the MVP (owner, 2026-11-25; D-094e) |
| MVP-09 | Send from the portal (type/paste recipients or upload a recipient file) | Client user | Essential | **MVP: single-message sending only.** Bulk sending (many recipients, including an uploaded recipient file) **deferred to a later release** (owner, 2026-11-25) |
| MVP-10 | Message composer with segment count and cost estimate | Client user | Essential | |
| MVP-11 | Contact book, groups, and segmentation | Client user | Deferrable | |
| MVP-12 | Campaign scheduling | Client user | Deferrable* | |
| MVP-13 | Campaign approval by admin (policy-based) | Admin | Essential | |
| MVP-14 | Opt-out / suppression enforcement | System, admin | Essential | |
| MVP-15 | Inbound messages (replies, keywords) | Client | Deferrable | |
| MVP-16 | Per-message delivery status and history | Client user, developer | Essential | |
| MVP-17 | Delivery report export (CSV) | Client user | Essential | |
| MVP-18 | Webhooks for status updates | Developer | Deferrable* | |
| MVP-19 | Analytics dashboards | Client user | Deferrable | |
| MVP-20 | Prepaid credit balance, admin top-up, credit ledger | Client, admin | Essential | |
| MVP-21 | Client-specific pricing configuration | Admin | Essential | |
| MVP-22 | Transaction history and statements | Client, admin | Essential | |
| MVP-23 | Postpaid accounts and invoicing | Admin | Deferrable | |
| MVP-24 | Online payments | Client | Future | |
| MVP-25 | Provider configuration and basic routing | Admin | Essential | |
| MVP-26 | Automatic failover routing between telcos | System | Future | |
| MVP-27 | Telco usage reconciliation report | Admin/finance | Essential | |
| MVP-28 | Operations monitoring (queues, binds, failures) | Ops | Essential | |
| MVP-29 | Audit log | Admin, auditor | Essential | |
| MVP-30 | Abuse controls: velocity limits, alerts, kill switch | System, admin | Essential | |
| MVP-31 | Sub-accounts and reseller hierarchy | Client, reseller | Future | |
| MVP-32 | Public API documentation | Developer | Essential | |
| MVP-33 | Telco simulator (development and demos) | Engineering | Essential | |
| MVP-34 | Data retention and expiry | System | Essential before production | |

\* Tier depends strongly on launch-customer answers (see each item's open questions).

## Capability details

### MVP-01 — Client account creation by admin
- **Problem:** clients need an isolated account before sending.
- **Capability:** admin creates a tenant with company details, status, and pricing.
- **Value:** controlled onboarding while volumes are small; strong abuse control.
- **Dependencies:** D-085 (KYC policy), D-161 (contracting entity).
- **Security:** tenant isolation from day one; audited creation.
- **Acceptance criteria:** admin can create, suspend, and reactivate a tenant; all actions audited.
- **Open questions:** what documents are required before activation (D-085, D-130)?

### MVP-02 — Self-service registration with KYC
- **Problem:** manual onboarding does not scale.
- **Capability:** prospects register and upload documents for review.
- **Value:** growth beyond launch customers.
- **Dependencies:** D-085, D-130, document storage design.
- **Security:** fraudulent sign-ups are a key abuse vector; document storage holds sensitive data.
- **Acceptance criteria:** registration creates a pending tenant that cannot send until approved.
- **Open questions:** is self-service expected at launch by the business plan (D-163)?

### MVP-03 — Client users and basic roles
- **Problem:** several people in a client organization use the portal.
- **Capability:** invite users; roles such as owner, sender, viewer, billing.
- **Value:** clients control who can send and spend.
- **Dependencies:** RBAC design (Phase 2B), D-064.
- **Security:** least privilege; MFA availability for clients.
- **Acceptance criteria:** a viewer cannot send; a sender cannot change billing; role changes audited.
- **Open questions:** which roles do launch customers need?

### MVP-04 — Admin users, roles, and mandatory MFA
- **Problem:** our staff hold powerful permissions (credits, pricing, routing).
- **Capability:** admin accounts with separated roles and mandatory MFA (D-021).
- **Value:** reduces insider and account-takeover risk.
- **Dependencies:** D-021, D-091 (separate admin portal).
- **Security:** highest-risk accounts in the system.
- **Acceptance criteria:** no admin action without MFA; role-restricted actions enforced server-side.
- **Open questions:** how many staff and which roles (D-078)?

### MVP-05 — Sender ID request and approval workflow
- **Problem:** branded sender IDs must be registered with telcos and protected from impersonation.
- **Capability:** client requests a sender ID with supporting documents; admin approves; per-telco
  registration status tracked.
- **Value:** required for branded messages; key abuse control (D-022).
- **Dependencies:** D-103 (telco process), D-123, D-128.
- **Security:** impersonation of banks, government, and brands.
- **Acceptance criteria:** messages cannot use unapproved sender IDs; decisions audited.
- **Open questions:** can we register on the client's behalf with each telco?

### MVP-06 — API keys
- **Problem:** developers need credentials to call the API.
- **Capability:** create, label, scope, and revoke keys; shown once; stored hashed.
- **Value:** enables integration.
- **Dependencies:** security baseline.
- **Security:** keys allow spending; leaks must be containable.
- **Acceptance criteria:** revoked keys stop working immediately; key events audited.
- **Open questions:** do customers require IP allowlisting per key (CU-22)?

### MVP-07 — Single message via public API
- **Problem:** OTPs and alerts are sent one at a time by client systems.
- **Capability:** submit a message with an idempotency key; receive a message ID.
- **Value:** core revenue path for API customers.
- **Dependencies:** ADR-0018 / D-094, D-023, ADR-0020, ADR-0021.
- **Security:** authentication, rate limits, input validation.
- **Acceptance criteria:** retries with the same key create one message and one reservation (scenarios IK-1–IK-11).
- **Open questions:** is the idempotency key required (D-094a)?

### MVP-08 — Batch send via API
- **Problem:** clients send the same or personalized message to many recipients.
- **Capability:** one request with many recipients; per-recipient results.
- **Value:** needed by API customers with campaigns.
- **Dependencies:** D-094e (bulk semantics), D-062.
- **Security:** large batches amplify abuse; approval policy applies (MVP-13).
- **Acceptance criteria:** partial rejections reported per recipient; no duplicate charges on retry.
- **Open questions:** do launch customers send campaigns via API or portal (CU-10)?
- **Owner decision (2026-11-25, PR #7 review):** bulk sending is excluded from the MVP (D-094e). The public API v1 contract has no bulk endpoint.
- **Owner decision (2026-11-25, final PR #7 review):** bulk sending through the API is **deferred to a later release**. The MVP supports single-message sending only (MVP-07).

### MVP-09 — Send from the portal
- **Problem:** non-technical clients need to send without integration.
- **Capability:** enter recipients or upload a recipient file for a single send.
- **Value:** serves clients without developers.
- **Dependencies:** D-071, file import limits.
- **Security:** file upload validation; size and row limits.
- **Acceptance criteria:** invalid numbers reported before sending; confirmation shows recipients, segments, cost.
- **Open questions:** maximum file size needed?
- **Owner decision (2026-11-25, final PR #7 review):** in the MVP, the portal supports **single-message sending** (one recipient per message). Bulk sending through the portal — many recipients, including an uploaded recipient file — is **deferred to a later release**, together with API bulk sending (MVP-08). The capability, acceptance criteria, and file-import items above describe the deferred bulk scope and are kept as history.

### MVP-10 — Composer with segment and cost estimate
- **Problem:** encoding surprises (e.g. emoji) can double cost (D-029).
- **Capability:** live segment count, encoding indicator, estimated cost before sending.
- **Value:** trust and fewer disputes.
- **Dependencies:** D-105 (telco encoding), ADR-0019 (money), D-093 (rounding).
- **Security:** none specific.
- **Acceptance criteria:** estimate equals the reservation for the same input (scenario FI-6).
- **Open questions:** do telcos count segments as we do (D-117)?

### MVP-11 — Contact book, groups, segmentation
- **Problem:** repeat senders manage audiences.
- **Capability:** stored contacts, groups, attributes, filters.
- **Value:** convenience and retention of marketing clients.
- **Dependencies:** D-071, D-073, privacy role (D-121).
- **Security:** large store of personal data; export controls.
- **Acceptance criteria:** tenant-isolated; import validation; export audited.
- **Open questions:** do launch customers keep contacts in their own systems instead?

### MVP-12 — Campaign scheduling
- **Problem:** campaigns go out at planned times.
- **Capability:** schedule a send; edit/cancel before release.
- **Value:** expected by marketing clients.
- **Dependencies:** D-122 (time-of-day rules may apply), approval policy.
- **Security:** scheduled sends must re-check balance, suppression, and approval at release time.
- **Acceptance criteria:** cancelled campaigns never send; release respects any time restrictions.
- **Open questions:** do launch customers need scheduling on day one?

### MVP-13 — Campaign approval
- **Problem:** large or first-time campaigns are the main abuse vector (D-025, D-087).
- **Capability:** policy decides which campaigns need admin approval.
- **Value:** protects telco relationships.
- **Dependencies:** D-087, D-078 (staffing).
- **Security:** approval actions audited.
- **Acceptance criteria:** campaigns needing approval cannot be released without it.
- **Open questions:** thresholds; turnaround time customers will accept.

### MVP-14 — Opt-out / suppression enforcement
- **Problem:** recipients who opted out must not be messaged (D-024).
- **Capability:** tenant and platform suppression lists checked before dispatch.
- **Value:** compliance and trust.
- **Dependencies:** D-122 (legal requirements), D-063/MVP-15 for keyword opt-outs.
- **Security:** suppression lists contain personal data.
- **Acceptance criteria:** a suppressed number is never sent to; suppression changes audited.
- **Open questions:** which opt-out keywords and records are legally required (LR-08, D-122)?

### MVP-15 — Inbound messages
- **Problem:** replies and keyword opt-outs require inbound handling.
- **Capability:** receive MO messages and route them to the right client.
- **Value:** two-way use cases; automated opt-out.
- **Dependencies:** D-063, telco MO support (checklist §12).
- **Security:** inbound content is personal data.
- **Acceptance criteria:** STOP-type replies update suppression automatically.
- **Open questions:** do launch customers need replies at all?

### MVP-16 — Delivery status and history
- **Problem:** clients must know what happened to each message.
- **Capability:** per-message status timeline using the proposed status model.
- **Value:** core product promise.
- **Dependencies:** ADR-0021, D-096, D-099, D-104.
- **Security:** tenant isolation; masked display where appropriate.
- **Acceptance criteria:** uncertain outcomes are shown as uncertain, not as failures (scenarios DR-1–DR-10).
- **Open questions:** status labels shown to customers (D-099).

### MVP-17 — Delivery report export
- **Problem:** clients reconcile and report internally.
- **Capability:** CSV export of message statuses for a period.
- **Value:** expected by enterprises.
- **Dependencies:** D-072, D-073.
- **Security:** formula-injection protection; export audited.
- **Acceptance criteria:** export matches the status history; large exports run in the background.
- **Open questions:** history length required.

### MVP-18 — Webhooks
- **Problem:** API clients want push notifications instead of polling.
- **Capability:** signed, at-least-once status callbacks with event IDs.
- **Value:** strongly expected by many API integrators.
- **Dependencies:** D-099, messaging reliability §2.4.
- **Security:** SSRF protection; HMAC signatures.
- **Acceptance criteria:** duplicate deliveries carry the same event ID; failed callbacks retried with backoff.
- **Open questions:** do launch API customers require webhooks on day one (CU-21)?

### MVP-19 — Analytics dashboards
- **Problem:** clients want trends and delivery rates.
- **Capability:** charts by period, network, campaign.
- **Value:** retention; upsell.
- **Dependencies:** D-072, ADR-0008.
- **Security:** tenant isolation in aggregates.
- **Acceptance criteria:** figures reconcile with exports.
- **Open questions:** which metrics matter to launch customers?

### MVP-20 — Prepaid credit, admin top-up, ledger
- **Problem:** we must not deliver traffic we cannot bill.
- **Capability:** balance, reservations, settlement, admin top-up with audit.
- **Value:** core financial control.
- **Dependencies:** ADR-0011, ADR-0019, D-080–D-083, D-092, D-093.
- **Security:** financial integrity; admin adjustments require reason and audit.
- **Acceptance criteria:** no overspend under concurrency; balance equals ledger (scenarios FI-1–FI-10).
- **Open questions:** credit unit (D-081); refund policy (D-083).

### MVP-21 — Client-specific pricing
- **Problem:** clients negotiate different rates.
- **Capability:** price per segment per client, possibly per network or traffic type.
- **Value:** commercial flexibility.
- **Dependencies:** D-084, D-092.
- **Security:** pricing changes audited; versioned.
- **Acceptance criteria:** each charge references the rate version in effect.
- **Open questions:** pricing dimensions (D-084).

### MVP-22 — Transaction history and statements
- **Problem:** clients and finance need records of top-ups and charges.
- **Capability:** ledger-based history and periodic statements.
- **Value:** trust; accounting.
- **Dependencies:** D-127 (invoicing rules).
- **Security:** tenant isolation.
- **Acceptance criteria:** statements reconcile to the ledger.
- **Open questions:** official invoice/receipt requirements (D-127).

### MVP-23 — Postpaid accounts and invoicing
- **Problem:** large enterprises and government may require monthly invoices.
- **Capability:** credit limits, invoice generation, collections tracking.
- **Value:** access to larger customers.
- **Dependencies:** D-082, D-127.
- **Security:** credit risk controls.
- **Acceptance criteria:** sending stops at the credit limit.
- **Open questions:** do launch customers require postpaid (CU-30)?

### MVP-24 — Online payments
- **Problem:** manual top-ups are slow.
- **Capability:** payment gateway integration.
- **Value:** self-service growth.
- **Dependencies:** D-065, payment provider choice.
- **Security:** payment data handling.
- **Acceptance criteria:** each confirmed payment produces one ledger credit, even if the payment
  provider notifies us more than once (idempotent on the payment reference).
- **Open questions:** which payment methods?

### MVP-25 — Provider configuration and basic routing
- **Problem:** traffic must go to the right telco connection.
- **Capability:** admin-configured routes per destination network with rate limits.
- **Value:** required to send at all.
- **Dependencies:** D-070 (MNP), D-107, D-101, ADR-0012.
- **Security:** routing changes audited; restricted role.
- **Acceptance criteria:** rate limits never exceed configured telco limits.
- **Open questions:** how the destination network is determined (D-070).

### MVP-26 — Automatic failover routing
- **Problem:** a telco outage stops delivery.
- **Capability:** automatic rerouting when allowed.
- **Value:** resilience.
- **Dependencies:** D-119 (telco permission), D-107.
- **Security:** must not violate telco terms.
- **Acceptance criteria:** failover only along approved routes.
- **Open questions:** is failover contractually allowed?

### MVP-27 — Telco usage reconciliation report
- **Problem:** telco invoices must match our records (R-08, R-32).
- **Capability:** compare our submissions and receipts with telco CDRs; list discrepancies.
- **Value:** protects margin.
- **Dependencies:** D-108 (CDR access), D-117.
- **Security:** finance data access restricted.
- **Acceptance criteria:** every discrepancy is reported; nothing auto-corrected.
- **Open questions:** CDR format and frequency.

### MVP-28 — Operations monitoring
- **Problem:** failures must be noticed quickly (R-40).
- **Capability:** dashboards and alerts for queues, connections, error rates, receipt delays.
- **Value:** reliability.
- **Dependencies:** D-066, ADR-0016.
- **Security:** monitoring must not expose personal data.
- **Acceptance criteria:** alerts fire for lost telco connections and growing queue age.
- **Open questions:** who receives alerts (D-066)?

### MVP-29 — Audit log
- **Problem:** security and financial actions must be traceable.
- **Capability:** append-only audit records viewable by authorized staff.
- **Value:** investigations, disputes, regulators.
- **Dependencies:** security baseline §9.
- **Security:** tamper evidence; restricted access.
- **Acceptance criteria:** every listed sensitive action produces an audit record.
- **Open questions:** retention period (D-124).

### MVP-30 — Abuse controls
- **Problem:** smishing through our platform could end telco relationships (R-20).
- **Capability:** per-tenant velocity limits, anomaly alerts, immediate suspension of tenant/sender ID/campaign.
- **Value:** protects the business.
- **Dependencies:** D-086, D-087, D-110, D-123.
- **Security:** central to the threat model.
- **Acceptance criteria:** suspension takes effect immediately and is audited.
- **Open questions:** content/URL rules required by telcos (D-110).

### MVP-31 — Sub-accounts and reseller hierarchy
- **Problem:** BPOs and resellers serve their own clients.
- **Capability:** nested accounts with delegated billing.
- **Value:** wholesale revenue.
- **Dependencies:** D-064, D-088, D-113.
- **Security:** complex isolation and billing boundaries.
- **Acceptance criteria:** to be defined if adopted.
- **Open questions:** do telcos and counsel permit it?

### MVP-32 — Public API documentation
- **Problem:** developers need clear docs to integrate.
- **Capability:** documentation generated from the OpenAPI contract.
- **Value:** faster integration.
- **Dependencies:** ADR-0015, ADR-0018.
- **Security:** no secrets in examples.
- **Acceptance criteria:** documentation matches the contract.
- **Open questions:** none.

### MVP-33 — Telco simulator
- **Problem:** development cannot wait for telco access.
- **Capability:** configurable simulated provider.
- **Value:** testing and demonstrations without real SMS.
- **Dependencies:** D-026, ADR-0012.
- **Security:** must never deliver to real numbers.
- **Acceptance criteria:** provider contract suite passes against it.
- **Open questions:** none.

### MVP-34 — Data retention and expiry
- **Problem:** personal data must not be kept longer than necessary (LR-09).
- **Capability:** retention classes and automated expiry.
- **Value:** compliance; storage cost.
- **Dependencies:** D-124, D-073.
- **Security:** deletion must respect legal holds.
- **Acceptance criteria:** data older than its retention period is removed or anonymized.
- **Open questions:** retention periods (counsel).
