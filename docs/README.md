# Documentation Index

This folder holds all project knowledge that is not code. AI agents and humans rely on it to make
consistent decisions, so **keeping it accurate is part of every task**.

## Document map

| Area | Document | Purpose | Update when... |
| ---- | -------- | ------- | -------------- |
| Start | [`../AGENTS.md`](../AGENTS.md) | Binding rules for agent conduct, permissions, task boundaries, and workflow; defines the document hierarchy (§1.1) | A standard or rule changes (owner approval) |
| Start | [`glossary.md`](glossary.md) | Plain-language explanations of SMS, telco, and architecture terms | A new term is used in docs or code |
| Product | [`product/product-brief.md`](product/product-brief.md) | Business overview, customers, business model, initial scope | Business direction or scope changes |
| Requirements | [`requirements/discovery-register.md`](requirements/discovery-register.md) | Every open question, assumption, and its status | Any question is answered or raised |
| Requirements | [`requirements/traffic-capacity-discovery.md`](requirements/traffic-capacity-discovery.md) | Investigation of the reported 5-billion monthly figure | Traffic data is received |
| Architecture | [`architecture/architecture-principles.md`](architecture/architecture-principles.md) | Principles that govern the design (not the final design) | An ADR changes a principle |
| Architecture | [`architecture/messaging-reliability.md`](architecture/messaging-reliability.md) | **Authoritative:** event delivery (outbox, at-least-once), message lifecycle, ambiguous submissions, delivery receipts | Messaging semantics change |
| Architecture | [`architecture/api-idempotency.md`](architecture/api-idempotency.md) | **Authoritative:** proposed API idempotency contract | Contract decisions are made |
| Architecture | [`architecture/financial-integrity.md`](architecture/financial-integrity.md) | **Authoritative:** money rules, precision options, rounding, ledger operations | Billing decisions are made |
| Standards | [`standards/coding-standards.md`](standards/coding-standards.md) | Go, React, TypeScript, Tailwind conventions with examples | Conventions change |
| Standards | [`standards/testing-standards.md`](standards/testing-standards.md) | Testing levels, expectations, tools | Testing approach changes |
| Security | [`security/security-baseline.md`](security/security-baseline.md) | Security principles and open security work | Threats, controls, or obligations change |
| Engineering | [`engineering/ai-agent-workflow.md`](engineering/ai-agent-workflow.md) | How agents receive, plan, implement, test, and submit tasks | Process changes |
| Engineering | [`engineering/git-workflow.md`](engineering/git-workflow.md) | Branching, PRs, review, merge, branch protection | Process changes |
| Engineering | [`engineering/decision-log.md`](engineering/decision-log.md) | Architecture Decision Records (ADRs) | A decision is proposed or its status changes |
| Risks | [`risks/risk-register.md`](risks/risk-register.md) | Technical, operational, commercial, security, regulatory risks | Risks change or are mitigated |
| Roadmap | [`roadmap/development-roadmap.md`](roadmap/development-roadmap.md) | Phases, dependencies, milestone acceptance criteria | Scope or sequencing changes |
| Integrations | [`integrations/telco-discovery-checklist.md`](integrations/telco-discovery-checklist.md) | Technical questions for Smart, Globe, PLDT (and others) | Telco information is received |

## How AI agents should use these documents

1. Read `AGENTS.md` fully, then the documents listed for your task type in `AGENTS.md` §4.
2. Treat the **discovery register** as the authority on what is known. If your task depends on an item
   marked *Unknown*, *Assumption*, or *Telco-dependent*, do not guess — flag it.
3. Treat the **decision log** as the authority on what has been decided. *Proposed* is direction, not
   approval.
4. If you learn something new or discover a gap, update the relevant document in the same PR (or report
   it if the document is outside your task scope).
5. Each topic has **one authoritative document**; others summarize and link to it. Change the
   authoritative document first, then any summaries (notably `AGENTS.md`).
6. The document hierarchy is defined only in [`AGENTS.md` §1.1](../AGENTS.md#11-document-authority-and-conflicts): `AGENTS.md` for agent conduct, permissions, task
   boundaries, workflow, and non-negotiable rules; specialist documents for their technical subjects;
   Accepted ADRs for approved decisions; the discovery register for the status of project information.
   No document overrides an owner restriction or task boundary in `AGENTS.md`. If two documents genuinely
   conflict, **stop, report the conflict, and request owner resolution** — never choose silently.

## Conventions for all documents

- Each document starts with a status block: **Status**, **Owner**, **Last updated**.
- Document statuses: `Draft` → `In review` → `Approved`. All Phase 1 documents are **Draft** until the
  project owner reviews them.
- Distinguish clearly between **confirmed facts (with evidence)**, **proposals**, **assumptions**, and
  **unknowns**. Never present a proposal as a fact.
- Do not cite external sources that were not actually consulted. Links to official documentation are
  pointers for further reading, not claims that the content has been verified.
- Never put credentials, real customer data, real phone numbers, or confidential commercial terms in
  this repository. Commercial agreements and signed documents are stored outside Git, with only a
  reference (document name, date, location) recorded here.

## Planned future documents (not yet created)

- `requirements/srs.md` — Software Requirements Specification for the MVP (after discovery).
- `architecture/system-architecture.md` — solution architecture (architecture phase).
- `architecture/domain-model.md` — core entities and their relationships.
- `security/threat-model.md` — structured threat model.
- `integrations/<telco>/` — per-telco technical specifications **received from the telco**.
- `operations/` — runbooks, incident response, on-call (before production).
