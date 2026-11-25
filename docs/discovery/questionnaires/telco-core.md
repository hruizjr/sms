# Questionnaire — Telco Common Core

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — not yet used | Project owner | 2026-11-24 |

**Use with** the provider-specific sheet: [Smart](telco-smart.md) · [Globe](telco-globe.md) ·
[PLDT Enterprise](telco-pldt-enterprise.md) · [DITO](telco-dito.md). The exhaustive technical list is the
[telco discovery checklist](../../integrations/telco-discovery-checklist.md) (§ numbers below refer to it);
this questionnaire is the meeting-ready subset.

**Respondents:** split the questions between the telco's **commercial/account** team (Part 1) and its
**technical/integration** team (Part 2). Do not expect sales staff to commit technical limits.

**Do not assume** every telco offers the same capabilities. Record each answer per telco.

## Part 1 — Commercial and onboarding (account / commercial team)

| # | Question | Register IDs | Checklist § | P |
| - | -------- | ------------ | ----------- | - |
| TC-01 | Do you offer direct A2P connectivity to aggregators like us? Under what kind of agreement? | D-002, D-118 | §19 | P0 |
| TC-02 | What authorization, registration, or documents do you require from an aggregator before signing (e.g. NTC registrations, company documents)? | D-113, D-120 | §18, §19 | P0 |
| TC-03 | May we carry traffic for multiple end customers? Must each end customer or sender ID be declared to you? Is reselling to other aggregators allowed? | D-113, D-088 | §19 | P0 |
| TC-04 | Will you deliver our messages to subscribers of **other** networks (off-net)? At what price and with what limits? | D-107, D-069 | §21 | P0 |
| TC-05 | What is your pricing model: per message or per segment; charged on submission or delivery; different rates by traffic type (OTP, transactional, promotional) or by volume? | D-108, D-041, D-080 | §15 | P0 |
| TC-06 | In what currency and with how many decimal places are rates quoted? How do you round on invoices, and how do you count segments for billing? | D-117, D-092 | §15 | P1 |
| TC-07 | Do you require a deposit, prepayment, credit limit, or minimum monthly commitment? | D-108, D-090 | §15 | P0 |
| TC-08 | What usage records (CDRs) do you provide, how often, and how are billing disputes reconciled? | D-108 | §15 | P1 |
| TC-09 | What throughput (messages per second) can you allocate to us at launch? How is it measured, and how can it be increased (process, lead time, cost)? | D-101 | §7 | P0 |
| TC-10 | What is the sender ID ("masking") registration process, lead time, and cost? Does registration with you carry over to other telcos? | D-103 | §9 | P1 |
| TC-11 | What content rules apply (URLs/links, prohibited content, promotional time restrictions)? What happens when rules are breached? | D-110, D-123 | §20 | P1 |
| TC-12 | What availability SLA, maintenance windows, support hours, and escalation contacts do you offer? | D-111 | §16 | P1 |
| TC-13 | What are the onboarding steps, typical duration, and any ramp-up limits after go-live? | D-109 | §18 | P1 |
| TC-14 | If your route is unavailable, may we send traffic for your subscribers through another telco, and may you route to others on our behalf? | D-119 | §21 | P1 |

## Part 2 — Technical (integration / engineering team)

| # | Question | Register IDs | Checklist § | P |
| - | -------- | ------------ | ----------- | - |
| TC-20 | Which protocols do you support for A2P submission (SMPP and version, HTTP API, other)? May we have the official interface specification? | D-100, D-040 | §1–§3 | P0 |
| TC-21 | How many simultaneous connections (binds) may we hold, of which types? From how many source IPs or sites? | D-101 | §8 | P0 |
| TC-22 | How must we connect: public internet with IP allowlisting, VPN, or private link? Static IPs required? Who provides VPN equipment? | D-102, D-044 | §5, §6 | P1 |
| TC-23 | Which encodings do you support (GSM-7, UCS-2), how do you handle concatenated messages, and what is the maximum number of segments? | D-105 | §10 | P1 |
| TC-24 | Do you return delivery receipts for all traffic, including off-net? In what format, per segment or per message, and with what typical and maximum delay? | D-104, D-116 | §11 | P1 |
| TC-25 | Is your message ID unique across connections and over time? When can an ID be reused? Can a receipt arrive before your submission response? Can two different final receipts arrive for one message? | D-114 | §11 | P1 |
| TC-26 | Please share the full list of error/status codes, which are temporary vs. permanent, and how throttling is signalled. | D-106 | §13 | P1 |
| TC-27 | If our submission times out and we don't know whether you accepted it, what do you recommend? Do you detect duplicate submissions? Is there a way to query a message's status? | D-115, D-095 | §14, §3 | P1 |
| TC-28 | How do you handle numbers ported from other networks? Is number-portability or network-lookup data available to us? | D-070 | §21 | P1 |
| TC-29 | Do you support inbound messages (replies, STOP keywords) and short codes or long numbers for two-way use? | D-063, D-024 | §12 | P1 |
| TC-30 | Is there a test environment? Do test messages reach real handsets? Is a certification or acceptance test required before production? | D-109 | §17 | P1 |
| TC-31 | What security requirements apply to us (credential rotation, encryption, audits), and how are security incidents escalated in both directions? | D-077 | §4, §16 | P2 |

**Evidence to request:** official technical specification (with version); draft commercial terms or rate
card; onboarding checklist; SLA document. Keep all of them outside Git; log references in the
[evidence register](../evidence-register.md).
