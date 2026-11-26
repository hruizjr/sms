# MVP Scope Worksheet

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| **Owner-approved (2026-11-26)** — all 34 capabilities decided (21 MVP, 8 Post-MVP, 5 Deferred) with production gates PG-1 to PG-8 (PG-9 added 2026-11-26 with the EX-3 simulator-only decision); launch customers TBD until pilot discovery (D-061, EX-1 amended). Earlier decisions: MVP-01, 05, 08, 09, 13, 29, 30 (2026-11-25); MVP-11, 12 (2026-11-26) | Project owner | 2026-11-26 |

This worksheet helps the owner decide what launches first (D-089, D-020). Every capability from the
original product expectations (D-007) is listed and classified:

1. **Essential** — required for the initial MVP.
2. **Deferrable** — important, but can follow shortly after launch.
3. **Future** — not needed for the initial platform.

The "Proposed tier" column is a **recommendation** based on a prepaid, admin-onboarded, Philippines-only
MVP serving a small set of launch customers. It is not a decision. Launch-customer answers
([customer questionnaire](questionnaires/enterprise-customer.md) CU-10, CU-11) and telco answers may change
it. The owner records the final tier in the "Owner decision" column.

**Owner decisions use four labels** (2026-11-26): **MVP** (= Essential), **Post-MVP** (= Deferrable, shortly
after launch), **Deferred** (= Future, not part of the initial platform) and **Out of scope** (not planned).
The approved scope, the production gates and the decisions that remain open are in
[Owner approval of the MVP scope](#owner-approval-of-the-mvp-scope-2026-11-26) below.

## Summary

| ID | Capability | Primary user | Proposed tier | Owner decision |
| -- | ---------- | ------------ | ------------- | -------------- |
| MVP-01 | Client account (tenant) creation by admin | Admin | Essential | **Essential** — admin-controlled tenant onboarding and activation (owner, 2026-11-25) |
| MVP-02 | Self-service registration with document upload (KYC) | Prospective client | Deferrable | **Post-MVP** — onboarding stays staff-controlled (MVP-01); self-service registration and document upload deferred (owner, 2026-11-26) |
| MVP-03 | Client users and basic roles | Client admin | Essential | **MVP (minimal)** — three client roles: Owner (manages users and API keys), Sender (sending functions), Viewer (read-only); a separate Billing role deferred (owner, 2026-11-26) |
| MVP-04 | Admin users, roles, and mandatory MFA | Our staff | Essential | **MVP** — internal roles Super Admin, Operations, Finance; MFA mandatory for all human internal admin accounts; least privilege (owner, 2026-11-26) |
| MVP-05 | Sender ID request and approval workflow | Client, admin | Essential | **Essential** — sender ID approval before use (owner, 2026-11-25) |
| MVP-06 | API keys (create, scope, revoke) | Developer | Essential | **MVP (core)** — create, label, show once, store hashed, revoke; scopes `messages:send` and `messages:read`; IP allowlisting and advanced scope management deferred (owner, 2026-11-26) |
| MVP-07 | Send a single message via public API (with idempotency key) | Developer | Essential | **MVP** — single-message public API with a mandatory idempotency key; no promise of exactly-once provider delivery (owner, 2026-11-26) |
| MVP-08 | Send to many recipients via API (batch) | Developer | Essential* | **Deferred to a later release** — not in the MVP (owner, 2026-11-25; D-094e) |
| MVP-09 | Send from the portal (type/paste recipients or upload a recipient file) | Client user | Essential | **MVP: single-message sending only** — exactly one recipient per send operation; no pasted recipient lists, recipient-file uploads, contact-group sends, or multi-recipient selection. Bulk sending **deferred to a later release** (owner, 2026-11-25) |
| MVP-10 | Message composer with segment count and cost estimate | Client user | Essential | **MVP (full)** — encoding, segment count and estimated cost; one authoritative segment calculation for preview and billing; approved client rates only (owner, 2026-11-26) |
| MVP-11 | Contact book, groups, and segmentation | Client user | Deferrable | **Deferrable** — saved contacts and contact groups deferred from the MVP; the MVP portal takes a manually entered mobile number per single-recipient message (owner, 2026-11-26) |
| MVP-12 | Campaign scheduling | Client user | Deferrable* | **Deferrable** — campaign scheduling deferred with campaign approval and bulk sending; a separate scheduled-single-message feature is not ruled out (owner, 2026-11-26) |
| MVP-13 | Campaign approval by admin (policy-based) | Admin | ~~Essential~~ **Deferrable** | **Deferred beyond the MVP** — no bulk sending or campaign-sending workflow in the MVP; essential abuse controls stay (MVP-01, MVP-05, MVP-29, MVP-30) (owner, 2026-11-25); counted as **Deferred** in the approved scope (owner, 2026-11-26) |
| MVP-14 | Opt-out / suppression enforcement | System, admin | Essential | **MVP (manual lists)** — platform-wide and tenant-specific suppression lists checked for every outbound message before dispatch (owner, 2026-11-26) |
| MVP-15 | Inbound messages (replies, keywords) | Client | Deferrable | **Post-MVP** — the MVP is one-way transactional messaging (owner, 2026-11-26) |
| MVP-16 | Per-message delivery status and history | Client user, developer | Essential | **MVP (API and portal)** — append-only history; masked numbers; uncertain kept distinct; no message content by default (owner, 2026-11-26) |
| MVP-17 | Delivery report export (CSV) | Client user | Essential | **Post-MVP** — API listing and portal history suffice initially (owner, 2026-11-26) |
| MVP-18 | Webhooks for status updates | Developer | Deferrable* | **Post-MVP** — API polling initially; no webhook infrastructure (owner, 2026-11-26) |
| MVP-19 | Analytics dashboards | Client user | Deferrable | **Post-MVP** — basic status counts may be part of MVP-16 (owner, 2026-11-26) |
| MVP-20 | Prepaid credit balance, admin top-up, credit ledger | Client, admin | Essential | **MVP (prepaid only)** — immutable credit ledger, manual Finance top-ups, reservations, charges, releases; implementation blocked until the billing rules are decided (owner, 2026-11-26) |
| MVP-21 | Client-specific pricing configuration | Admin | Essential | **MVP (flat)** — one versioned per-segment rate per client; never send with a missing or unapproved rate (owner, 2026-11-26) |
| MVP-22 | Transaction history and statements | Client, admin | Essential | **MVP (history only)** — top-ups, reservations, charges, releases; client Owners and authorized Finance staff only (owner, 2026-11-26) |
| MVP-23 | Postpaid accounts and invoicing | Admin | Deferrable | **Post-MVP** — launch prepaid-only (owner, 2026-11-26) |
| MVP-24 | Online payments | Client | Future | **Deferred** — manual Finance top-ups; no payment gateway (owner, 2026-11-26) |
| MVP-25 | Provider configuration and basic routing | Admin | Essential | **MVP (simulator first)** — Operations-controlled routing against the simulator; no real telco adapter without separate approval (owner, 2026-11-26) |
| MVP-26 | Automatic failover routing between telcos | System | Future | **Deferred** — no automatic failover; manual route changes apply to new submissions only (owner, 2026-11-26) |
| MVP-27 | Telco usage reconciliation report | Admin/finance | Essential | **MVP (internal usage summary)** — by client, simulated provider and period; actual telco reconciliation mandatory before production billed traffic (PG-3) (owner, 2026-11-26) |
| MVP-28 | Operations monitoring (queues, binds, failures) | Ops | Essential | **MVP (metrics and minimal alerts)** — existing observability only; D-066 open (owner, 2026-11-26) |
| MVP-29 | Audit log | Admin, auditor | Essential | **Essential** — audit logging of administrative and security actions (owner, 2026-11-25) |
| MVP-30 | Abuse controls: velocity limits, alerts, kill switch | System, admin | Essential | **Essential** — tenant/API-key velocity limits, abuse alerts, operational kill switch (owner, 2026-11-25) |
| MVP-31 | Sub-accounts and reseller hierarchy | Client, reseller | Future | **Deferred** — flat tenant-per-client model (owner, 2026-11-26) |
| MVP-32 | Public API documentation | Developer | Essential | **MVP** — reference generated from OpenAPI, plus quickstart and retry/idempotency guides (owner, 2026-11-26) |
| MVP-33 | Telco simulator (development and demos) | Engineering | Essential | **MVP** — acceptance, rejection, timeout, delayed, duplicate and out-of-order receipt scenarios; never sends real SMS (owner, 2026-11-26) |
| MVP-34 | Data retention and expiry | System | Essential before production | **MVP (classification first)** — retention classes from the first table; automated expiry before production (PG-4) (owner, 2026-11-26) |

\* Tier depends strongly on launch-customer answers (see each item's open questions).

## Owner approval of the MVP scope (2026-11-26)

The owner reviewed the 25 undecided capabilities in five batches and, on 2026-11-26, approved all 34 decisions
in the summary above, the related discovery-register recordings, an amendment to Phase 2 exit criterion EX-1,
and the production gates below. This approval is the written record that EX-1 requires
([Phase 2 exit checklist](phase2-exit-checklist.md)); it is also recorded in
[decision log §14](../engineering/decision-log.md#14-owner-decisions--mvp-scope-phase-2-ex-1-2026-11-26).
It approves **scope only**: it does not authorize implementation, Phase 4, or any change to the billing rules,
legal questions or launch-customer identity listed under "Still open" below.

### Approved scope

| Owner decision | Count | Capabilities |
| -------------- | ----- | ------------ |
| **MVP** | 21 | MVP-01, 03, 04, 05, 06, 07, 09, 10, 14, 16, 20, 21, 22, 25, 27, 28, 29, 30, 32, 33, 34 |
| **Post-MVP** | 8 | MVP-02, 11, 12, 15, 17, 18, 19, 23 |
| **Deferred** | 5 | MVP-08, 13, 24, 26, 31 |
| **Out of scope** | 0 | — |

Within MVP capabilities the owner also deferred: a separate client Billing role (MVP-03); IP allowlisting and
advanced scope management for API keys (MVP-06); pricing by network, message type and volume tier (MVP-21);
periodic statements and official tax documents (MVP-22). Detailed client and admin permissions belong in the
RBAC design (Phase 2B, EX-10).

### Launch customers (EX-1 amendment)

Launch customers **may remain TBD until pilot discovery** (owner amendment to EX-1, 2026-11-26). **At least one
validated pilot customer is required before production onboarding or traffic** (gate PG-8). D-061 stays open
with this trigger; no launch customer is named or assumed here.

### Production gates

None of these is met today. Each must be met before the activity it names; they are in addition to, not
instead of, the Phase 2 exit criteria and any later phase approvals.

| Gate | Requirement | Source |
| ---- | ----------- | ------ |
| PG-1 | **No real telco adapter or production provider integration** without separate owner approval and official telco documentation; telco protocols, throughput limits, network routing and MNP behaviour are never invented (D-070, D-100, D-101, D-107; EX-3) | MVP-25, batch 1 condition |
| PG-2 | **Production provider integration stays gated by the separate regulatory decisions** (counsel's opinion, EX-4) | Batch 1 condition |
| PG-3 | **Reconciliation against actual telco usage records** is in place before production billed telco traffic | MVP-27 |
| PG-4 | **Automated retention expiry** is in place before production, using retention periods approved through legal review; financial records and legal holds are respected | MVP-34 |
| PG-5 | **No message is sent with a missing or unapproved client rate**; prices are never invented (a runtime rule from the first billed message) | MVP-10, MVP-21 |
| PG-6 | **The simulator never serves production customer traffic or generates customer-billable usage** | MVP-33 (owner, 2026-11-26) |
| PG-7 | **Production-critical alert coverage and D-066 operational ownership are resolved** before production. This does not by itself require 24/7 staffing | MVP-28 (owner, 2026-11-26) |
| PG-8 | **At least one validated pilot customer** before production onboarding or traffic | EX-1 amendment, D-061 |
| PG-9 | **Every real adapter passes the provider contract suite using the provider's official specifications**, plus validation in the provider's test environment where one is available. If none is available, the gap is documented and an alternative validation is explicitly approved by the owner | EX-3 simulator-only decision (owner, 2026-11-26; [decision log §15](../engineering/decision-log.md#15-owner-decisions--simulator-only-development-phase-2-ex-3-2026-11-26)) |

### Prerequisites before implementation

Choosing a capability for the MVP does not make it buildable. These decisions or deliverables come first
(their owners and status are in the [discovery register](../requirements/discovery-register.md) and the
[decision log](../engineering/decision-log.md)):

| Capability | Needs first |
| ---------- | ----------- |
| MVP-10, MVP-20, MVP-21, MVP-22 (money) | ADR-0011 (ledger) and ADR-0019 (representation) decided; D-080 (charge point), D-081 (credit unit), D-083 (refunds), D-092 (precision), D-093 (rounding), D-096 (billing of uncertain messages) |
| MVP-07 (uncertain outcomes) | D-095 (ambiguous submission policy) |
| MVP-25, MVP-33 (providers, simulator) | ADR-0012 (provider abstraction and simulator) decided |
| MVP-03, MVP-04 (roles) | RBAC design (Phase 2B, EX-10) |
| Any tenant-owned table | Database role separation (RLS-4), in its own approved group |
| Any implementation at all | Phase 4 authorization by the owner (not given) |

### Still open (not decided by this approval)

- **Billing rules:** D-080, D-081, D-083, D-092, D-093, D-096 (billing part), D-097; ADR-0011 and ADR-0019.
- **Uncertain-outcome policy:** D-095; webhook semantics (D-099, webhook part) until webhooks are scheduled.
- **Legal questions:** consent and opt-out obligations (D-122), retention periods (D-124, D-073), due diligence
  (D-130), reseller authorization (D-088, D-113) and the regulatory path (EX-4).
- **Launch-customer identity:** D-061.
- **Operations:** alert recipients and coverage (D-066); staffing numbers and turnaround (D-078).

## Capability details

### MVP-01 — Client account creation by admin
- **Problem:** clients need an isolated account before sending.
- **Capability:** admin creates a tenant with company details, status, and pricing.
- **Value:** controlled onboarding while volumes are small; strong abuse control.
- **Dependencies:** D-085 (KYC policy), D-161 (contracting entity).
- **Security:** tenant isolation from day one; audited creation.
- **Acceptance criteria:** admin can create, suspend, and reactivate a tenant; all actions audited.
- **Open questions:** what documents are required before activation (D-085, D-130)?
- **Owner decision (2026-11-25, PR #7 review):** confirmed **Essential**. Admin-controlled tenant onboarding
  and activation is one of the essential abuse-prevention controls kept when campaign approval (MVP-13) was
  deferred.

### MVP-02 — Self-service registration with KYC
- **Problem:** manual onboarding does not scale.
- **Capability:** prospects register and upload documents for review.
- **Value:** growth beyond launch customers.
- **Dependencies:** D-085, D-130, document storage design.
- **Security:** fraudulent sign-ups are a key abuse vector; document storage holds sensitive data.
- **Acceptance criteria:** registration creates a pending tenant that cannot send until approved.
- **Open questions:** is self-service expected at launch by the business plan (D-163)?
- **Owner decision (2026-11-26, EX-1 review):** **Post-MVP.** Onboarding stays staff-controlled (MVP-01); self-service registration and document upload are deferred.

### MVP-03 — Client users and basic roles
- **Problem:** several people in a client organization use the portal.
- **Capability:** invite users; roles such as owner, sender, viewer, billing.
- **Value:** clients control who can send and spend.
- **Dependencies:** RBAC design (Phase 2B), D-064.
- **Security:** least privilege; MFA availability for clients.
- **Acceptance criteria:** a viewer cannot send; a sender cannot change billing; role changes audited.
- **Open questions:** which roles do launch customers need?
- **Owner decision (2026-11-26, EX-1 review):** **MVP (minimal).** Three client roles: **Owner** manages users and API keys;
  **Sender** accesses sending functions; **Viewer** is read-only. A separate Billing role is deferred. Detailed
  permissions belong in the RBAC design (Phase 2B).

### MVP-04 — Admin users, roles, and mandatory MFA
- **Problem:** our staff hold powerful permissions (credits, pricing, routing).
- **Capability:** admin accounts with separated roles and mandatory MFA (D-021).
- **Value:** reduces insider and account-takeover risk.
- **Dependencies:** D-021, D-091 (separate admin portal).
- **Security:** highest-risk accounts in the system.
- **Acceptance criteria:** no admin action without MFA; role-restricted actions enforced server-side.
- **Open questions:** how many staff and which roles (D-078)?
- **Owner decision (2026-11-26, EX-1 review):** **MVP.** Internal roles **Super Admin**, **Operations** and **Finance**. MFA is mandatory
  for **all human internal admin accounts** (D-021, widened from Super Admin only), and least privilege is
  enforced. Detailed permissions belong in the RBAC design (Phase 2B). Staff numbers stay open (D-078).

### MVP-05 — Sender ID request and approval workflow
- **Problem:** branded sender IDs must be registered with telcos and protected from impersonation.
- **Capability:** client requests a sender ID with supporting documents; admin approves; per-telco
  registration status tracked.
- **Value:** required for branded messages; key abuse control (D-022).
- **Dependencies:** D-103 (telco process), D-123, D-128.
- **Security:** impersonation of banks, government, and brands.
- **Acceptance criteria:** messages cannot use unapproved sender IDs; decisions audited.
- **Open questions:** can we register on the client's behalf with each telco?
- **Owner decision (2026-11-25, PR #7 review):** confirmed **Essential** — sender IDs are approved before
  use. Kept as an essential abuse-prevention control when MVP-13 was deferred.

### MVP-06 — API keys
- **Problem:** developers need credentials to call the API.
- **Capability:** create, label, scope, and revoke keys; shown once; stored hashed.
- **Value:** enables integration.
- **Dependencies:** security baseline.
- **Security:** keys allow spending; leaks must be containable.
- **Acceptance criteria:** revoked keys stop working immediately; key events audited.
- **Open questions:** do customers require IP allowlisting per key (CU-22)?
- **Owner decision (2026-11-26, EX-1 review):** **MVP (core).** Create, label, show once, store hashed and revoke API keys. The existing
  `messages:send` and `messages:read` scopes of the public API contract are supported. IP allowlisting (CU-22)
  and advanced scope management are deferred.

### MVP-07 — Single message via public API
- **Problem:** OTPs and alerts are sent one at a time by client systems.
- **Capability:** submit a message with an idempotency key; receive a message ID.
- **Value:** core revenue path for API customers.
- **Dependencies:** ADR-0018 / D-094, D-023, ADR-0020, ADR-0021.
- **Security:** authentication, rate limits, input validation.
- **Acceptance criteria:** retries with the same key create one message and one reservation (scenarios IK-1–IK-11).
- **Open questions:** is the idempotency key required (D-094a)?
- **Owner decision (2026-11-26, EX-1 review):** **MVP.** Single-message public API with a mandatory idempotency key (ADR-0018, D-094a).
  The same key with the same payload must not create another message record; the same key with a different
  payload is rejected (`422`, `idempotency-key-reused`, [API idempotency §3.4](../architecture/api-idempotency.md)).
  This does **not** promise exactly-once provider delivery.

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
- **Owner decision (2026-11-25, MVP-13 review) — portal recipient scope:** one portal send operation
  accepts **exactly one recipient**. The MVP portal does **not** support pasting multiple recipient numbers
  into one send operation, uploading recipient files, or sending to contact groups or several selected
  recipients in one operation. A customer may still submit several separate single-recipient messages;
  each is subject to the platform's essential rate limits and abuse controls (MVP-30).

### MVP-10 — Composer with segment and cost estimate
- **Problem:** encoding surprises (e.g. emoji) can double cost (D-029).
- **Capability:** live segment count, encoding indicator, estimated cost before sending.
- **Value:** trust and fewer disputes.
- **Dependencies:** D-105 (telco encoding), ADR-0019 (money), D-093 (rounding).
- **Security:** none specific.
- **Acceptance criteria:** estimate equals the reservation for the same input (scenario FI-6).
- **Open questions:** do telcos count segments as we do (D-117)?
- **Owner decision (2026-11-26, EX-1 review):** **MVP (full).** Display encoding, segment count and estimated cost. One authoritative
  segment calculation is used for both the preview and billing. Only approved client rate cards are used;
  prices are never invented (PG-5). Representation and rounding stay open (ADR-0019, D-092, D-093).

### MVP-11 — Contact book, groups, segmentation
- **Problem:** repeat senders manage audiences.
- **Capability:** stored contacts, groups, attributes, filters.
- **Value:** convenience and retention of marketing clients.
- **Dependencies:** D-071, D-073, privacy role (D-121).
- **Security:** large store of personal data; export controls.
- **Acceptance criteria:** tenant-isolated; import validation; export audited.
- **Open questions:** do launch customers keep contacts in their own systems instead?
- **Owner decision (2026-11-26, PR #7 review):** confirmed **Deferrable**. Both saved contact management and
  contact groups are deferred from the MVP. In the MVP portal, the customer enters the mobile number
  manually for each single-recipient message (MVP-09); customers may submit several separate messages, but
  there is no contact-group sending. Saved contacts may be introduced **independently** of contact groups
  in a later release. Contact volumes and import sizes (D-071) and contact-data retention (D-073) are not
  decided by this.

### MVP-12 — Campaign scheduling
- **Problem:** campaigns go out at planned times.
- **Capability:** schedule a send; edit/cancel before release.
- **Value:** expected by marketing clients.
- **Dependencies:** D-122 (time-of-day rules may apply), approval policy.
- **Security:** scheduled sends must re-check balance, suppression, and approval at release time.
- **Acceptance criteria:** cancelled campaigns never send; release respects any time restrictions.
- **Open questions:** do launch customers need scheduling on day one?
- **Owner decision (2026-11-26, PR #7 review):** confirmed **Deferrable**. **Campaign scheduling** is
  deferred from the MVP, alongside campaign approval (MVP-13) and bulk sending (MVP-08, MVP-09). This
  decision concerns campaign scheduling only; it does **not** permanently rule out a separate
  scheduled-single-message feature in a future release, which would need its own decision.

### MVP-13 — Campaign approval
- **Problem:** large or first-time campaigns are the main abuse vector (D-025, D-087).
- **Capability:** policy decides which campaigns need admin approval.
- **Value:** protects telco relationships.
- **Dependencies:** D-087, D-078 (staffing).
- **Security:** approval actions audited.
- **Acceptance criteria:** campaigns needing approval cannot be released without it.
- **Open questions:** thresholds; turnaround time customers will accept.
- **Owner decision (2026-11-25, PR #7 review):** tier changed from Essential to **Deferrable**. Dedicated
  campaign-level approval workflows are **deferred beyond the MVP**, because the MVP has no bulk sending and
  no campaign-sending workflow (MVP-08, MVP-09). This does **not** remove the essential abuse-prevention
  controls, which stay in the MVP: admin-controlled tenant onboarding and activation (MVP-01), sender ID
  approval before use (MVP-05), tenant/API-key velocity limits, abuse alerts, and an operational kill
  switch (MVP-30), and audit logging of administrative and security actions (MVP-29). The approval policy
  itself is **not** fully decided: detailed approval thresholds, first-campaign policies, and operational
  turnaround remain open for a later release (D-025, D-087).
- **Owner decision (2026-11-26, EX-1 review):** counted as **Deferred** in the approved scope (21 MVP, 8 Post-MVP, 5 Deferred); the
  2026-11-25 decision above is unchanged.

### MVP-14 — Opt-out / suppression enforcement
- **Problem:** recipients who opted out must not be messaged (D-024).
- **Capability:** tenant and platform suppression lists checked before dispatch.
- **Value:** compliance and trust.
- **Dependencies:** D-122 (legal requirements), D-063/MVP-15 for keyword opt-outs.
- **Security:** suppression lists contain personal data.
- **Acceptance criteria:** a suppressed number is never sent to; suppression changes audited.
- **Open questions:** which opt-out keywords and records are legally required (LR-08, D-122)?
- **Owner decision (2026-11-26, EX-1 review):** **MVP (manual lists).** Platform-wide and tenant-specific suppression lists are maintained
  and every outbound message is checked before dispatch. Changes are audited and tenant isolation is preserved.
  Keyword opt-out waits for inbound messages (MVP-15). Legal obligations stay open (D-122).

### MVP-15 — Inbound messages
- **Problem:** replies and keyword opt-outs require inbound handling.
- **Capability:** receive MO messages and route them to the right client.
- **Value:** two-way use cases; automated opt-out.
- **Dependencies:** D-063, telco MO support (checklist §12).
- **Security:** inbound content is personal data.
- **Acceptance criteria:** STOP-type replies update suppression automatically.
- **Open questions:** do launch customers need replies at all?
- **Owner decision (2026-11-26, EX-1 review):** **Post-MVP.** The MVP focuses on one-way transactional messaging. Revisit when inbound telco
  support, customer demand or legal requirements justify it (D-063).

### MVP-16 — Delivery status and history
- **Problem:** clients must know what happened to each message.
- **Capability:** per-message status timeline using the proposed status model.
- **Value:** core product promise.
- **Dependencies:** ADR-0021, D-096, D-099, D-104.
- **Security:** tenant isolation; masked display where appropriate.
- **Acceptance criteria:** uncertain outcomes are shown as uncertain, not as failures (scenarios DR-1–DR-10).
- **Open questions:** status labels shown to customers (D-099).
- **Owner decision (2026-11-26, EX-1 review):** **MVP (API and portal).** Per-message status with append-only history. Phone numbers are
  masked. Uncertain outcomes stay distinct from failed, accepted and delivered. Message content is not exposed
  by default (D-073). Basic status counts may be included here (see MVP-19).

### MVP-17 — Delivery report export
- **Problem:** clients reconcile and report internally.
- **Capability:** CSV export of message statuses for a period.
- **Value:** expected by enterprises.
- **Dependencies:** D-072, D-073.
- **Security:** formula-injection protection; export audited.
- **Acceptance criteria:** export matches the status history; large exports run in the background.
- **Open questions:** history length required.
- **Owner decision (2026-11-26, EX-1 review):** **Post-MVP.** The API listing and the portal history (MVP-16) are sufficient initially.
  Revisit CSV export based on customer demand.

### MVP-18 — Webhooks
- **Problem:** API clients want push notifications instead of polling.
- **Capability:** signed, at-least-once status callbacks with event IDs.
- **Value:** strongly expected by many API integrators.
- **Dependencies:** D-099, messaging reliability §2.4.
- **Security:** SSRF protection; HMAC signatures.
- **Acceptance criteria:** duplicate deliveries carry the same event ID; failed callbacks retried with backoff.
- **Open questions:** do launch API customers require webhooks on day one (CU-21)?
- **Owner decision (2026-11-26, EX-1 review):** **Post-MVP.** API clients poll (`GET /v1/messages/{message_id}`) initially; webhook
  infrastructure is deferred. Webhook semantics stay open (D-099).

### MVP-19 — Analytics dashboards
- **Problem:** clients want trends and delivery rates.
- **Capability:** charts by period, network, campaign.
- **Value:** retention; upsell.
- **Dependencies:** D-072, ADR-0008.
- **Security:** tenant isolation in aggregates.
- **Acceptance criteria:** figures reconcile with exports.
- **Open questions:** which metrics matter to launch customers?
- **Owner decision (2026-11-26, EX-1 review):** **Post-MVP.** Analytics dashboards are deferred. Basic message status counts may be included
  in MVP-16.

### MVP-20 — Prepaid credit, admin top-up, ledger
- **Problem:** we must not deliver traffic we cannot bill.
- **Capability:** balance, reservations, settlement, admin top-up with audit.
- **Value:** core financial control.
- **Dependencies:** ADR-0011, ADR-0019, D-080–D-083, D-092, D-093.
- **Security:** financial integrity; admin adjustments require reason and audit.
- **Acceptance criteria:** no overspend under concurrency; balance equals ledger (scenarios FI-1–FI-10).
- **Open questions:** credit unit (D-081); refund policy (D-083).
- **Owner decision (2026-11-26, EX-1 review):** **MVP (prepaid only).** An immutable credit ledger, manual top-ups by Finance, reservations,
  charges and releases. **Implementation remains blocked until the required billing rules are decided**
  (ADR-0011, ADR-0019, D-080, D-081, D-083, D-092, D-093, D-096); this scope decision does not decide them.

### MVP-21 — Client-specific pricing
- **Problem:** clients negotiate different rates.
- **Capability:** price per segment per client, possibly per network or traffic type.
- **Value:** commercial flexibility.
- **Dependencies:** D-084, D-092.
- **Security:** pricing changes audited; versioned.
- **Acceptance criteria:** each charge references the rate version in effect.
- **Open questions:** pricing dimensions (D-084).
- **Owner decision (2026-11-26, EX-1 review):** **MVP (flat).** One versioned per-segment rate per client. Pricing by network, by message
  type and by volume tier is deferred. **No message is sent with a missing or unapproved rate** (PG-5).

### MVP-22 — Transaction history and statements
- **Problem:** clients and finance need records of top-ups and charges.
- **Capability:** ledger-based history and periodic statements.
- **Value:** trust; accounting.
- **Dependencies:** D-127 (invoicing rules).
- **Security:** tenant isolation.
- **Acceptance criteria:** statements reconcile to the ledger.
- **Open questions:** official invoice/receipt requirements (D-127).
- **Owner decision (2026-11-26, EX-1 review):** **MVP (history only).** Show top-ups, reservations, charges and releases. Periodic
  statements and official tax documents (D-127) are deferred. Financial history is initially restricted to
  client **Owners** and authorized **Finance** staff.

### MVP-23 — Postpaid accounts and invoicing
- **Problem:** large enterprises and government may require monthly invoices.
- **Capability:** credit limits, invoice generation, collections tracking.
- **Value:** access to larger customers.
- **Dependencies:** D-082, D-127.
- **Security:** credit risk controls.
- **Acceptance criteria:** sending stops at the credit limit.
- **Open questions:** do launch customers require postpaid (CU-30)?
- **Owner decision (2026-11-26, EX-1 review):** **Post-MVP.** Launch prepaid-only (D-082). Revisit for enterprise requirements (CU-30).

### MVP-24 — Online payments
- **Problem:** manual top-ups are slow.
- **Capability:** payment gateway integration.
- **Value:** self-service growth.
- **Dependencies:** D-065, payment provider choice.
- **Security:** payment data handling.
- **Acceptance criteria:** each confirmed payment produces one ledger credit, even if the payment
  provider notifies us more than once (idempotent on the payment reference).
- **Open questions:** which payment methods?
- **Owner decision (2026-11-26, EX-1 review):** **Deferred.** Finance performs manual top-ups; there is no payment gateway initially (D-065).

### MVP-25 — Provider configuration and basic routing
- **Problem:** traffic must go to the right telco connection.
- **Capability:** admin-configured routes per destination network with rate limits.
- **Value:** required to send at all.
- **Dependencies:** D-070 (MNP), D-107, D-101, ADR-0012.
- **Security:** routing changes audited; restricted role.
- **Acceptance criteria:** rate limits never exceed configured telco limits.
- **Open questions:** how the destination network is determined (D-070).
- **Owner decision (2026-11-26, EX-1 review):** **MVP (simulator first).** Basic provider configuration and Operations-controlled routing
  against the simulator. **No real telco adapter without separate approval** (PG-1, PG-2). Telco protocols,
  throughput limits, network routing and MNP behaviour are never invented (D-070, D-100, D-101, D-107).

### MVP-26 — Automatic failover routing
- **Problem:** a telco outage stops delivery.
- **Capability:** automatic rerouting when allowed.
- **Value:** resilience.
- **Dependencies:** D-119 (telco permission), D-107.
- **Security:** must not violate telco terms.
- **Acceptance criteria:** failover only along approved routes.
- **Open questions:** is failover contractually allowed?
- **Owner decision (2026-11-26, EX-1 review):** **Deferred.** No automatic failover. Messages with an uncertain submission outcome are never
  blindly retried. Manual route changes (MVP-25) apply to new submissions only.

### MVP-27 — Telco usage reconciliation report
- **Problem:** telco invoices must match our records (R-08, R-32).
- **Capability:** compare our submissions and receipts with telco CDRs; list discrepancies.
- **Value:** protects margin.
- **Dependencies:** D-108 (CDR access), D-117.
- **Security:** finance data access restricted.
- **Acceptance criteria:** every discrepancy is reported; nothing auto-corrected.
- **Open questions:** CDR format and frequency.
- **Owner decision (2026-11-26, EX-1 review):** **MVP (internal usage summary).** Internal usage summaries by client, simulated provider and
  period. **Reconciliation against actual telco usage records is mandatory before production billed telco
  traffic** (PG-3; D-108, D-117).

### MVP-28 — Operations monitoring
- **Problem:** failures must be noticed quickly (R-40).
- **Capability:** dashboards and alerts for queues, connections, error rates, receipt delays.
- **Value:** reliability.
- **Dependencies:** D-066, ADR-0016.
- **Security:** monitoring must not expose personal data.
- **Acceptance criteria:** alerts fire for lost telco connections and growing queue age.
- **Open questions:** who receives alerts (D-066)?
- **Owner decision (2026-11-26, EX-1 review):** **MVP (metrics and minimal alerts).** Use the existing observability infrastructure. Critical
  failures, uncertain submissions and kill-switch events are observable. No new monitoring infrastructure
  without approval. Alert recipients and coverage (D-066) stay open and must be resolved before production
  (PG-7).

### MVP-29 — Audit log
- **Problem:** security and financial actions must be traceable.
- **Capability:** append-only audit records viewable by authorized staff.
- **Value:** investigations, disputes, regulators.
- **Dependencies:** security baseline §9.
- **Security:** tamper evidence; restricted access.
- **Acceptance criteria:** every listed sensitive action produces an audit record.
- **Open questions:** retention period (D-124).
- **Owner decision (2026-11-25, PR #7 review):** confirmed **Essential** — audit logging of administrative
  and security actions. Kept as an essential abuse-prevention control when MVP-13 was deferred.

### MVP-30 — Abuse controls
- **Problem:** smishing through our platform could end telco relationships (R-20).
- **Capability:** per-tenant velocity limits, anomaly alerts, immediate suspension of tenant/sender ID/campaign.
- **Value:** protects the business.
- **Dependencies:** D-086, D-087, D-110, D-123.
- **Security:** central to the threat model.
- **Acceptance criteria:** suspension takes effect immediately and is audited.
- **Open questions:** content/URL rules required by telcos (D-110).
- **Owner decision (2026-11-25, PR #7 review):** confirmed **Essential** — tenant and API-key velocity
  limits, abuse alerts, and an operational kill switch. Kept when MVP-13 was deferred; in the MVP they also
  bound customers who submit many separate single-recipient messages (MVP-09). Campaign-level suspension
  applies once campaigns exist in a later release.

### MVP-31 — Sub-accounts and reseller hierarchy
- **Problem:** BPOs and resellers serve their own clients.
- **Capability:** nested accounts with delegated billing.
- **Value:** wholesale revenue.
- **Dependencies:** D-064, D-088, D-113.
- **Security:** complex isolation and billing boundaries.
- **Acceptance criteria:** to be defined if adopted.
- **Open questions:** do telcos and counsel permit it?
- **Owner decision (2026-11-26, EX-1 review):** **Deferred.** A flat tenant-per-client model. Sub-accounts, reseller hierarchy and delegated
  billing are deferred (D-064, D-088).

### MVP-32 — Public API documentation
- **Problem:** developers need clear docs to integrate.
- **Capability:** documentation generated from the OpenAPI contract.
- **Value:** faster integration.
- **Dependencies:** ADR-0015, ADR-0018.
- **Security:** no secrets in examples.
- **Acceptance criteria:** documentation matches the contract.
- **Open questions:** none.
- **Owner decision (2026-11-26, EX-1 review):** **MVP.** API reference documentation generated from the OpenAPI contract, plus a concise
  quickstart and a retry/idempotency guide. The documentation states that idempotency does **not** guarantee
  exactly-once SMS delivery.

### MVP-33 — Telco simulator
- **Problem:** development cannot wait for telco access.
- **Capability:** configurable simulated provider.
- **Value:** testing and demonstrations without real SMS.
- **Dependencies:** D-026, ADR-0012.
- **Security:** must never deliver to real numbers.
- **Acceptance criteria:** provider contract suite passes against it.
- **Open questions:** none.
- **Owner decision (2026-11-26, EX-1 review):** **MVP.** The simulator covers acceptance, rejection, timeout, and delayed, duplicate and
  out-of-order receipt scenarios. It never sends a real SMS or connects to a real telco, and it never serves
  production customer traffic or generates customer-billable usage (PG-6).

### MVP-34 — Data retention and expiry
- **Problem:** personal data must not be kept longer than necessary (LR-09).
- **Capability:** retention classes and automated expiry.
- **Value:** compliance; storage cost.
- **Dependencies:** D-124, D-073.
- **Security:** deletion must respect legal holds.
- **Acceptance criteria:** data older than its retention period is removed or anonymized.
- **Open questions:** retention periods (counsel).
- **Owner decision (2026-11-26, EX-1 review):** **MVP (classification first).** Retention classes are assigned from the first database
  table. **Automated expiry is mandatory before production** (PG-4), using periods approved through legal
  review. Retention periods are never invented; financial records and legal holds are respected.
