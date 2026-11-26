# Questionnaire — Legal Counsel (Philippine telecommunications, data privacy, tax)

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — not yet used | Project owner | 2026-11-24 |

**Purpose:** obtain qualified advice on the regulatory path and obligations. Share the
[legal and regulatory research log](../legal-regulatory-research.md) with counsel as **background only**:
it lists public texts we found and could not access; it is not legal analysis.

**Respondents:** Philippine telecommunications/regulatory counsel; data-privacy counsel; a tax adviser for
Part E. Ask for **written** advice on P0 questions.

**Context to give counsel:** the proposed business (A2P SMS aggregation for businesses and government in
the Philippines), the candidate operating models (D-160), the corporate structure (D-161), the intended
customer segments, and the fact that the platform will store recipient phone numbers and message content
on behalf of customers.

## Part A — Telecommunications regulation

| # | Question | Register IDs | Research | P |
| - | -------- | ------------ | -------- | - |
| LC-01 | Under each candidate operating model (direct aggregator, reseller, via intermediary; **2026-11-26: only direct aggregation is pursued**, D-160), what is our regulatory classification (e.g. value-added service provider, data transmission industry participant under RA 12234, or other)? | D-120, D-160 | LR-01, LR-02 | P0 |
| LC-02 | Which NTC registrations or certificates must we hold before carrying commercial traffic? What are the steps, timelines, fees, and ongoing obligations? | D-120 | LR-02–LR-05, LR-16 | P0 |
| LC-12 | Is A2P SMS aggregation a "data transmission service" ("the conveyance of data offered on a commercial basis or to the public for a fee", IRR Rule I Sec. 4(n)) making us a DTIP, or a VAS outside data transmission, or both? If a DTIP, does any network segment (last mile, middle mile, core/backbone, gateway) apply to a platform that does not construct or operate network facilities? | D-120, D-160 | LR-02, LR-05 | P0 |
| LC-03 | What is the current status and effect of NTC MC 02-05-2008, the reported MO 005-06-2025, the RA 12234 implementing rules (including Rule X, Sec. 5(c) superseding prior registration rules), and the reported **NTC MC 002-02-2026**? Please **obtain and assess the signed official copy of MC 002-02-2026**, including its reported "modified VAS CoR" for VAS that is not data transmission, its eligibility criteria, and its transitional rules. | D-120 | LR-03–LR-05, LR-16, LR-17 | P0 |
| LC-04 | Do any facilities we may operate (VPN links, servers, connections to multiple telcos) amount to "putting up a network" or require additional authorization? | D-120, D-102 | LR-01, LR-02 | P1 |
| LC-05 | Are there regulatory restrictions on reselling to other aggregators or offering sub-accounts? | D-088, D-113 | — | P1 |
| LC-06 | Reconcile MC 03-03-2005, MC 04-07-2009, and MC 07-08-2018 and state what they require **separately** for OTP, transactional, promotional, survey, government advisory, and paid content/subscription messages: consent (including what "prior written consent" means in practice), push-message prohibition, opt-out mechanics and keywords, sender identification, time-of-day limits, and record retention (2 months vs. 6 months). Are we, our customers, or both the "content provider" bearing these obligations? Do not assume any exemption for transactional traffic. | D-122, D-123, D-024 | LR-06–LR-08 | P1 |
| LC-07 | What rules apply to sender IDs and to preventing impersonation, spoofing, and scam messages (including RA 11934)? What liability do we carry if a customer sends fraudulent messages? | D-123, D-128 | LR-12 | P1 |
| LC-08 | Under RA 12010 (AFASA): what exposure could a platform have under Sec. 5(a) (willful aiding or abetting) when a customer uses it as a "mass mailer" (Sec. 3(g), Sec. 4(c)(3))? What due diligence, abuse-prevention, and evidence-preservation measures would counsel recommend? What will bank and e-wallet customers require contractually? | D-131, D-130 | LR-13 | P2 |
| LC-09 | What due diligence (KYC) and acceptable-use obligations are required by law, as opposed to good practice? | D-130, D-085, D-086 | — | P1 |
| LC-10 | How must we respond to lawful requests from authorities for records? | D-126 | — | P2 |
| LC-11 | Are there specific requirements for serving government agencies (procurement, accreditation, data residency)? | D-125 | — | P2 |

## Part B — Data privacy (RA 10173)

| # | Question | Register IDs | Research | P |
| - | -------- | ------------ | -------- | - |
| LC-20 | For each data set — customer-uploaded contacts, message content, delivery receipts, customer user accounts, billing data — are we a personal information controller or processor? | D-121 | LR-09 | P0 |
| LC-21 | Must we register data processing systems and appoint a DPO with the NPC, and when? | D-121 | LR-11 | P0 |
| LC-22 | What must our customer contracts (data processing agreements) contain? | D-121 | LR-09, LR-10 | P1 |
| LC-23 | What are our breach-notification obligations and timelines, as processor and as controller, including the conditions under which the IRR's 72-hour notification applies? | D-121 | LR-09, LR-10 | P1 |
| LC-24 | What obligations apply if hosting, backups, or service providers are outside the Philippines? | D-129, D-068 | LR-09 | P1 |
| LC-25 | How long may or must we keep message content, delivery records, opt-out records, and audit logs? How do we handle erasure requests that conflict with retention duties? | D-124, D-073 | LR-09, LR-07, LR-08 | P1 |
| LC-26 | Would automated abuse detection, spam scoring, or sender-risk scoring on our platform be "automated decision-making or profiling" requiring notification or registration under NPC Circular 2022-04? | D-121 | LR-11 | P1 |

## Part C — Contracts and telcos

| # | Question | Register IDs | P |
| - | -------- | ------------ | - |
| LC-30 | Review telco agreements and term sheets for authorization requirements, liability, exclusivity, and termination risk. | D-113, D-002 | P0 |
| LC-31 | Review our customer terms and acceptable use policy. | D-086 | P1 |

## Part D — Corporate

| # | Question | Register IDs | P |
| - | -------- | ------------ | - |
| LC-40 | Is the proposed contracting entity suitable (ownership restrictions, licences held, foreign-ownership limits if any apply)? | D-161, D-047 | P0 |

## Part E — Tax and invoicing (tax adviser)

| # | Question | Register IDs | P |
| - | -------- | ------------ | - |
| LC-50 | What invoicing and receipting rules apply to prepaid credit sales and usage-based charges? | D-127, D-081 | P1 |
| LC-51 | How should taxes be shown on rates and invoices (gross vs. net), and does this affect how credits are denominated? | D-127, D-092 | P1 |
| LC-52 | What financial and tax records must be retained, and for how long? | D-124, D-127 | P1 |

**Evidence to request:** written opinions or memos (stored outside Git; reference in the
[evidence register](../evidence-register.md)), and copies of the official texts listed as "not accessed" in
the research log.
