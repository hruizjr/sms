# Owner Discovery Workbook

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — Phase 2A; no item has been resolved by this workbook (owner decisions of 2026-11-25 and 2026-11-26 on the public API contract and MVP scope are noted below and recorded in the discovery register) | Project owner | 2026-11-26 |

This workbook lists **every open item** in the [discovery register](../requirements/discovery-register.md),
grouped by workstream, in plain language. For each item it says why it matters, what we know today, what
evidence would close it, who should answer, its priority, and what it blocks.

**Division of authority** ([`AGENTS.md` §1.1](../../AGENTS.md#11-document-authority-and-conflicts)):
the discovery register holds each item's **status and category**; this workbook holds its **routing**
(priority, respondent, evidence needed, questionnaire). Nothing here changes an item's status. An item is
resolved only when evidence is recorded in the [evidence register](evidence-register.md) and the register
is updated in a reviewed pull request.

## How to read the tables

- **Ask** — who can answer, linked to the questionnaire to use. "Owner" means it is your decision; you may
  want advice first.
- **P** — priority: **P0** blocks commercial feasibility, the regulatory path, or the architecture
  direction; **P1** needed for MVP scope and core design; **P2** after the initial architecture;
  **Future** not needed for the initial platform.
- **Blocks** — what cannot be finalized without the answer: *Commercial*, *Regulatory*, *Architecture*,
  *MVP scope*, *Implementation* (a specific feature), *Launch* (production go-live).
- Questionnaire links: [Partner](questionnaires/business-partner.md) ·
  [Telcos](questionnaires/telco-core.md) · [Customers](questionnaires/enterprise-customer.md) ·
  [Counsel](questionnaires/legal-counsel.md) · [Ops](questionnaires/operations.md).

---

## A. Business model and commercial structure

| ID | Question | Why it matters | Known today / options | Evidence that closes it | Ask | P | Blocks |
| -- | -------- | -------------- | --------------------- | ----------------------- | --- | - | ------ |
| D-160 | Which **operating model**: independent aggregator with direct telco contracts, reseller of another aggregator, connectivity through an intermediary, or a hybrid (start as reseller, go direct later)? | Decides who we sign with, which licences apply, how much we build, and our margin. | Owner intends direct aggregation (D-001, unverified). Options: direct · reseller/white-label · via intermediary (D-118) · hybrid. | Recorded owner decision, informed by counsel (D-120) and telco answers (D-113, D-118). | Owner, [Partner](questionnaires/business-partner.md), [Counsel](questionnaires/legal-counsel.md) | P0 | Commercial, Regulatory, Architecture |
| D-001 | Confirm the intent to operate as an independent aggregator and the business case for it. | Everything downstream assumes it. | Owner-stated only. | Business plan or partner/board minutes. | Owner, [Partner](questionnaires/business-partner.md) | P0 | Commercial |
| D-161 | **Who signs**: which legal entity contracts with telcos and customers; what role investors play; who holds decision rights. | Licences, contracts, and invoices are issued to an entity; investor rights affect decisions. | Unknown. D-047 assumes one entity. | Corporate registration and shareholder arrangements (reference only; documents stay outside Git). | Owner, [Partner](questionnaires/business-partner.md), [Counsel](questionnaires/legal-counsel.md) | P0 | Commercial, Regulatory |
| D-047 | Assumption: a single legal entity operates the platform. | Several entities would change billing and ledger design. | Assumption. | Same as D-161. | Owner | P0 | Architecture |
| D-162 | **Revenue model and target gross margin**: price per segment vs. telco cost; other fees (setup, sender ID, platform). | Determines viability and the pricing features to build. | Unknown. | Financial model reviewed by owner and accountant, using telco rate cards (D-108). | Owner, [Partner](questionnaires/business-partner.md) | P0 | Commercial |
| D-163 | **Commitments already made** to customers, investors, or telcos (dates, prices, volumes, exclusivity). | Hidden commitments constrain scope and schedule. | Unknown. | References to any LOIs, MOUs, or emails. | Owner, [Partner](questionnaires/business-partner.md) | P0 | Commercial, MVP scope |
| D-090 | **Budget** for build and operations (e.g. first 12–24 months). | Limits infrastructure, staffing, and redundancy. | Unknown. | Approved budget range. | Owner, [Partner](questionnaires/business-partner.md) | P0 | Commercial, Architecture |
| D-084 | Customer **pricing structure**: per network, per traffic type (OTP vs. promotional), volume tiers? | Competitiveness and pricing-engine complexity. | Unknown. | Owner decision; market price evidence from customers. | Owner, [Customers](questionnaires/enterprise-customer.md) | P0 | Commercial, Implementation |
| D-080 | Charge customers on **acceptance, submission, or delivery**? | If telcos charge per submission and we charge per delivery, we pay for undelivered messages (risk R-08). | Options: accepted · submitted · delivered · other agreed event. Depends on D-041, D-108. | Owner decision after telco billing basis is known. | Owner | P0 | Commercial, Architecture |
| D-081 | Credit unit: **peso balance or "SMS credits"**? | Ledger representation (ADR-0019), customer clarity, tax treatment. | Options: pesos · credits · both. | Owner decision with accountant. | Owner | P1 | Architecture |
| D-082 | **Prepaid only**, or postpaid with credit limits for some customers? | Postpaid needs credit control, invoicing, collections. | D-045 assumes prepaid. | Owner decision; customer expectations. | Owner, [Customers](questionnaires/enterprise-customer.md) | P1 | MVP scope |
| D-045 | Assumption: prepaid credit at launch. | As D-082. | Assumption. | As D-082. | Owner | P1 | MVP scope |
| D-083 | **Refund policy** for failed or undelivered messages. | Ledger release rules; customer trust. | Depends on D-080 and D-096. | Owner decision. | Owner | P1 | Implementation |
| D-088 | Allow **reseller / wholesale** customers? | Tenant model; may need telco permission (D-113) and has regulatory exposure. | Unknown. | Owner decision informed by D-113 and counsel. | Owner, [Telcos](questionnaires/telco-core.md), [Counsel](questionnaires/legal-counsel.md) | P1 | Architecture |
| D-092 | **Monetary precision**: currency and decimal places of real wholesale and retail rates. | Determines the ledger storage format (ADR-0019). | Open. Centavos may be insufficient. | Telco rate cards; our price list. | [Telcos](questionnaires/telco-core.md), Owner | P1 | Architecture |
| D-093 | **Rounding rules** (where, which mode, residue). | Estimate/charge consistency; reconciliation with telco invoices. | Open. | Owner decision with accountant; telco invoice rules (D-117). | Owner | P1 | Implementation |

## B. Traffic and customer demand

Use the [traffic evidence worksheet](traffic-evidence-worksheet.md) to capture answers; the questions
themselves are defined in [traffic capacity discovery](../requirements/traffic-capacity-discovery.md).

| ID | Question | Why it matters | Known today / options | Evidence that closes it | Ask | P | Blocks |
| -- | -------- | -------------- | --------------------- | ----------------------- | --- | - | ------ |
| D-004 | What does **"~5 billion monthly"** count? | The same number can mean loads differing by more than 10×. | Unit unknown. Candidates: API requests · recipients · billable segments · submission attempts · delivered messages · totals across customers or providers · an annual figure. | The source document or spreadsheet and a written explanation from whoever produced it. | Originator via [Partner](questionnaires/business-partner.md) | P0 | Commercial, Architecture |
| D-060 | How much is **committed vs. projected**? | Capacity and telco negotiations should rest on committed volume. | Unknown. | Signed contracts or LOIs stating volumes; historical volume reports from current providers. | [Partner](questionnaires/business-partner.md), [Customers](questionnaires/enterprise-customer.md) | P0 | Commercial |
| D-061 | Who are the **launch customers** and what are their use cases? | MVP scope and priorities depend on real customers. | Unknown. | Named prospects with written statements of intent. | Owner, [Customers](questionnaires/enterprise-customer.md) | P0 | MVP scope, Commercial |
| D-062 | **Peak** messages per second and **burst** patterns. | Capacity design and telco throughput negotiation. | Unknown. | Customer estimates or hourly history from current providers. | [Customers](questionnaires/enterprise-customer.md) | P1 | Architecture |
| D-074 | **Ramp-up** by month after launch. | Capacity planning and cash flow. | Unknown. | Customer onboarding schedules. | [Customers](questionnaires/enterprise-customer.md), Owner | P1 | Architecture, Commercial |
| D-075 | Message **length and encoding mix** (single vs. multipart; share needing UCS-2). | Segments drive both cost and telco throughput. | Unknown. | Redacted/synthetic templates; segment statistics from current providers. | [Customers](questionnaires/enterprise-customer.md) | P1 | Commercial, Architecture |
| D-042 | Assumption: traffic is mostly MT; MO limited to replies/opt-outs. | Inbound design and number provisioning. | Assumption. | Customer requirements for replies. | [Customers](questionnaires/enterprise-customer.md) | P1 | Architecture |
| D-063 | Are **two-way messaging, keywords, or short codes** needed? | Requires inbound numbers and MO processing. | Unknown. | Customer requirements; telco availability (checklist §12). | [Customers](questionnaires/enterprise-customer.md), [Telcos](questionnaires/telco-core.md) | P1 | MVP scope |
| D-043 | Assumption: Philippine destinations only. | International routing is a different business. | Assumption. | Customer destination mix. | [Customers](questionnaires/enterprise-customer.md) | P1 | Architecture, Regulatory |
| D-071 | Contacts per tenant and import sizes. | Import and storage design. | Unknown. | Customer estimates. | [Customers](questionnaires/enterprise-customer.md) | P2 | Implementation |

## C. Telco connectivity and technical discovery

Technical detail lives in the [telco discovery checklist](../integrations/telco-discovery-checklist.md);
meeting questions are in the [telco questionnaires](questionnaires/telco-core.md).

| ID | Question | Why it matters | Known today / options | Evidence that closes it | Ask | P | Blocks |
| -- | -------- | -------------- | --------------------- | ----------------------- | --- | - | ------ |
| D-002 | Which telcos will actually contract with us? | No telco, no business. | Owner-stated targets: Smart, Globe, PLDT Enterprise. | Written engagement (NDA, term sheet, or letter). | [Telcos](questionnaires/telco-core.md) | P0 | Commercial |
| D-003 | What happened in the reported discussions with Smart, Globe, PLDT, NTC? | Avoid restarting or contradicting earlier conversations. | Reported, unrecorded. | Meeting notes: date, attendees, roles, outcomes. | Owner | P0 | Commercial |
| D-112 | Are **PLDT Enterprise and Smart** one counterparty/route or separate? | Determines how many contracts and integrations we need. | Reportedly the same corporate group — unverified lead (research log LR-15); technical relationship unknown. | Written statement from PLDT Enterprise / Smart. | [Smart](questionnaires/telco-smart.md), [PLDT Ent.](questionnaires/telco-pldt-enterprise.md) | P0 | Commercial, Architecture |
| D-118 | **Direct** telco connectivity, or through an **intermediary** aggregator/connectivity provider (interim or permanent)? | Changes protocol, margin, control, and regulatory position. | Unknown. | Telco answers on direct aggregator access; intermediary offers. | [Telcos](questionnaires/telco-core.md), [Intermediary](questionnaires/telco-intermediary.md) | P0 | Commercial, Architecture |
| D-113 | What **aggregator/reseller authorization** does each telco require? | We may not be allowed to carry third-party traffic without it. | Unknown. | Written telco requirements; contract terms. | [Telcos](questionnaires/telco-core.md) | P0 | Commercial, Regulatory |
| D-100 | Which **protocols** and versions (SMPP, HTTP APIs, other)? | Adapter design. | Unknown. | Official technical specification. | [Telcos](questionnaires/telco-core.md) | P0 | Architecture |
| D-040 | Assumption: SMPP. | Adapter design changes otherwise. | Assumption. | As D-100. | [Telcos](questionnaires/telco-core.md) | P1 | Architecture |
| D-101 | **Throughput** (messages/second) per account and per connection; number of connections (binds). | Must cover verified peak demand (D-062). | Unknown. | Written allocation in a term sheet or specification. | [Telcos](questionnaires/telco-core.md) | P0 | Commercial, Architecture |
| D-107 | Does each telco accept traffic to **other networks' subscribers** (off-net)? | Without it we need every network directly. | Unknown. | Written telco terms. | [Telcos](questionnaires/telco-core.md) | P0 | Commercial, Architecture |
| D-108 | **Pricing, billing basis**, CDR access, settlement and reconciliation. | Margin and billing design. | Unknown. | Rate card; draft commercial terms. | [Telcos](questionnaires/telco-core.md) | P0 | Commercial |
| D-041 | Assumption: telcos charge per segment. | Billing model and margin. | Assumption. | As D-108. | [Telcos](questionnaires/telco-core.md) | P0 | Commercial |
| D-069 | Must we reach **DITO** subscribers, and by which route? | Messages to unreachable networks fail. | Not in scope yet. | Customer coverage expectation; telco off-net answers; DITO offer. | [Customers](questionnaires/enterprise-customer.md), [DITO](questionnaires/telco-dito.md) | P1 | Commercial, Architecture |
| D-070 | How do we determine a recipient's **current network** given number portability? | Routing correctness and cost. | Portability is law (RA 11202, research log LR-14). Method unknown. | Telco answers (checklist §21); MNP data availability. | [Telcos](questionnaires/telco-core.md) | P1 | Architecture |
| D-119 | **Failover**: may traffic for one network go via another telco if a route fails? | Resilience and cost. | Unknown. | Telco commercial and technical terms. | [Telcos](questionnaires/telco-core.md) | P1 | Architecture |
| D-102 | Network connectivity: internet with IP allowlisting, VPN, or private link? | Constrains hosting (D-044, D-068). | Unknown. | Technical specification. | [Telcos](questionnaires/telco-core.md) | P1 | Architecture |
| D-103 | **Sender ID** registration process and lead time. | Customers cannot send branded messages without it. | Unknown. | Written process. | [Telcos](questionnaires/telco-core.md) | P1 | Launch |
| D-104 | **Delivery receipt** support, format, timing, status meanings. | Status model and billing (ADR-0021). | Unknown. | Specification. | [Telcos](questionnaires/telco-core.md) | P1 | Architecture |
| D-105 | **Encoding** and segmentation support. | Cost estimates and correct display. | Unknown. | Specification. | [Telcos](questionnaires/telco-core.md) | P1 | Implementation |
| D-106 | **Error codes**; retryable vs. permanent; throttling. | Retry safety (no duplicate SMS). | Unknown. | Specification. | [Telcos](questionnaires/telco-core.md) | P1 | Implementation |
| D-109 | **Test environment** and production onboarding. | We cannot verify integration without it. | Unknown. | Written onboarding plan; test access. | [Telcos](questionnaires/telco-core.md) | P1 | Launch |
| D-110 | **Content restrictions** (URLs, keywords) and traffic classification. | Customers' messages may be blocked. | Unknown. | Written policy. | [Telcos](questionnaires/telco-core.md) | P1 | MVP scope |
| D-111 | **SLA**, maintenance windows, escalation contacts. | Our own customer SLA depends on it. | Unknown. | SLA document. | [Telcos](questionnaires/telco-core.md) | P1 | Launch |
| D-114 | Provider **message ID** format, uniqueness, reuse. | Receipt correlation. | Unknown. | Specification. | [Telcos](questionnaires/telco-core.md) | P1 | Implementation |
| D-115 | Can we **query message status** or rely on telco **duplicate detection**? | Resolving ambiguous submissions without duplicates (D-095). | Unknown. | Specification. | [Telcos](questionnaires/telco-core.md) | P1 | Architecture |
| D-117 | Rate **decimal places**; telco rounding and segment counting on invoices. | Ledger precision (D-092) and reconciliation. | Unknown. | Rate card; sample invoice. | [Telcos](questionnaires/telco-core.md) | P1 | Architecture |
| D-116 | Typical and maximum **receipt delay**. | Sets the `UNKNOWN` window per provider. | Unknown. | Specification; test-environment observations. | [Telcos](questionnaires/telco-core.md) | P2 | Implementation |

## D. Legal and regulatory

Questions for counsel are in the [legal counsel questionnaire](questionnaires/legal-counsel.md); public
sources already consulted are in the [research log](legal-regulatory-research.md). **Nothing in this
repository is legal advice.**

| ID | Question | Why it matters | Known today / options | Evidence that closes it | Ask | P | Blocks |
| -- | -------- | -------------- | --------------------- | ----------------------- | --- | - | ------ |
| D-120 | Which **NTC registrations / authorizations** does the chosen model need (VAS registration; effect of RA 12234 "DTIP" registration; anything else)? | Operating without the right registration risks sanctions and blocks telco onboarding. | Texts read from unofficial full-text copies: 2008 VAS circular (SMS classified as VAS; Certificate of Registration), RA 12234 and its IRR (DTIP registration vs. authorization). Reported 2026 DTIP circular unverified. Research log LR-01–LR-05, LR-16. **Our classification is not determined.** | Counsel's written opinion; NTC certificates once obtained. | [Counsel](questionnaires/legal-counsel.md) | P0 | Regulatory, Commercial |
| D-121 | **Data Privacy Act** obligations: our role (controller or processor) per data type, NPC registration, DPO, breach notification. | Legal compliance; contract terms with customers. | Act text read (LR-09). DPA IRR not read (LR-10). NPC Circular 2022-04 known from official register metadata only (LR-11). Role, registration criteria, and breach timing unconfirmed. | Counsel's written opinion. | [Counsel](questionnaires/legal-counsel.md) | P0 | Regulatory, Architecture |
| D-129 | **Cross-border** processing: obligations if hosting or service providers are outside the Philippines. | Hosting choice (D-068). | DPA Sec. 21 accountability identified (LR-09); interpretation pending. | Counsel's opinion. | [Counsel](questionnaires/legal-counsel.md) | P1 | Architecture |
| D-122 | **Consent and opt-out** rules for promotional vs. transactional messages (including NTC broadcast messaging rules). | Product features (opt-out keywords, records) and customer obligations. | Circulars read from unofficial full-text copies (LR-06–LR-08): prior consent / opt-in; prior written consent for promotional broadcasts; push-message prohibition for content providers; opt-out and record rules for paid content. OTP/transactional treatment not addressed in the texts. | Counsel's opinion on applicability to A2P aggregation. | [Counsel](questionnaires/legal-counsel.md) | P1 | MVP scope, Implementation |
| D-123 | **Sender ID, anti-spam, anti-scam** rules (regulatory and telco-imposed). | Abuse controls; liability. | SIM Registration Act spoofing provisions identified (LR-12). | Counsel's opinion; telco policy (D-110). | [Counsel](questionnaires/legal-counsel.md), [Telcos](questionnaires/telco-core.md) | P1 | MVP scope |
| D-130 | Legally required **customer due diligence** and acceptable-use obligations. | Onboarding workflow (D-085, D-086). | Unknown. | Counsel's opinion. | [Counsel](questionnaires/legal-counsel.md) | P1 | MVP scope |
| D-124 | **Record-keeping and retention** obligations (messages, billing, tax). | Retention schedule and storage cost. | DPA retention principle identified (LR-09); specific periods unknown. | Counsel's opinion; tax adviser. | [Counsel](questionnaires/legal-counsel.md) | P1 | Architecture |
| D-073 | Must full **message content** be stored? For how long? Who may see it? | Privacy, storage, and support tooling. | Unknown. | Owner decision informed by D-124 and customer needs. | Owner, [Counsel](questionnaires/legal-counsel.md) | P1 | Architecture |
| D-127 | **Tax and invoicing** requirements. | Billing documents and ledger fields. | Unknown. | Tax adviser's guidance. | Tax adviser via [Counsel](questionnaires/legal-counsel.md) | P1 | Implementation |
| D-024 | Proposed: platform-wide opt-out/suppression enforced before dispatch. | Recipient protection; likely legal expectation. | Proposed requirement. | Owner approval, informed by D-122. | Owner, [Counsel](questionnaires/legal-counsel.md) | P1 | MVP scope |
| D-125 | Requirements for **government** clients (procurement, accreditation, data residency). | Only if government customers are targeted at launch. | Unknown. | Counsel's opinion; tender documents. | [Counsel](questionnaires/legal-counsel.md) | P2 | Commercial |
| D-126 | Handling **lawful requests** from authorities. | Process and data access controls. | Unknown. | Counsel's opinion. | [Counsel](questionnaires/legal-counsel.md) | P2 | Launch |
| D-128 | Implications of the **SIM Registration Act** for sender identities. | Sender ID policy. | Act text identified (LR-12); applicability to aggregators not stated in the text. | Counsel's opinion. | [Counsel](questionnaires/legal-counsel.md) | P2 | MVP scope |
| D-131 | Does the **Anti-Financial Account Scamming Act** create obligations or risks for us (e.g. its "mass mailer" definition)? | Banks and e-wallets are likely customers. | Act read (LR-13): institution duties fall on BSP-supervised entities; using a mass mailer (≥ 50 recipients) makes scamming offenses economic sabotage; willful aiding or abetting is an offense. Platform exposure not concluded. | Counsel's opinion. | [Counsel](questionnaires/legal-counsel.md) | P2 | Regulatory |

## E. Product scope and MVP

Use the [MVP scope worksheet](mvp-scope-worksheet.md) to classify capabilities.

| ID | Question | Why it matters | Known today / options | Evidence that closes it | Ask | P | Blocks |
| -- | -------- | -------------- | --------------------- | ----------------------- | --- | - | ------ |
| D-089 | **MVP scope and launch customers** — what launches first? | Phase 2 exit criterion; drives everything after. | Candidate list in the product brief and MVP worksheet. **Partly decided 2026-11-25 (PR #7 review):** single-message sending through the portal and the public API; bulk sending through both channels deferred to a later release (MVP-08, MVP-09); one recipient per portal send; campaign approval workflow (MVP-13) deferred; essential abuse controls confirmed. **2026-11-26:** saved contacts and contact groups (MVP-11) and campaign scheduling (MVP-12) deferred. | Owner's written approval of the worksheet. | Owner, [Customers](questionnaires/enterprise-customer.md) | P0 | MVP scope |
| D-020 | Proposed: define an MVP before building features. | Same as D-089. | Proposed. | Same as D-089. | Owner | P0 | MVP scope |
| D-007 | Original capability list treated as expectations, not requirements. | Prevents building everything at once. | Owner-stated. | MVP worksheet approval. | Owner | P1 | MVP scope |
| D-064 | **Sub-accounts / hierarchies** (e.g. BPO with several clients)? | Tenant data model. | Unknown. | Customer requirements. | [Customers](questionnaires/enterprise-customer.md) | P1 | Architecture |
| D-085 | Customer **onboarding / KYC** requirements. | Abuse prevention; onboarding effort. | Unknown; legal input D-130. | Owner policy. | Owner | P1 | MVP scope |
| D-086 | **Acceptable use policy**. | Abuse prevention; contract terms. | Unknown. | Owner policy reviewed by counsel. | Owner, [Counsel](questionnaires/legal-counsel.md) | P1 | MVP scope |
| D-087 | **Campaign approval** policy. | Admin workload; abuse prevention. | **Workflow deferred from the MVP 2026-11-25 (PR #7 review; MVP-13).** Policy not decided: thresholds, first-campaign policies, and turnaround remain open for a later release. | Owner policy before campaigns are scheduled. | Owner | P2 | Implementation |
| D-025 | Proposed: approval for large campaigns or new tenants. | As D-087. | Proposed. **Workflow deferred from the MVP 2026-11-25 (MVP-13)**; the proposal itself remains open. | Owner approval before campaigns are scheduled. | Owner | P2 | Implementation |
| D-022 | Proposed: sender IDs approved before use. | Impersonation risk. | Proposed. | Owner approval; telco process (D-103). | Owner | P1 | MVP scope |
| D-021 | Proposed: MFA mandatory for Super Admins. | Admins control money and routing. | Proposed. | Owner approval. | Owner | P1 | Implementation |
| D-023 | Proposed: idempotency keys on message submission. | Prevents duplicate operations. | **Decided 2026-11-25 (PR #7 review):** approved; key required (D-094a). **ADR-0018 accepted 2026-11-25.** | Owner decision — recorded. | Owner | P1 | Architecture |
| D-094 | Idempotency details (required?, scope, retention, bulk). | API contract. | **Decided 2026-11-25 (PR #7 review):** required; scope account + operation; insufficient-credit rejections not recorded; retention seven days; bulk sending through the API and the portal deferred to a later release (MVP-08, MVP-09). **ADR-0018 accepted 2026-11-25.** | Owner decision — recorded. | Owner, [Customers](questionnaires/enterprise-customer.md) | P1 | Architecture |
| D-143 | How API keys are sent, and permission names. | Public API contract. | **Decided 2026-11-25 (PR #7 review):** `Authorization: Bearer <key>`; permissions `messages:send`, `messages:read`. | Owner decision — recorded. | Owner | P1 | Architecture |
| D-144 | Maximum message length and segment count. | Public API contract; cost surprises. | **Decided 2026-11-25 (PR #7 review):** API maximum 1,600 characters. **Segment count deferred** to product/provider policy (depends on D-105). | Segment count: owner/product policy once provider limits are known. | Owner, [Telcos](questionnaires/telco-core.md) | P1 | Implementation |
| D-145 | Public API and documentation **domain**. | Public API contract; problem-type URIs. | **Owner decision 2026-11-25 (PR #7 review):** keep the placeholder for now. | Owner decision when the domain is chosen. | Owner | P2 | Launch |
| D-146 | HTTP status for insufficient credit. | Public API contract. | **Decided 2026-11-25 (PR #7 review):** `402`. | Owner decision — recorded. | Owner | P1 | Implementation |
| D-147 | Explicit sender ID in every request, or an account default? | Public API contract; sender ID control. | **Decided 2026-11-25 (PR #7 review):** explicit `sender_id` required; no account-level default. | Owner decision — recorded. | Owner | P1 | Implementation |
| D-026 | Proposed: simulator provider before any real telco adapter. | Development without telco access. | Proposed. | Owner approval. | Owner | P1 | Implementation |
| D-027 | Proposed: admin portal deployed separately. | Security. | Proposed (ADR-0013). | Owner approval (with D-091). | Owner | P1 | Architecture |
| D-091 | One portal or **two** (client / admin)? | Security vs. simplicity. | **Decided 2026-11-25 (ADR-0013):** one application, `client-portal`, initially, plus `packages/ui`; `admin-portal` added when admin functionality begins. | Owner decision — recorded. | Owner | P1 | Architecture |
| D-140 | Which **headless UI library**, if any? | Accessibility of complex widgets. | Direction in ADR-0013 (accepted). **Deferred by the owner on 2026-11-25.** | Owner decision when complex widgets are needed. | Owner | P1 | Implementation |
| D-029 | Proposed: show segments and cost before sending. | Avoids cost surprises. | Proposed. | Owner approval. | Owner | P1 | MVP scope |
| D-095 | Policy for **ambiguous submissions** (timeouts). | Loss vs. duplicate SMS trade-off. | Options in [messaging reliability §4](../architecture/messaging-reliability.md#4-provider-submission-and-ambiguous-outcomes). Depends on D-115. | Owner decision per traffic type. | Owner | P1 | Architecture |
| D-096 | Customer label and billing for **uncertain** messages. | Customer trust and margin. | **Label decided 2026-11-25 (PR #7 review):** `submission_uncertain` is a separate status (and `unknown` is shown as `unknown`). **Billing of `unknown` and other uncertain messages still open** (separate decision, not settled by ADR-0021). | Owner decision on billing treatment. | Owner | P1 | Architecture |
| D-099 | **Customer-visible statuses** and webhook semantics. | Public API contract. | **Statuses decided 2026-11-25 (PR #7 review):** eight statuses and ten initial reason codes, extensible (public API contract §3). **ADR-0021 accepted 2026-11-25, limited to these statuses and reason codes.** **Webhook semantics still open** (separate decision, not settled by ADR-0021). | Owner decision on webhook semantics when webhooks are scheduled. | Owner, [Customers](questionnaires/enterprise-customer.md) | P1 | Architecture |
| D-097 | Multi-segment receipt roll-up and partial billing. | Billing correctness. | Depends on D-104. | Owner decision after telco specs. | Owner | P2 | Implementation |
| D-098 | Conflicting final receipts policy. | Reconciliation. | Open. | Owner decision. | Owner | P2 | Implementation |
| D-065 | Required **integrations** (payments, accounting, CRM). | Scope. | Unknown. | Customer and owner needs. | Owner, [Customers](questionnaires/enterprise-customer.md) | P2 | MVP scope |
| D-072 | **Analytics and reporting** needs. | Whether an analytics store is needed (ADR-0008). | Unknown. | Customer requirements. | [Customers](questionnaires/enterprise-customer.md) | P2 | Architecture |
| D-028 | Proposed: UTC storage, Philippine-time display. | Consistency. | Proposed. | Owner approval. | Owner | P2 | Implementation |
| D-046 | Assumption: English UI at launch. | Localization effort. | Assumption. | Customer feedback. | [Customers](questionnaires/enterprise-customer.md) | P2 | Implementation |

## F. Operations and security

| ID | Question | Why it matters | Known today / options | Evidence that closes it | Ask | P | Blocks |
| -- | -------- | -------------- | --------------------- | ----------------------- | --- | - | ------ |
| D-066 | Who runs the platform **24/7** (monitoring, on-call)? | OTP failures at night are invisible without it (R-40). | Unknown. | Staffing plan or outsourcing agreement. | Owner, [Ops](questionnaires/operations.md) | P1 | Launch |
| D-067 | Availability and support **SLAs** we offer customers. | Redundancy, cost, contracts. | Unknown; bounded by telco SLAs (D-111). | Owner decision; customer expectations. | Owner, [Customers](questionnaires/enterprise-customer.md) | P1 | Architecture |
| D-068 | **Hosting** model and **data-residency** expectations. | Telco connectivity, privacy, cost. | Unknown. | Owner decision informed by D-102, D-129, customer requirements. | Owner, [Ops](questionnaires/operations.md), [Customers](questionnaires/enterprise-customer.md) | P1 | Architecture |
| D-044 | Assumption: hosting can meet telco connectivity needs (static IPs, VPN). | Hosting choice. | Assumption. | Telco specs (D-102) vs. hosting offers. | [Ops](questionnaires/operations.md), [Telcos](questionnaires/telco-core.md) | P1 | Architecture |
| D-076 | **Backup and disaster-recovery** targets (how much data loss and downtime are acceptable). | Database replication and cost. | Unknown (architecture question AQ-8). | Owner decision. | Owner, [Ops](questionnaires/operations.md) | P1 | Architecture |
| D-078 | Who reviews **abuse alerts, sender IDs, and campaign approvals** day to day? | Manual approval workflows need staff. | Unknown. Campaign approvals not needed in the MVP (MVP-13 deferred, 2026-11-25). | Staffing plan. | Owner, [Ops](questionnaires/operations.md) | P1 | MVP scope |
| D-077 | **Security responsibility split** and incident escalation between us, telcos, and customers. | Incident handling and contracts. | Unknown. | Telco and customer contract terms; owner policy. | [Ops](questionnaires/operations.md), [Telcos](questionnaires/telco-core.md) | P2 | Launch |
| D-079 | **Customer support** model (hours, channels, tiers). | Staffing and tooling. | Unknown. | Owner decision. | Owner, [Ops](questionnaires/operations.md) | P2 | Launch |
| D-141 | Adopt PostgreSQL Row-Level Security? | Defense-in-depth for tenant isolation. | Evaluation planned in Phase 3 (ADR-0010). | Evaluation results; owner decision. | Owner | P2 | Architecture |

## G. Items needing no external discovery

| ID | Reason |
| -- | ------ |
| D-005 | Technology preferences are resolved through ADR decisions — see [decision readiness](decision-readiness.md). |
| D-006 | Monorepo — already implemented. |
| D-008 | AI-assisted workflow — process, already documented. |
| D-142 | ADR acceptance — tracked in [decision readiness](decision-readiness.md). |

## H. P0 summary (the critical questions)

D-001, D-002, D-003, D-004, D-020, D-041, D-047, D-060, D-061, D-080, D-084, D-089, D-090, D-100, D-101,
D-107, D-108, D-112, D-113, D-118, D-120, D-121, D-160, D-161, D-162, D-163.

Recommended order: see the [critical path](phase2-discovery-guide.md#7-critical-path-and-who-to-contact-first).
