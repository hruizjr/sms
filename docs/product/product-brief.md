# Product Brief — Enterprise A2P SMS Platform

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — pending owner review | Project owner | 2026-11-24 |

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
| PLDT Enterprise | Initial discussions reported | **No** — relationship to Smart (same group) to be clarified |
| NTC | Initial discussions reported | **No** — required authorizations unknown |
| DITO Telecommunity | Not mentioned | Not in scope; see discovery register |

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
| Charging unit | Per segment | Per segment vs. per message; charge on submission vs. delivery |
| Payment model | Prepaid credits | Prepaid only, or postpaid for enterprise/government? |
| Credit unit | Not decided | Peso-denominated balance vs. "SMS credits" |
| Pricing | Client-specific price per segment, possibly per destination network | Volume tiers? Different rates for OTP vs. marketing? |
| Telco costs | Unknown | Telco rates, minimum commitments, deposits, settlement terms |
| Taxes | Unknown | VAT treatment, official receipts/invoicing requirements |

## 5. Initial product scope

The owner's initial capability list (see `AGENTS.md` §3.2) is treated as **product expectations, not an
SRS**. The proposed approach is to define a **Minimum Viable Product (MVP)** after discovery. A candidate
MVP (pending approval):

**Candidate MVP — in scope**
- Tenant onboarding (admin-created accounts at first), users, roles, MFA for admins.
- Sender ID registry with manual admin approval.
- Single-message and bulk send via portal and public API.
- Prepaid credit ledger with reservation and settlement; manual top-up by admin.
- Message processing with idempotency, routing to one or two providers, rate limiting, retries.
- Delivery receipt processing and per-message status.
- Basic delivery reports and CSV export.
- Opt-out/suppression list enforcement.
- Audit log of admin and financial actions.
- Simulator provider for development and demonstration.

**Candidate MVP — deferred**
- Self-service registration and online payments.
- Advanced analytics and dashboards (ClickHouse).
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
