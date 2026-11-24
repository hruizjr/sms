# Discovery Register

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — living document | Project owner | 2026-11-24 |

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
| D-023 | Public API supports idempotency keys on message submission. | Prevents duplicate sends and charges on client retries. |
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

## 6. Business decisions required

| ID | Decision | Options / notes |
| -- | -------- | --------------- |
| D-080 | Charge on submission or on delivery? | Delivery-based is friendlier to clients but exposes us to telco charges for undelivered messages. Depends on telco billing basis. |
| D-081 | Credit unit: pesos or SMS credits? Precision of prices (e.g. fractions of a centavo)? | Affects ledger design (ADR-0011). |
| D-082 | Prepaid only, or postpaid for some customers? | Credit limits, invoicing, collections. |
| D-083 | Refund policy for failed/undelivered messages. | Ledger release rules. |
| D-084 | Pricing structure: per network, per message type, volume tiers? | Pricing engine complexity. |
| D-085 | Customer onboarding / KYC requirements (business documents, verification). | Onboarding workflow. |
| D-086 | Acceptable use policy: prohibited content and industries. | Compliance and abuse prevention. |
| D-087 | Campaign approval policy (which campaigns need manual approval). | Admin workflow. |
| D-088 | Whether to allow reseller/wholesale customers. | Tenant model, regulatory exposure. |
| D-089 | MVP scope and launch customers. | Roadmap. |
| D-090 | Budget envelope for infrastructure and operations. | Technology choices. |
| D-091 | Single vs. separate admin and client portals (ADR-0013). | Security vs. simplicity. |

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

## 8. Regulatory-dependent requirements

These require advice from qualified Philippine telecommunications and data-privacy counsel. Nothing here
is legal advice.

| ID | Item |
| -- | ---- |
| D-120 | Which NTC registrations, licences, or authorizations are required to operate as an A2P aggregator / value-added service provider, and current status of each. Confirm the current legal framework, including any recent legislative changes. |
| D-121 | Data Privacy Act of 2012 (RA 10173) obligations: lawful basis for processing recipient data, roles (we are likely a *personal information processor* for client data — to be confirmed), breach notification, Data Protection Officer, NPC registration requirements. |
| D-122 | Consent and opt-out obligations for marketing vs. transactional messages. |
| D-123 | Sender ID and anti-spam/anti-scam rules (regulatory and telco-imposed). |
| D-124 | Record-keeping and retention obligations (message logs, billing records, tax records). |
| D-125 | Requirements for serving government clients (procurement, security accreditation, data residency). |
| D-126 | Lawful requests from authorities for records: process and obligations. |
| D-127 | Tax and invoicing requirements (e.g. BIR rules for receipts/invoices). |
| D-128 | Implications of the SIM Registration Act (RA 11934) for sender identities, if any. |
