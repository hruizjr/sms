# Phase 2 Discovery Guide

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — Phase 2A preparation; **discovery has not started** | Project owner | 2026-11-24 |

**Authority** ([`AGENTS.md` §1.1](../../AGENTS.md#11-document-authority-and-conflicts)): the
[discovery register](../requirements/discovery-register.md) remains the authority on the **status and
category** of every discovery item. This guide and the other Phase 2A documents organize *how* those items
are investigated. They do not change any item's status, approve any ADR, or grant permissions.

---

## 1. What Phase 2 discovery is for

Phase 1 produced an engineering foundation built on almost no verified facts. Phase 2 replaces the most
important unknowns with **evidence** so that the owner can:

1. decide whether the business is **commercially feasible** (telco costs vs. customer prices vs. budget);
2. confirm a **regulatory path** with qualified Philippine counsel;
3. agree an **MVP scope** grounded in real launch customers;
4. make the **architecture decisions** that depend on external information.

Phase 2A (this package) is **preparation only**: questionnaires, worksheets, and templates the owner uses
in meetings. Nothing in this package is evidence by itself. **Phase 2 is not complete** until the exit
checklist is satisfied with real evidence and owner approvals.

## 2. The Phase 2A package

| Document | Use it to... |
| -------- | ------------ |
| This guide | Understand the workstreams, priorities, critical path, and completion criteria |
| [Owner discovery workbook](owner-discovery-workbook.md) | See every open question by workstream: why it matters, options, who answers, what evidence closes it, priority |
| [Stakeholder questionnaires](questionnaires/README.md) | Run meetings with each stakeholder group (one questionnaire per group; one per telco) |
| [Legal and regulatory research log](legal-regulatory-research.md) | See what public sources say (with citations) and what counsel must confirm |
| [Traffic evidence worksheet](traffic-evidence-worksheet.md) | Capture traffic data per customer/source to test the 5-billion figure |
| [Evidence register](evidence-register.md) | Record every piece of evidence received (references only — no confidential documents in Git) |
| [MVP scope worksheet](mvp-scope-worksheet.md) | Prioritize capabilities into Essential / Deferrable / Future |
| [Decision readiness](decision-readiness.md) | See which ADRs can be decided now and which wait for which information |
| [Phase 2 exit checklist](phase2-exit-checklist.md) | Track the roadmap's Phase 2 acceptance criteria against evidence and approvals |

Existing documents remain authoritative for their subjects and are **referenced, not copied**:
[discovery register](../requirements/discovery-register.md) (item status),
[telco discovery checklist](../integrations/telco-discovery-checklist.md) (telco technical questions),
[traffic capacity discovery](../requirements/traffic-capacity-discovery.md) (traffic questions and formulas),
[decision log](../engineering/decision-log.md) (ADRs), [risk register](../risks/risk-register.md),
[development roadmap](../roadmap/development-roadmap.md).

## 3. Workstreams

| Workstream | Goal | Main stakeholders | Key register items |
| ---------- | ---- | ----------------- | ------------------ |
| **A. Business model and commercial structure** | Decide the operating model, contracting entity, pricing/billing basis, and budget; check margin feasibility | Owner, business partner/financier, telco commercial teams, accountant | D-001, D-047, D-080–D-084, D-088, D-090, D-160–D-163 |
| **B. Traffic and customer demand** | Establish what the 5-billion figure means and what launch customers will really send | Owner, financier (source of figure), prospective customers | D-004, D-060–D-063, D-074, D-075 |
| **C. Telco connectivity and technical discovery** | Learn what each telco actually offers, technically and commercially | Smart, Globe, PLDT Enterprise, DITO (if in scope), intermediaries | D-002, D-003, D-040, D-041, D-069, D-070, D-100–D-119 |
| **D. Legal and regulatory** | Obtain counsel's view of the regulatory path and obligations | Philippine telecom counsel, data-privacy counsel, tax adviser | D-120–D-131, D-073 |
| **E. Product scope and MVP** | Agree what launches first and what waits | Owner, launch customers | D-007, D-020–D-029, D-064, D-065, D-072, D-085–D-087, D-089, D-091, D-094–D-099 |
| **F. Operations and security** | Decide who runs, supports, and secures the platform | Owner, internal operations staff, hosting providers | D-044, D-066–D-068, D-076–D-079, D-141 |

Each item is routed (respondent, evidence, priority) in the [owner discovery workbook](owner-discovery-workbook.md).

## 4. Priority levels

| Priority | Meaning | Rule of thumb |
| -------- | ------- | ------------- |
| **P0** | Blocks commercial feasibility, the regulatory path, or the architecture direction | Answer before committing significant money or engineering effort |
| **P1** | Needed for MVP scope and core design | Answer before the solution architecture and SRS are finalized |
| **P2** | Can be addressed after the initial architecture | Answer before the related feature is built |
| **Future** | Not required for the initial platform | Record; revisit after launch |

## 5. Evidence standards

An answer only changes a register item when it is backed by evidence recorded in the
[evidence register](evidence-register.md).

| Evidence type | Examples | Strength |
| ------------- | -------- | -------- |
| Signed or official document | Signed agreement, term sheet, official telco technical specification, regulator-issued certificate, counsel's written opinion | Strongest — can confirm an item |
| Written statement from an authorized person | Email from a telco account manager, customer's written volume estimate, minutes approved by both sides | Good — confirms with the stated scope and date |
| Official public source | Law or regulation text, regulator publication | Establishes what the text says; **interpretation still needs counsel** |
| Verbal statement / meeting notes by one side | "They said they can give us 100 TPS" | Weak — record as a lead, request written confirmation |
| Secondary source | News article, law-firm blog, AI summary | A lead only — never confirms an item |

Rules:

- **Confidential material stays out of Git.** Store agreements, rate cards, specifications, and
  credentials in the owner's secure document store; the evidence register records only a reference.
- Record **who** said it, **when**, and **in what capacity**. Telco sales statements are not technical commitments.
- Contradictory evidence is recorded, not resolved silently; the owner decides.
- AI-generated research is a starting point, **not** legal advice and not confirmation of anything.

## 6. Dependencies between workstreams

```
 A. Operating model & entity (D-160, D-161, D-001, D-047)
        │  determines who signs with telcos, what licence path counsel assesses
        ├──────────────────────────────┐
        ▼                              ▼
 D. Regulatory path (D-120, D-113)   C. Telco engagement (D-112 first: who are the counterparties?)
        │                              │  protocol, capacity, pricing, sender IDs, off-net, DITO
        │                              ▼
        │                     C→A. Telco pricing & billing basis (D-108, D-041, D-117)
        │                              │
 B. Launch customers & traffic (D-060, D-061, D-062) ──► A. Margin check: price vs. cost vs. budget
        │                                                     (D-080, D-084, D-090, D-162)
        ▼                                                     │
 E. MVP scope (D-089, D-020)  ◄───────────────────────────────┘
        │
        ▼
 F. Operations model (D-066, D-067, D-068)  →  Phase 2B: architecture, threat model, SRS
```

## 7. Critical path and who to contact first

The critical path is the sequence of P0 answers without which nothing else can be finalized.

| Step | What | Who | Unblocks |
| ---- | ---- | --- | -------- |
| 1 | **Internal alignment:** operating model, contracting entity, investor role, budget envelope, any external commitments already made (D-160, D-161, D-001, D-047, D-090, D-163) | Owner + business partner/financier | Steps 2–4 (what to ask counsel and telcos) |
| 2a | **Regulatory path:** registrations/authorizations needed for the chosen model, including the effect of RA 12234 and NTC VAS rules (D-120, D-113, D-121) | Telecom and privacy counsel | Commercial commitments; telco onboarding documents |
| 2b | **Telco counterparties and offers:** clarify Smart / PLDT Enterprise relationship first (D-112), then protocol, capacity, pricing, aggregator terms, off-net and DITO reach (D-100, D-101, D-107, D-108, D-113, D-118, D-069) | Smart, PLDT Enterprise, Globe (in parallel); DITO if coverage requires | Margin check; connectivity design; hosting |
| 2c | **Demand evidence:** what "5 billion" means; launch customers and their volumes (D-004, D-060, D-061, D-062) | Source of the figure; 3–5 prospective customers | Capacity negotiation; MVP scope |
| 3 | **Commercial feasibility:** billing basis, pricing structure, margin (D-080, D-084, D-041, D-162) | Owner (with accountant) using steps 2b + 2c | Go/no-go; MVP scope |
| 4 | **MVP scope** approved (D-089, D-020) | Owner | Phase 2B architecture and SRS |

Steps 2a, 2b, and 2c can run **in parallel** once step 1 is done. Counsel and telcos should be engaged
early because their lead times are outside the owner's control.

## 8. Completion criteria

Phase 2 discovery is complete only when the [Phase 2 exit checklist](phase2-exit-checklist.md) is
satisfied: every roadmap acceptance criterion is backed by recorded evidence or an explicit, recorded owner
decision. Preparing these documents does **not** complete Phase 2.

Phase 2A itself is complete when the owner has reviewed this package and starts using it.

## 9. How AI agents use this package

- Agents may help prepare meeting materials, transcribe notes into the evidence register, and propose
  register updates — but **must not** mark items confirmed, accept ADRs, or contact third parties.
- When the owner supplies evidence, agents update the register only as the evidence supports, citing the
  evidence ID, in a pull request for owner review.
