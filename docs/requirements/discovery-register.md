# Discovery Register

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — living document | Project owner | 2026-11-25 (owner decisions recorded on D-023, D-094a–e, D-096 (customer-visible part), D-099 (status part), D-143–D-147 for the public API contract; no ADR status changed) |

This register lists everything we know, assume, or still need to find out. **It is the authority on
what is known.** AI agents must not treat any item as confirmed unless it is in §1 with evidence.

## How to use this register

- Each item has an ID (`D-###`), a category, and a status.
- When an item is resolved, record the answer, the **evidence** (document name, date, who confirmed),
  and move it to §1 if it becomes a confirmed fact. Do not delete resolved items.
- Categories:
  - **Confirmed** — verified with evidence.
  - **Owner-stated** — stated by the project owner as intent or preference; not externally verified.
  - **Proposed** — a requirement or design suggestion awaiting approval.
  - **Assumption** — something we are provisionally treating as true to make progress; must be validated.
  - **Unknown** — information we do not have.
  - **Business decision** — the owner must decide.
  - **Telco-dependent** — only the telcos can answer.
  - **Regulatory-dependent** — depends on law/regulators; needs qualified legal advice.
  - **Engineering decision** — a technical choice reserved for the owner (§9).
- **Routing (Phase 2A):** each open item's priority (P0/P1/P2/Future), intended respondent, questionnaire,
  and the evidence needed to resolve it are recorded in the
  [owner discovery workbook](../discovery/owner-discovery-workbook.md). This register remains the authority
  for each item's **status and category**; evidence is logged in the
  [evidence register](../discovery/evidence-register.md).

---

## 1. Confirmed information (externally verified)

| ID | Item | Evidence |
| -- | ---- | -------- |
| — | *No business, telco, or regulatory item has been externally verified yet.* | — |

## 2. Owner-stated information (intent, not verified)

| ID | Item | Notes |
| -- | ---- | ----- |
| D-001 | The business intends to operate as an independent A2P SMS aggregator in the Philippines. | Subject to telco agreements and regulatory approval. |
| D-002 | Target telco partners: Smart, Globe, PLDT Enterprise; others possibly later. | Discussions reported, not verified. |
| D-003 | Initial discussions have reportedly taken place with Smart, Globe, PLDT, and NTC. | Need meeting records, contacts, and outcomes. |
| D-004 | Projected monthly traffic of ~5 billion (unit unknown). | See [traffic capacity discovery](traffic-capacity-discovery.md). |
| D-005 | Technology preferences: React/TS/Vite/Tailwind, Go/Chi, PostgreSQL; candidates Kafka/Redpanda, Redis, ClickHouse; Docker/Prometheus/Grafana. | Each infrastructure component requires justification (decision log). |
| D-006 | Monorepo with frontend, backend, infrastructure, contracts, and docs. | Implemented in Phase 1. |
| D-007 | Product capabilities as listed in the Phase 1 brief (client portal, admin portal, messaging, billing, developer platform, security/compliance). | Treated as expectations, not an SRS. |
| D-008 | AI coding agents will perform most implementation; one human owner approves. | Shapes the workflow documents. |

## 3. Proposed requirements (awaiting approval)

| ID | Proposal | Rationale |
| -- | -------- | --------- |
| D-020 | Define an MVP scope (candidate list in the product brief) before building features. | Avoids building everything at once. |
| D-021 | MFA mandatory for all Super Admin users. | Admins can move money and change routing. |
| D-022 | Sender IDs must be approved before use; impersonation checks during approval. | Smishing and brand impersonation risk. |
| D-023 | Public API supports idempotency keys on message submission (contract proposed in ADR-0018; open details in D-094). | Prevents duplicate operations and charges on client retries inside our platform. Does not by itself prevent duplicate telco submissions. **Owner decision 2026-11-25 (PR #7 review):** approved; the key is required (D-094a). ADR-0018 remains Proposed. |
| D-024 | Opt-out/suppression list is enforced platform-wide before dispatch. | Compliance and recipient protection. |
| D-025 | Campaigns above a configurable size (or from new tenants) require admin approval. | Abuse prevention; owner listed "campaign approval". |
| D-026 | A simulator provider adapter is built before any real telco adapter. | Enables development and testing without telco access. |
| D-027 | Admin portal deployed separately from client portal (ADR-0013). | Reduced attack surface. |
| D-028 | All timestamps stored in UTC and displayed in Philippine time (Asia/Manila) by default. | Consistency. |
| D-029 | Customers can see estimated segments and cost before sending. | Encoding surprises (emoji → UCS-2) can double cost. |

## 4. Assumptions (must be validated)

| ID | Assumption | Impact if wrong |
| -- | ---------- | --------------- |
| D-040 | Telco connectivity will be via SMPP (version to be confirmed). | Adapter design changes if HTTP APIs or other protocols are used. |
| D-041 | Telcos charge per segment. | Billing model and margin calculation change. |
| D-042 | Traffic is predominantly MT (outbound); MO (inbound) is limited to opt-outs/replies. | Inbound capacity and short-code needs change. |
| D-043 | Initial destinations are Philippine mobile numbers only. | International routing adds complexity. |
| D-044 | The platform will be hosted where it can meet telco connectivity requirements (e.g. fixed IPs, VPN). | Hosting choice may be constrained. |
| D-045 | Prepaid credit is the initial payment model. | Postpaid requires credit control, invoicing, collections. |
| D-046 | English-language UI at launch. | Filipino localization adds work. |
| D-047 | A single legal entity operates the platform and contracts with telcos. | Multi-entity billing affects ledger design. |

## 5. Unknowns

| ID | Question | Why it matters |
| -- | -------- | -------------- |
| D-060 | What does "5 billion monthly" measure, and how much is committed vs. projected? | Infrastructure sizing, telco capacity negotiation, cost. |
| D-061 | Who are the first launch customers and their use cases (OTP vs. marketing mix)? | MVP scope and priorities. |
| D-062 | Expected peak throughput (messages/second) and burst patterns. | Capacity design. |
| D-063 | Does the business need two-way messaging (MO), keywords, short codes? | Inbound processing and number provisioning. |
| D-064 | Do customers need sub-accounts / hierarchies (e.g. BPO with multiple clients, resellers)? | Tenant data model. |
| D-065 | Required integrations: payment gateways, accounting/ERP, CRM? | Scope. |
| D-066 | Who will operate the platform 24/7 (NOC, on-call)? | Operational design and alerting. |
| D-067 | Target availability and support SLAs offered to customers. | Redundancy and cost. |
| D-068 | Hosting: cloud provider, local data centre, or hybrid? Any data-residency expectations from customers (especially government and banks)? | Infrastructure and compliance. |
| D-069 | Coverage of DITO subscribers: required? Through which route? | Without a route, messages to DITO numbers may fail. |
| D-070 | How will the recipient's current network be determined given Mobile Number Portability? | Routing correctness and cost. |
| D-071 | Expected volume of contacts per tenant and size of CSV imports. | Import design and storage. |
| D-072 | Analytics and reporting requirements (granularity, history length, export formats). | Whether ClickHouse is needed. |
| D-073 | Message content storage: must full content be kept? For how long? Who may view it? | Privacy, storage, retention. |
| D-074 | Expected monthly ramp-up of traffic after launch, per launch customer. | Capacity planning, telco capacity requests, cash flow. |
| D-075 | Message length and encoding mix (single vs. multipart; share requiring UCS-2). | Segments drive cost and telco throughput. |
| D-076 | Backup and disaster-recovery targets (acceptable data loss and downtime). | Database replication, hosting, and cost (architecture question AQ-8). |
| D-077 | Split of security and incident responsibilities, and escalation paths, between us, telcos, and customers. | Incident handling and contract terms. |
| D-078 | Who reviews sender ID requests, campaign approvals, and abuse alerts operationally, and with what turnaround. | Manual approval workflows need staffing. |
| D-079 | Customer support model (hours, channels, tiers). | Staffing and tooling. |

## 6. Business decisions required

| ID | Decision | Options / notes |
| -- | -------- | --------------- |
| D-080 | Charge on submission or on delivery? | Delivery-based is friendlier to clients but exposes us to telco charges for undelivered messages. Depends on telco billing basis. |
| D-081 | Credit unit: pesos or SMS credits? | Affects ledger design (ADR-0011) and representation (ADR-0019). |
| D-082 | Prepaid only, or postpaid for some customers? | Credit limits, invoicing, collections. |
| D-083 | Refund policy for failed/undelivered messages. | Ledger release rules. |
| D-084 | Pricing structure: per network, per message type, volume tiers? | Pricing engine complexity. |
| D-085 | Customer onboarding / KYC requirements (business documents, verification). | Onboarding workflow. |
| D-086 | Acceptable use policy: prohibited content and industries. | Compliance and abuse prevention. |
| D-087 | Campaign approval policy (which campaigns need manual approval). | Admin workflow. |
| D-088 | Whether to allow reseller/wholesale customers. | Tenant model, regulatory exposure. |
| D-089 | MVP scope and launch customers. | Roadmap. |
| D-090 | Budget envelope for infrastructure and operations. | Technology choices. |
| D-091 | Single vs. separate admin and client portals (ADR-0013). | Security vs. simplicity. **Owner decision 2026-11-25 (ADR-0013):** one application, `client-portal`, initially, with shared `packages/ui`; `admin-portal` added when admin functionality begins. |
| D-092 | **Monetary representation and precision**: currency(ies); precision (decimal places) of actual telco wholesale rates and our retail rates; resulting storage representation and scale (scaled integer vs. exact decimal). | Requires actual rate cards and the pricing model. Centavos may be insufficient. See ADR-0019 and [financial integrity](../architecture/financial-integrity.md). |
| D-093 | **Rounding rules**: where rounding occurs (per segment, message, campaign, invoice), rounding mode, residue handling, and audit inputs recorded. | Determines estimate/charge consistency and reconciliation with telco invoices. |
| D-094 | **API idempotency details** (ADR-0018): (a) key required vs. optional; (b) scope per tenant + operation vs. per API credential; (c) whether business rejections (e.g. insufficient credit) are recorded; (d) retention period; (e) bulk request semantics (all-or-nothing vs. per-recipient). | Recommendations in [API idempotency](../architecture/api-idempotency.md). **Owner decision 2026-11-25 (PR #7 review):** (a) required; (b) per account (tenant) + operation; (c) insufficient-credit rejections are not recorded; (d) retention seven days; (e) bulk sending is excluded from the MVP, so bulk semantics are not needed now. Applied in the [public API contract](../../contracts/openapi/README.md#7-owner-decisions-and-remaining-provisional-items). ADR-0018 remains Proposed. |
| D-095 | **Policy for ambiguous provider submissions** (timeouts, crash after sending), per provider and traffic type: never resubmit, resubmit after status query, wait for receipt, or resubmit for selected traffic types. | Trade-off between losing a message and sending a duplicate SMS. Depends on provider capabilities (D-115). |
| D-096 | **Treatment of uncertain messages** (`SUBMIT_UNCERTAIN`, `UNKNOWN`): customer-visible label, billing (hold, charge, release), and effect of a late receipt after a release or refund. | Customer trust, margin, and reconciliation. **Owner decision 2026-11-25 (PR #7 review), customer-visible part only:** `SUBMIT_UNCERTAIN` is shown as its own status, `submission_uncertain`; `UNKNOWN` is shown as `unknown`. **Still open:** billing treatment and the effect of late receipts after a release or refund. |
| D-097 | **Multi-segment roll-up**: how per-segment outcomes determine message status, and billing of partially delivered messages. | Depends on provider receipt behaviour (D-104). |
| D-098 | **Conflicting final receipts**: resolution policy when a provider reports different final states for one message. | Reconciliation and customer reporting. |
| D-099 | **Customer-visible status set and webhook semantics** (statuses exposed; webhooks documented as at-least-once with event IDs). | Public API contract. **Owner decision 2026-11-25 (PR #7 review), status part:** the eight statuses (`accepted`, `provider_accepted`, `submission_uncertain`, `delivered`, `undelivered`, `unknown`, `rejected`, `failed`) and ten initial reason codes in the [public API contract](../../contracts/openapi/README.md#3-status-reference-customer-visible-statuses) are approved, with room for future extension. **Still open:** webhook semantics (webhooks are not in the contract). ADR-0021 remains Proposed. |

Business decisions continued (the D-080–D-099 range is full; new business decisions use D-160 onward):

| ID | Decision | Options / notes |
| -- | -------- | --------------- |
| D-160 | **Operating model**: independent aggregator with direct telco contracts, reseller of another aggregator, connectivity through an intermediary, or a hybrid. | Relates to owner-stated intent D-001; depends on D-113, D-118, D-120. |
| D-161 | **Ownership and contracting structure**: which legal entity signs telco and customer contracts; investors' role and decision rights. | Relates to assumption D-047; affects licensing and billing. |
| D-162 | **Revenue model and target gross margin** (per-segment margin, platform/setup/sender ID fees). | Depends on telco pricing (D-108) and customer price expectations. |
| D-163 | **Commitments already made** to customers, investors, or telcos (launch dates, prices, volumes, exclusivity). | Hidden commitments constrain scope and schedule. |

## 7. Telco-dependent requirements

Full checklist: [telco discovery checklist](../integrations/telco-discovery-checklist.md).

| ID | Item |
| -- | ---- |
| D-100 | Connectivity protocol(s) and versions (SMPP version, HTTP APIs, others). |
| D-101 | Throughput limits (TPS) per account and per bind; number of binds allowed. |
| D-102 | Network connectivity: public internet with IP allowlisting, VPN, or private link. |
| D-103 | Sender ID (masking) registration process, lead time, documentation required. |
| D-104 | Delivery receipt availability, format, and timing; status code meanings. |
| D-105 | Encoding support (GSM-7, UCS-2), concatenation method, maximum segments. |
| D-106 | Error codes and which are retryable; throttling behaviour. |
| D-107 | Whether each telco accepts off-net traffic (to other networks' subscribers). |
| D-108 | Pricing, billing basis (submitted vs. delivered), CDR access, settlement and reconciliation process. |
| D-109 | Test environment availability and production onboarding steps. |
| D-110 | Content restrictions (e.g. URLs, keywords), and A2P traffic classification rules. |
| D-111 | SLAs, maintenance windows, escalation contacts. |
| D-112 | Whether PLDT Enterprise and Smart are separate integrations or a single route. |
| D-113 | Aggregator/reseller authorization requirements imposed by each telco. |
| D-114 | Provider message ID format, uniqueness (across binds and over time), and reuse period. |
| D-115 | Provider capabilities for resolving ambiguous submissions: status-query API, provider-side duplicate detection. |
| D-116 | Receipt timing characteristics (typical and maximum delay) to set per-provider `UNKNOWN` windows. |
| D-117 | Number of decimal places in telco rates; telco's own rounding and segment-counting rules for billing. |
| D-118 | Whether connectivity will be direct to each telco or through an intermediary aggregator/connectivity provider (interim or permanent), and whether telcos offer direct aggregator access. |
| D-119 | Whether traffic for one network may fail over to another telco's route (commercial and technical terms). |

## 8. Regulatory-dependent requirements

These require advice from qualified Philippine telecommunications and data-privacy counsel. Nothing here
is legal advice. Public sources consulted so far are listed in the
[legal and regulatory research log](../discovery/legal-regulatory-research.md); they do not resolve any item.

| ID | Item |
| -- | ---- |
| D-120 | Which NTC registrations, licences, or authorizations are required to operate as an A2P aggregator / value-added service provider, and current status of each. Confirm the current legal framework, including any recent legislative changes. (Texts and leads: research log LR-01–LR-05, LR-16, LR-17.) |
| D-121 | Data Privacy Act of 2012 (RA 10173) obligations: lawful basis for processing recipient data, roles (we are likely a *personal information processor* for client data — to be confirmed), breach notification, Data Protection Officer, NPC registration requirements. (Texts and leads: research log LR-09–LR-11, LR-18.) |
| D-122 | Consent and opt-out obligations for marketing vs. transactional messages. (NTC broadcast messaging circulars: research log LR-06–LR-08.) |
| D-123 | Sender ID and anti-spam/anti-scam rules (regulatory and telco-imposed). |
| D-124 | Record-keeping and retention obligations (message logs, billing records, tax records). |
| D-125 | Requirements for serving government clients (procurement, security accreditation, data residency). |
| D-126 | Lawful requests from authorities for records: process and obligations. |
| D-127 | Tax and invoicing requirements (e.g. BIR rules for receipts/invoices). |
| D-128 | Implications of the SIM Registration Act (RA 11934) for sender identities, if any. |
| D-129 | Obligations for cross-border processing if hosting, backups, or service providers are outside the Philippines (Data Privacy Act Sec. 21 identified in public research; interpretation pending). |
| D-130 | Customer due diligence and acceptable-use obligations required by law (as distinct from business policy D-085, D-086). |
| D-131 | Whether the Anti-Financial Account Scamming Act (RA 12010) creates obligations or exposure for an SMS platform. (Research log LR-13.) |

## 9. Engineering decisions requiring owner approval

| ID | Decision | Notes |
| -- | -------- | ----- |
| D-140 | Selection of a headless accessible UI primitives library (if any) for complex interactions. | Direction in ADR-0013 (accepted 2026-11-25); evaluate maintenance, accessibility, licence, compatibility, footprint. **Owner decision 2026-11-25: deferred.** No package added until approved. |
| D-141 | Adoption of PostgreSQL Row-Level Security after evaluation. | ADR-0010 lists the evaluation requirements (pooling, transaction-scoped context, roles, performance, tests). |
| D-142 | Acceptance of each Proposed ADR. | See the review summary in the [decision log](../engineering/decision-log.md#5-review-summary-2026-11-24). |
| D-143 | **Public API key presentation and permission names**: `Authorization: Bearer <key>` vs. a custom header such as `X-API-Key`; permission names. | **Owner decision 2026-11-25 (PR #7 review):** Bearer API key authentication (`Authorization: Bearer <key>`) with the permissions `messages:send` and `messages:read`. Other API key design details remain in security baseline S-2/S-3. |
| D-144 | **Maximum message length**: maximum segments per message accepted by the public API. | **Owner decision 2026-11-25 (PR #7 review):** maximum API message length 1,600 characters. The maximum SMS segment count is **deferred** to product/provider policy (depends on D-105); above it the API returns `422 message-too-long`. |
| D-145 | **Public API and documentation domain**, which also fixes the RFC 9457 problem-type URI base. | Draft contract uses the placeholder `https://api.example.invalid`. **Owner decision 2026-11-25 (PR #7 review):** keep the placeholder for now; the final domain remains open. |
| D-146 | **HTTP status for insufficient credit** on message submission. | **Owner decision 2026-11-25 (PR #7 review):** `402` with problem type `insufficient-credit`; not recorded for idempotency (D-094c). |
| D-147 | **Sender ID on message submission**: explicit in every request, or an account-level default. | **Owner decision 2026-11-25 (PR #7 review):** `sender_id` is required in every request; no account-level default is assumed. Sender ID approval itself remains D-022. |
