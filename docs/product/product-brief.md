# Product Brief — Enterprise A2P SMS Platform

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — pending owner review; §1 and §4 updated for the EX-9 business-decisions review (2026-11-26) | Project owner | 2026-11-26 |

> This brief describes **intent**, not confirmed facts. Items requiring verification are tracked in the
> [discovery register](../requirements/discovery-register.md).

## 1. Business overview

The business intends to become an **independent A2P SMS aggregator in the Philippines**: a company that
lets many organizations send SMS to mobile subscribers through a single platform, using its own direct
connections to Philippine telecommunications providers.

```
 Client systems / portal users        Our platform                         Telcos            Recipients
 ┌────────────────────────┐    ┌──────────────────────────────┐    ┌──────────────────┐    ┌──────────┐
 │ Banks, retailers, LGUs │───▶│ Portal + API                 │───▶│ Smart SMSC       │───▶│ Mobile   │
 │ schools, logistics ... │    │ Billing, compliance, routing │───▶│ Globe SMSC       │───▶│ phones   │
 │                        │◀───│ Delivery reports, analytics  │◀───│ (others later)   │    │          │
 └────────────────────────┘    └──────────────────────────────┘    └──────────────────┘    └──────────┘
        requests / reports            our product                    delivery receipts
```

### Status of external relationships (as of this document)

| Party | Reported status | Verified? |
| ----- | --------------- | --------- |
| Smart Communications | Initial discussions reported | **No** |
| Globe Telecom | Initial discussions reported | **No** |
| PLDT Enterprise | Initial discussions reported | **No** — not a separate mobile network; a possible commercial or enterprise contracting counterpart; relationship to Smart (same group) to be clarified (D-112) |
| NTC | Initial discussions reported | **No** — required authorizations unknown |
| DITO Telecommunity | Not mentioned | **No.** Included in the coverage objective (Smart, Globe, DITO) on 2026-11-26 — owner-approved, pending partner ratification; not promised at launch (D-002, D-160) |

## 2. Problem and value proposition (to be validated)

Businesses need reliable, traceable, compliant SMS delivery to Philippine mobile subscribers. A direct
aggregator can potentially offer:

- **Reliability and speed** through direct telco connectivity (fewer intermediaries).
- **Transparency** through delivery reports and analytics.
- **Control** through self-service portal, API, sender ID management, and scheduling.
- **Compliance support** through consent records, opt-out handling, and audit trails.
- **Commercial flexibility** through client-specific pricing and credit management.

The competitive landscape (existing aggregators, pricing, differentiators) has not been analysed yet.

## 3. Intended customers

| Segment | Typical use cases | Notable requirements |
| ------- | ----------------- | -------------------- |
| Banks, fintech, e-wallets | OTPs, transaction alerts | Very low latency, high reliability, strict security, sender ID protection |
| Retail and e-commerce | Order updates, promotions | Campaign tools, opt-out handling, scheduling |
| Logistics | Delivery notifications | API integration, high volume, delivery reports |
| BPOs and enterprises | Customer notifications on behalf of their clients | Sub-accounts? (open question), reporting |
| Government agencies and LGUs | Advisories, emergency information | Procurement rules, possibly data residency, high burst traffic |
| Schools, healthcare | Reminders, announcements | Simple portal use, contact management |
| Resellers / other aggregators | Wholesale traffic | **Open question** whether reselling is allowed or desired |

## 4. Business model (to be confirmed)

| Element | Current assumption | Decision needed |
| ------- | ------------------ | --------------- |
| Charging unit | **Per message** (partner-approved direction, D-162). Owner proposals pending partner approval: billable quantity = segment count; charge settled on provider acceptance (D-080) | Partner approval of the billable unit and the charge event ([decision log §17](../engineering/decision-log.md#17-owner-decisions--business-decisions-review-phase-2-ex-9-2026-11-26)) |
| Payment model | Prepaid credits | Prepaid only, or postpaid for enterprise/government? |
| Credit unit | Owner proposal pending partner approval: balance in **PHP**, not SMS credits (D-081) | Partner approval; precision and rounding (EX-12); withdrawal or transfer of balance undecided |
| Pricing | Owner proposal pending partner approval: one versioned rate per client per billable unit; no customer pricing dimensions in the MVP (D-084) | Partner approval; list versus negotiated rates deferred; additional fees undecided (D-162) |
| Telco costs | Unknown | Telco rates, minimum commitments, deposits, settlement terms |
| Taxes | Unknown | VAT treatment, official receipts/invoicing requirements |

## 5. Initial product scope

The owner's initial capability list (see `AGENTS.md` §3.2) is treated as **product expectations, not an
SRS**. The proposed approach is to define a **Minimum Viable Product (MVP)** after discovery. A candidate
MVP (pending approval):

**Candidate MVP — in scope**
- Tenant onboarding (admin-created accounts at first, admin-controlled activation), users, roles, MFA for
  admins.
- Sender ID registry with manual admin approval; sender IDs approved before use.
- Single-message send via the client portal and the public API (one recipient per message). *Originally
  "single-message and bulk send via portal and public API"; bulk sending deferred by owner decision
  2026-11-25.* In the portal (owner decisions 2026-11-25 and 2026-11-26), the customer enters the mobile
  number manually, and one send operation accepts **exactly one recipient**: no pasted recipient
  lists, recipient-file uploads, contact-group sends, or multi-recipient selection. Customers may submit
  several separate single-recipient messages, subject to rate limits and abuse controls.
- Prepaid credit ledger with reservation and settlement; manual top-up by admin.
- Message processing with idempotency, routing to one or two providers, rate limiting, retries.
- Delivery receipt processing and per-message status.
- Basic delivery reports and CSV export.
- Opt-out/suppression list enforcement.
- Audit log of administrative, security, and financial actions.
- Abuse controls: tenant/API-key velocity limits, abuse alerts, and an operational kill switch
  (owner-confirmed essential, 2026-11-25).
- Simulator provider for development and demonstration.

**Candidate MVP — deferred**
- Self-service registration and online payments.
- Advanced analytics and dashboards (ClickHouse).
- **Bulk sending** (many recipients per send) through the public API and through the client portal,
  including uploaded recipient files — deferred to a later release (owner decision 2026-11-25; MVP-08,
  MVP-09).
- **Campaign approval workflows** (dedicated campaign-level approval by admins) — deferred beyond the MVP,
  which has no bulk sending or campaign-sending workflow (owner decision 2026-11-25; MVP-13). Essential
  abuse controls remain in the MVP (see above).
- **Saved contacts and contact groups** (contact book, groups, segmentation) — deferred from the MVP (owner
  decision 2026-11-26; MVP-11). Saved contacts may be introduced independently of groups in a later release.
- **Campaign scheduling** — deferred with campaign approval and bulk sending (owner decision 2026-11-26;
  MVP-12). A separate scheduled-single-message feature is not ruled out for a future release.
- Webhooks (unless early API customers require them).
- Two-way messaging / inbound keywords (depends on short code availability).
- Reseller/sub-account hierarchy.
- Automated failover routing across telcos.

## 6. Out of scope (until decided otherwise)

- International SMS (non-Philippine destinations).
- Other channels (voice, email, chat apps, RCS).
- Hosting or operating telco network equipment.

## 7. Success measures (to be defined)

Business and technical success metrics (delivery rate, latency for OTPs, uptime, cost per message,
customer onboarding time) will be defined with the owner once telco SLAs and target customers are known.

## 8. Key open questions

The most important open questions for the business are summarized in the
[discovery register](../requirements/discovery-register.md), especially:

1. What authorizations does the business need from the NTC (and others) to operate as an aggregator?
2. What do Smart, Globe, and PLDT actually offer: protocol, throughput, pricing, sender ID process?
3. What does the "5 billion monthly" figure measure, and how much is committed?
4. Who are the first 3–5 launch customers, and what do they need on day one?
