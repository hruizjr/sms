# AGENTS.md — Instructions for AI Coding Agents

> **Every AI agent must read this entire file before doing any work in this repository.**
> It is authoritative for **agent conduct, permissions, task boundaries, repository workflow, and
> project-wide non-negotiable rules**. Technical subjects have their own authoritative documents.
> The full hierarchy, and what to do when documents conflict, is in
> [§1.1 Document authority and conflicts](#11-document-authority-and-conflicts).

| Field | Value |
| ----- | ----- |
| Project | Enterprise A2P SMS Platform (Philippines) |
| Current phase | **Phase 3 — Walking skeleton: complete (2026-11-26).** All eight groups are merged: 0 (decisions), 1 (Go API skeleton), 2 (PostgreSQL foundation), 3 (OpenAPI contract for the public messages API), 4 (Go API contract integration: Go types and Chi routes generated from the contract; no endpoint mounted), 5 (frontend foundation: client portal shell and shared UI package; no features, no API calls), 6 (local development stack and expanded CI, PR #10) and 7 (PostgreSQL Row-Level Security evaluation, PR #11; the owner accepted ADR-0010 for tenant-owned control-plane tables) (groups renumbered by the owner, [decision log §9](docs/engineering/decision-log.md#9-owner-decisions--go-contract-integration-phase-3-group-4-2026-11-25)). Closure: [decision log §13](docs/engineering/decision-log.md#13-phase-3-closure-2026-11-26). **Phase 4 — Group 1 (database role separation, tenant-context helper, tenant registry, RLS catalog guard) authorized by the owner on 2026-11-26** ([decision log §18](docs/engineering/decision-log.md#18-owner-decisions--database-role-separation-and-tenant-registry-phase-4-group-1-2026-11-26)) and merged. **Phase 4 — Group 2 (messaging backend and provider integration foundation, simulator only) authorized by the owner on 2026-11-27** ([decision log §19](docs/engineering/decision-log.md#19-owner-decisions--messaging-backend-and-provider-integration-foundation-phase-4-group-2-2026-11-27)) and merged. **Phase 4 — Group 3 (first working client portal: session sign-in, messages, single-SMS compose) authorized by the owner on 2026-11-27** ([decision log §20](docs/engineering/decision-log.md#20-owner-decisions--first-working-client-portal-phase-4-group-3-2026-11-27)); no other Phase 4 group is authorized. |
| Authorized scope | **Phase 3 walking skeleton** (owner approval 2026-11-25), complete; **Phase 4 Group 1** (owner approval 2026-11-26; [decision log §18](docs/engineering/decision-log.md#18-owner-decisions--database-role-separation-and-tenant-registry-phase-4-group-1-2026-11-26)), merged; **Phase 4 Group 2** (owner approval 2026-11-27; scope and exclusions in [decision log §19](docs/engineering/decision-log.md#19-owner-decisions--messaging-backend-and-provider-integration-foundation-phase-4-group-2-2026-11-27)), merged; and **Phase 4 Group 3** (owner approval 2026-11-27; [decision log §20](docs/engineering/decision-log.md#20-owner-decisions--first-working-client-portal-phase-4-group-3-2026-11-27)). Every other Phase 4 group needs explicit owner approval. Each implementation group starts only after the previous group is reviewed and merged. All other ADRs need separate owner approval. |
| Decision authority | The project owner has decision authority over day-to-day project matters. Decisions affecting pricing, product scope, or launch require business-partner approval in addition to the project owner's approval. Both approvals, including their dates, must be recorded before a decision is labelled approved. Supporting approval records are kept outside Git. Agents propose; they never record a partner approval that the owner has not confirmed in writing ([decision log §17](docs/engineering/decision-log.md#17-owner-decisions--business-decisions-review-phase-2-ex-9-2026-11-26)). |
| Last updated | 2026-11-26 (Phase 3 closure; §12 MFA wording for owner decision D-021; §3.3 ADR-0012 status; EX-9 governance, operating model and billing wording: header, §2 rule 21, §3.1, §11.9, §17; Phase 4 Group 1: header, §1, §4, §7.1, §10.3, §10.7; 2026-11-27, Phase 4 Group 2: header, §1, §4; 2026-11-27, Phase 4 Group 3: header, §1, §4, §8.2) |

---

## Contents

1. [Current status and what you may do](#1-current-status-and-what-you-may-do)
   - [1.1 Document authority and conflicts](#11-document-authority-and-conflicts)
2. [AI agent operating rules](#2-ai-agent-operating-rules)
3. [Project context](#3-project-context)
4. [Repository map and required reading](#4-repository-map-and-required-reading)
5. [Engineering principles](#5-engineering-principles)
6. [Architectural constraints](#6-architectural-constraints)
7. [Go standards](#7-go-standards)
8. [Frontend standards](#8-frontend-standards)
9. [API standards](#9-api-standards)
10. [Database standards](#10-database-standards)
11. [Messaging standards](#11-messaging-standards)
12. [Security standards](#12-security-standards)
13. [Testing requirements](#13-testing-requirements)
14. [Git and pull request workflow](#14-git-and-pull-request-workflow)
15. [Definition of Done](#15-definition-of-done)
16. [Completion report format](#16-completion-report-format)
17. [Decisions: how to read and propose them](#17-decisions-how-to-read-and-propose-them)

The words **MUST**, **MUST NOT**, **SHOULD**, and **MAY** are used deliberately. MUST rules are not
optional; if you believe one must be broken, stop and ask the project owner.

---

## 1. Current status and what you may do

The repository contains documentation, repository hygiene files, the Go API skeleton in `backend/` (health
and readiness endpoints, metrics, CI), the PostgreSQL foundation (connection pool, goose migrations without
business tables, local Docker Compose database, integration tests), the draft OpenAPI contract for the
public messages API in `contracts/openapi/`, Go types and Chi routes generated from that contract in
`backend/internal/publicapi/`, the frontend foundation in
`frontend/` (client portal shell with one placeholder page, shared UI package, no features, no API calls),
a local Docker Compose stack in `infra/` (PostgreSQL, a one-off migration service, the API) with expanded
CI (stack check, secret scan, dependency review), and the Row-Level Security evaluation tests in
`backend/internal/database/rlseval/` (Group 7; synthetic test-only tables; no product code). The Phase 3 walking
skeleton (technical foundation only) was added one reviewed group at a time and is complete. **Phase 4 Group 1**
adds the database foundation: separate database roles (`sms_migrator` for migrations, `sms_app` for the API, with
checks that refuse unsafe roles), the transaction-local tenant-context helper, the platform-owned tenant registry
(`tenants` table, sqlc queries, `internal/tenancy` repository without HTTP endpoints) and the RLS catalog guard in
CI. **Phase 4 Group 2** adds the working messaging backend: the public API v1 (`/v1/messages`) with Bearer API
keys, the messaging engine and `dispatcher` (transactional outbox, submission attempts, receipts), the provisional
provider interface with the **simulator** as the only provider (no real telco adapter; no SMS is sent), and the
local administration tool `smsctl`. **Phase 4 Group 3** adds the first working client portal: session-authenticated
portal API (`/portal/v1`, contract `contracts/openapi/portal-api-v1.yaml`) with Argon2id passwords, server-side
sessions, CSRF protection and account lockout; portal users of the three client roles provisioned with `smsctl`; and
the client portal pages (sign in, messages, message details, compose one SMS) using the same messaging engine.
**No other business functionality exists or is authorized.**

**Verified facts:** almost none. The business *intends* to become an A2P SMS aggregator. No telco
agreement, technical integration, accreditation, regulatory authorization, or traffic volume has been
verified. Treat everything else as a proposal until the
[discovery register](docs/requirements/discovery-register.md) says otherwise.

Until a task explicitly authorizes it, you MUST NOT:

- implement application features, REST endpoints, business logic, or UI pages;
- create database schemas or migrations;
- write SMS-sending code or SMPP/telco connector code;
- add dependencies, `go.mod`, or `package.json`;
- provision servers, deploy anything, or configure real telco credentials;
- send any message to any real phone number.

Phase 3 group tasks authorize only what [decision log §7](docs/engineering/decision-log.md#7-owner-decisions--phase-3-walking-skeleton-2026-11-25)
lists: the approved dependencies, `go.mod` / `package.json`, migrations **without business tables**, CI, and
local development tooling. Phase 4 Group 1 additionally authorizes only its own scope ([decision log §18](docs/engineering/decision-log.md#18-owner-decisions--database-role-separation-and-tenant-registry-phase-4-group-1-2026-11-26)):
the database roles, the tenant-context helper, the `tenants` migration and its sqlc queries and repository, the
pinned sqlc tool, and the RLS catalog guard. Phase 4 Group 2 additionally authorizes only its own scope
([decision log §19](docs/engineering/decision-log.md#19-owner-decisions--messaging-backend-and-provider-integration-foundation-phase-4-group-2-2026-11-27)):
the `/v1/messages` endpoints, API keys, the messaging tables, the dispatcher, the provider interface, the simulator and
`smsctl`. Phase 4 Group 3 additionally authorizes only its own scope
([decision log §20](docs/engineering/decision-log.md#20-owner-decisions--first-working-client-portal-phase-4-group-3-2026-11-27)):
client portal sign-in and sessions, portal users and their read/send permissions, the `/portal/v1` endpoints, the
login directory (the one approved RLS guard exception) and the client portal pages. Admin login, MFA, user and API key
management, RBAC beyond the client roles' read/send permissions, billing, pricing, SMPP, real telco adapters or credentials,
third-party routing, fallback routing, Kafka, RabbitMQ, Redis, and ClickHouse remain prohibited. The metrics endpoint must not
be publicly exposed, and GitHub Actions are pinned to commit SHAs.

### 1.1 Document authority and conflicts

This section is the **only** definition of the document hierarchy; other documents refer to it.

| Source | Authoritative for |
| ------ | ----------------- |
| **`AGENTS.md`** (this file) | Agent conduct, permissions, task boundaries, repository workflow, and project-wide non-negotiable rules (MUST / MUST NOT rules in this file). [Git workflow](docs/engineering/git-workflow.md) and [AI agent workflow](docs/engineering/ai-agent-workflow.md) give the detailed procedures; they add detail but cannot relax this file. |
| **Specialist documents** | Their own technical subjects (listed below the table). |
| **Accepted ADRs** in the [decision log](docs/engineering/decision-log.md) | Approved architectural decisions. **Proposed** ADRs are direction, not binding (§2 rule 3). |
| **[Discovery register](docs/requirements/discovery-register.md)** | Status of known, assumed, and unresolved project information. |

Specialist documents and their subjects:

- [messaging reliability](docs/architecture/messaging-reliability.md) — event delivery, message lifecycle,
  provider submission outcomes, delivery receipts;
- [API idempotency](docs/architecture/api-idempotency.md) — idempotency contract;
- [financial integrity](docs/architecture/financial-integrity.md) — monetary representation, rounding,
  ledger rules;
- [security baseline](docs/security/security-baseline.md) — security principles and controls;
- [testing standards](docs/standards/testing-standards.md) — test levels, scenarios, expectations;
- [coding standards](docs/standards/coding-standards.md) — language conventions and examples.

Rules:

1. **Summaries are pointers.** Where this file summarizes a technical subject (e.g. §11), the specialist
   document holds the full rule. A difference between a summary and its specialist document is a
   conflict (rule 4), not a choice.
2. **Proposals stay proposals.** When a specialist document describes something tied to a Proposed ADR or
   an open discovery item, it is the authoritative description of that **proposal or open question**,
   not an approved decision.
3. **No document overrides owner restrictions.** No specialist document, ADR, or register entry may grant
   a permission, widen a task's scope, or override a restriction or task boundary in this file or in the
   owner's task definition.
4. **Genuine conflicts stop work.** If two documents genuinely conflict (they require incompatible
   actions or state incompatible facts), agents MUST stop the affected work, report the conflict (both
   documents, sections, and wording), and request owner resolution. Agents MUST NOT silently choose
   one. Additional detail or a stricter requirement that is consistent with the other document is not a
   conflict.
5. **Fixing a conflict.** Once the owner decides, update the authoritative document first, then any
   summaries that refer to it, in the same pull request.

---

## 2. AI agent operating rules

These rules apply to every agent, on every task, with no exceptions.

1. **Read this `AGENTS.md` before starting work.**
2. **Read the relevant project documentation** before implementing a task (see §4 for what to read).
3. **Follow approved architectural decisions** in the [decision log](docs/engineering/decision-log.md).
   Decisions marked *Proposed* are the current direction but not binding — if your task depends on one,
   say so in your report.
4. **Never invent telco API specifications, SMPP configurations, or commercial terms.** Hostnames, ports,
   system IDs, system types, TON/NPI values, throughput limits, error-code meanings, delivery-receipt
   formats, encoding behaviour, rates, and billing terms MUST come from official telco documentation or
   signed agreements stored or referenced in `docs/integrations/`. If it is not documented, write
   `TODO(telco-confirm): <question>` and record the question in the
   [telco discovery checklist](docs/integrations/telco-discovery-checklist.md).
5. **Never introduce a dependency without explaining why.** In the PR, state: what it does, why the
   standard library or an existing dependency is insufficient, its licence, its maintenance status, and
   its security record. New dependencies require owner approval.
6. **Never modify unrelated files.** No drive-by refactors, reformatting, or "improvements" outside the
   task scope. If you spot a problem elsewhere, report it instead.
7. **Never silently change database schemas or API contracts.** Any change to migrations,
   `contracts/`, event schemas, or public API behaviour MUST be listed under "Schema / contract changes"
   in the PR description with its compatibility impact.
8. **Write appropriate automated tests** for every behaviour change (see §13).
9. **Run the relevant tests before declaring completion**, and report the exact commands and results.
   If you could not run them, say so plainly and explain why.
10. **Report every changed file** and explain the change (see §16).
11. **Identify assumptions and unresolved issues** explicitly. Never hide uncertainty.
12. **Never commit directly to the `main` branch.** Work on a feature branch and open a pull request.
13. **Never expose credentials, API keys, tokens, or secrets** — not in code, tests, fixtures, logs,
    commit messages, PR descriptions, screenshots, or chat output.
14. **Never claim an integration works without an actual test.** State precisely what was tested:
    "verified against the telco test environment on <date>", "verified against the local simulator",
    or "mock test only". Untested integrations MUST be described as untested.

Additional rules:

15. **Work only on a clearly defined task** (see [AI agent workflow](docs/engineering/ai-agent-workflow.md)).
    If the task is ambiguous, ask before writing code.
16. **Avoid overlapping modifications to shared files.** Shared "hot" files — `AGENTS.md`, anything in
    `contracts/`, migrations, `go.mod`/`go.sum`, `package.json`/lockfiles, CI configuration, and
    `infra/` — may only be changed when the task explicitly includes them.
17. **Do not mark business, telco, or regulatory items as confirmed** without evidence (a signed
    document, official specification, or written confirmation referenced in the docs).
18. **Prefer the simplest design that meets the documented requirement.** Do not add infrastructure,
    services, abstractions, or configurability "for the future" without an approved requirement.
19. **Keep documentation true.** If your change makes a document inaccurate, update the document in the
    same pull request.
20. **Never use real personal data.** Test data must be synthetic. Do not paste customer data,
    phone numbers, or message content into prompts, logs, or fixtures.
21. **Stop and ask** when a task would require a decision reserved for the project owner or requiring
    business-partner approval (§17), an
    action outside the repository (deployment, account creation, contacting third parties), or breaking
    a MUST rule. This includes any implementation that depends on an **open** item in the
    [discovery register](docs/requirements/discovery-register.md) or a **Proposed** ADR whose choice
    affects the code — ask for owner approval instead of picking an answer.

### 2.1 Critical correctness rules (messaging and money)

These summarize rules whose full definitions live in the linked authoritative documents. They apply to
code, comments, API documentation, UI text, and reports.

22. **Never claim exactly-once SMS delivery** unless a provider-specific, documented guarantee exists and
    an ADR records it. Internal events are at-least-once; consumers are idempotent
    ([messaging reliability §1–§2](docs/architecture/messaging-reliability.md)).
23. **Treat a provider submission timeout as an ambiguous outcome**, not as a failure. **Never blindly
    retry** an ambiguous submission — it can send a duplicate SMS to a real person
    ([messaging reliability §4](docs/architecture/messaging-reliability.md#4-provider-submission-and-ambiguous-outcomes)).
24. **Treat delivery receipts as potentially late, missing, duplicated, conflicting, or out of order.**
    A missing receipt is not proof of non-delivery
    ([messaging reliability §5](docs/architecture/messaging-reliability.md#5-delivery-receipt-dlr-handling)).
25. **Never use floating-point arithmetic for money**, and never choose a monetary scale or rounding rule
    yourself — they are open decisions
    ([financial integrity](docs/architecture/financial-integrity.md), ADR-0019).
26. **Preserve financial and status audit history.** Ledger entries, raw receipts, and status history are
    append-only; corrections are new entries.
27. **An idempotency key prevents duplicate operations inside our platform only** — it does not guarantee
    a single telco submission ([API idempotency](docs/architecture/api-idempotency.md)).

---

## 3. Project context

### 3.1 The business

We are building the software for a **Philippine A2P SMS aggregation business**. "A2P"
(Application-to-Person) means messages sent by a business system to a person's phone — one-time
passwords (OTPs), transaction alerts, appointment reminders, government advisories, and marketing.
An **aggregator** connects many business customers to the mobile networks through one platform.

**Operating model (D-160, owner-approved, pending partner ratification):** **direct SMS aggregation only**,
through the business's own commercial agreements and direct technical integrations with telcos. No third-party
SMS aggregator or intermediary may route messages (including the first live connection and any fallback), and
there is no reseller or white-label model. **Coverage objective:** direct integration with the Philippine mobile
network operators **Smart Communications**, **Globe Telecom** and **DITO Telecommunity** — an objective, not a
promise of any network at launch. **PLDT Enterprise is not a separate mobile network**; it may be a commercial
or enterprise contracting counterpart, and its relationship to Smart (same corporate group) must be clarified
(D-112). Initial discussions have reportedly occurred with Smart, Globe, PLDT, and the
**NTC** (National Telecommunications Commission, the Philippine regulator). **None of these are
confirmed.**

The owner has reported a projected **~5 billion monthly traffic** figure. This is **unverified** and its
unit is unknown (messages? segments? recipients? prospective totals?). **Do not size or design anything
on this number.** See [traffic capacity discovery](docs/requirements/traffic-capacity-discovery.md).

### 3.2 What the platform does (intended)

- **Client Portal:** onboarding, users, contacts and imports, campaigns and scheduling, message
  composition, delivery reports, analytics, credit balance, API keys.
- **Super Admin Portal:** client and user management, sender ID management, campaign approval and
  monitoring, pricing, credit allocation, billing history, provider configuration, monitoring, audit logs.
- **Messaging infrastructure:** provider adapters (Smart, Globe, ...), ingestion, queueing, routing,
  per-provider rate limiting, delivery receipts, retries, duplicate protection, connection monitoring.
- **Billing:** client-specific pricing, credits, reservations, deductions, ledger, reconciliation.
- **Developer platform:** public REST API, authentication, documentation, webhooks, rate limits.
- **Security and compliance:** tenant isolation, RBAC, consent, opt-out/suppression, audit logs,
  data protection, campaign restrictions, retention.

Details and gaps: [product brief](docs/product/product-brief.md) and
[discovery register](docs/requirements/discovery-register.md).

### 3.3 Intended architecture (direction, not final)

- **Monorepo** containing frontend, backend, contracts, infrastructure config, and docs.
- **Modular monolith** in Go: one module, several executables, strict module boundaries (ADR-0003, **Accepted**).
- **Control plane / data plane separation** — logical from the start, physical (separate processes)
  where justified (ADR-0009, Proposed).
- **Provider abstraction** — every telco is an adapter behind one interface (ADR-0012, **Accepted — limited scope**
  2026-11-26: provisional interface, capability declarations, adapter abstraction and simulator; no SMPP library
  or real adapter). Development and testing are **simulator-only** until a telco specification is received
  (owner decision, [decision log §15](docs/engineering/decision-log.md#15-owner-decisions--simulator-only-development-phase-2-ex-3-2026-11-26)).
- **PostgreSQL** as the system of record (ADR-0005, **Accepted**; PostgreSQL 18).
- Kafka/Redpanda, Redis, ClickHouse are **candidates, deferred** until verified requirements justify them
  (ADR-0006/0007/0008, Proposed).

### 3.4 Technology preferences and their status

| Area | Preference | Status |
| ---- | ---------- | ------ |
| Frontend | React, TypeScript (strict), Vite, Tailwind CSS v4; pnpm 10 (exact version pinned); Node.js 26 LTS | **Accepted** (ADR-0013); one app `client-portal` + `packages/ui`; router, data-fetching, and form libraries deferred to Phase 4 |
| Styling | Tailwind CSS only; no other styling or CSS frameworks | Owner requirement |
| Backend language | Go **1.27.4** (pinned locally and in CI) | **Owner decision** (decision log §7) |
| HTTP router | Chi (`go-chi/chi/v5`) | **Accepted** (ADR-0004) |
| API style | REST + JSON, OpenAPI contracts | **Accepted** (ADR-0015) |
| Primary database | PostgreSQL 18 (latest 18.x patch) | **Accepted** (ADR-0005); review PostgreSQL 19 before production |
| Database access / migrations | `pgx/v5` + `sqlc`; goose | **Accepted** (ADR-0014) |
| Logging | `log/slog` | **Accepted** (ADR-0016) |
| Message broker | Kafka or Redpanda | **Deferred** (ADR-0006) |
| Cache / coordination | Redis | **Deferred** (ADR-0007) |
| Analytics store | ClickHouse | **Deferred** (ADR-0008) |
| Containers | Docker | Preferred |
| Metrics / dashboards | Prometheus (`client_golang`) / Grafana | Prometheus client approved for Phase 3 (endpoint not publicly exposed); Grafana preferred, not yet introduced |

---

## 4. Repository map and required reading

```
AGENTS.md          This file
backend/           Go module (cmd/ = executables, internal/ = modules)   — API, messaging engine, dispatcher, simulator
frontend/          React applications (pnpm workspace)                   — client portal (Phase 4 Group 3) + packages/ui
contracts/         OpenAPI specs, event and webhook schemas              — draft public API v1, client portal API v1
backend/internal/publicapi/  Go code GENERATED from contracts/openapi/  — never edit by hand (Group 4)
infra/             Docker, local dev stack, observability config         — backend image + Compose stack (Group 6)
scripts/           Helper scripts                                        — local stack check (Group 6), RLS evaluation (Group 7)
tests/             E2E, load, provider-contract suites                   — empty
docs/              All project documentation (index: docs/README.md)
.github/           PR template, task template
```

**Required reading by task type** (in addition to this file):

| Task touches... | Read first |
| --------------- | ---------- |
| Anything | [docs/engineering/ai-agent-workflow.md](docs/engineering/ai-agent-workflow.md), [docs/engineering/git-workflow.md](docs/engineering/git-workflow.md) |
| Go code | [docs/standards/coding-standards.md](docs/standards/coding-standards.md), [docs/architecture/architecture-principles.md](docs/architecture/architecture-principles.md) |
| Frontend | [docs/standards/coding-standards.md](docs/standards/coding-standards.md) (frontend sections) |
| Database | §10 below, [decision log](docs/engineering/decision-log.md) (ADR-0005, 0010, 0011, 0014) |
| Messaging / providers / events | §11 below, [messaging reliability](docs/architecture/messaging-reliability.md), [telco discovery checklist](docs/integrations/telco-discovery-checklist.md), [glossary](docs/glossary.md) |
| Public API / message submission | §9 below, [API idempotency](docs/architecture/api-idempotency.md) |
| Billing / credits / pricing | §10, §11.9, [financial integrity](docs/architecture/financial-integrity.md), ADR-0011, ADR-0019 |
| Auth, tenants, API keys, PII | [docs/security/security-baseline.md](docs/security/security-baseline.md) |
| Tests | [docs/standards/testing-standards.md](docs/standards/testing-standards.md) |
| Requirements questions | [docs/requirements/discovery-register.md](docs/requirements/discovery-register.md) |

---

## 5. Engineering principles

1. **Maintainable and readable code.** Code is read far more than written, mostly by other agents.
   Prefer obvious over clever. Name things for what they mean in the business domain.
2. **Separation of concerns.** Transport (HTTP, SMPP), application logic, and persistence are separate
   layers. Business rules never live in HTTP handlers, SQL strings, or React components.
3. **Modular architecture.** Organize by business capability. Modules expose a small Go API and own
   their data. Cross-module calls go through that API, never through another module's tables.
4. **Security by design.** Tenant isolation, least privilege, input validation, and auditability are
   designed in from the first line, not added later.
5. **Testability.** Inject dependencies (database, clock, ID generator, provider adapters) so logic can be
   tested without real infrastructure. No hidden globals.
6. **Scalability without premature complexity.** Design so components *can* be separated and scaled
   (stateless services, idempotent processing, explicit boundaries), but do not introduce distributed
   infrastructure until verified requirements need it.
7. **Explicit interface contracts.** APIs, events, and provider interfaces are written down, versioned,
   and tested. Implicit coupling (shared tables, undocumented JSON fields) is not allowed.
8. **Database integrity.** Use the database's guarantees — constraints, foreign keys, transactions,
   unique indexes — rather than hoping application code is always correct.
9. **Correctness over throughput for money.** Financial operations must be exact, auditable, and
   reconcilable even if that costs performance.
10. **Honesty about uncertainty.** Unknowns are documented, not guessed.

---

## 6. Architectural constraints

These constraints must be preserved by all future work. Changing one requires an accepted ADR.

1. **Provider abstraction.** Smart, Globe, and any future telco are independent **adapters** that
   implement a common provider interface defined by the messaging module. Business logic (campaigns,
   billing, routing policy) MUST NOT import a specific adapter or contain telco-specific branches.
   A **simulator adapter** MUST exist so the whole system can run and be tested without any telco.
2. **Control plane vs. data plane.** Control-plane concerns (tenants, users, auth, configuration,
   pricing, billing, administration) and data-plane concerns (ingestion, dispatch, routing, delivery
   receipts) are separate modules with explicit interfaces, so they can run as separate processes.
   An outage or slowdown of the admin portal must not stop message dispatch, and a traffic spike must
   not take down the portals.
3. **Multi-tenancy.** Every client business is a **tenant**. Tenant data is strictly isolated. The
   tenant is always derived from the authenticated identity, never from user-supplied input.
4. **Financial integrity.** Credits and charges are recorded in an append-only, concurrency-safe ledger.
   Balances can never be silently changed. Every charge is traceable to a message or an authorized
   administrative action.
5. **At-least-once processing with idempotency.** Internal processing assumes events may be delivered
   more than once. Every consumer is idempotent. Reconciliation jobs detect drift and report it; repairs
   are explicit, audited actions. **Never promise exactly-once SMS delivery** — telco submission is
   inherently ambiguous on timeouts (see §11.5).
6. **Scalability based on verified requirements.** No single process is assumed to handle all traffic,
   but capacity targets come only from verified traffic data and contracted telco throughput.

---

## 7. Go standards

Details and examples: [coding standards](docs/standards/coding-standards.md#go).

### 7.1 Package organization
- One Go module in `backend/`. Executables in `backend/cmd/<name>/`; application code in
  `backend/internal/<module>/`. Shared technical plumbing is one focused package per concern, directly
  under `backend/internal/` (`config`, `database`, `migration`, `httpserver`, `middleware`,
  `observability`); there is no `internal/platform/` umbrella (owner decision, Phase 3 Group 2).
  Technical packages contain no business rules.
- `main.go` is thin: load config → build dependencies → run → shut down gracefully.
- Package names are short, lowercase, singular, without underscores (`billing`, not `billing_service`).
- **Generated code MUST NOT be edited by hand** (`backend/internal/publicapi/*.gen.go`, and sqlc output in
  `internal/database/sqlcgen`). Change the source (specification, SQL, or generator
  configuration) and regenerate; CI fails on stale or hand-edited generated code
  ([decision log §9](docs/engineering/decision-log.md#9-owner-decisions--go-contract-integration-phase-3-group-4-2026-11-25)).
- MUST NOT create `utils`, `common`, `helpers`, `misc`, or `pkg/` (unless code is genuinely meant for
  external import, which requires an ADR).
- Define interfaces **where they are consumed**, keep them small, and return concrete types.
- No import cycles; dependencies point inward (handler → service → repository).

### 7.2 Error handling
- Every error is handled. `_ = err` requires a comment explaining why.
- Wrap with context: `fmt.Errorf("reserve credits for message %s: %w", id, err)`. Inspect with
  `errors.Is` / `errors.As`.
- Expected conditions use exported sentinel or typed errors (`ErrInsufficientCredit`, `ErrNotFound`).
- Map domain errors to HTTP status codes **only at the transport boundary**. Never return internal error
  text, SQL errors, or stack traces to clients.
- Log an error **once**, at the boundary that handles it — do not log and return.
- `panic` only for programmer errors at startup. Recover panics at HTTP middleware and at the top of
  every long-running goroutine, log them, and convert them to errors.

### 7.3 Context propagation
- `ctx context.Context` is the first parameter of any function that does I/O or may block.
- Never store a context in a struct; never pass `nil`; never use `context.Background()` inside request
  handling (only in `main` and tests).
- Every outbound call (database, HTTP, telco, broker) has a deadline.
- Context values are only for request-scoped metadata (request ID, authenticated principal, tenant),
  using unexported typed keys and accessor functions.

### 7.4 Concurrency and goroutine lifecycle
- **Every goroutine has an owner** responsible for stopping it and waiting for it to exit.
- No fire-and-forget `go func()` in request paths. No unbounded goroutine creation per message/request.
- Concurrency is **bounded**: worker pools or semaphores with configured limits.
- Long-running loops `select` on `ctx.Done()`.
- The sender closes a channel; receivers never close.
- Shared state is protected by a mutex or confined to one goroutine. Do not use `time.Sleep` to
  synchronize.
- Graceful shutdown on SIGTERM/SIGINT: stop accepting work → finish or hand back in-flight work within a
  deadline → close resources in reverse order of creation.
- All Go tests run with `-race` in CI.

### 7.5 Resource cleanup
- `defer x.Close()` immediately after successful acquisition. Check `Close` errors on writes.
- Always close HTTP response bodies, `rows.Close()` and check `rows.Err()`, stop tickers/timers.
- Connection pools (DB, HTTP clients) are created once and shared; never per request.

### 7.6 Logging
- Use the standard library `log/slog`. JSON output outside local development.
- Structured fields, not string concatenation. Standard keys: `request_id`, `tenant_id`, `user_id`,
  `message_id`, `provider`, `campaign_id`, `error`.
- **MUST NOT log** secrets, tokens, passwords, API keys, full message bodies, OTP values, or full
  phone numbers. Mask phone numbers (e.g. `+63917****567`).
- Levels: `DEBUG` (development detail), `INFO` (state changes), `WARN` (recoverable problems),
  `ERROR` (failed operations needing attention).

### 7.7 Configuration
- Configuration comes from environment variables, parsed **once** at startup into a typed struct,
  validated, and failing fast with a clear message if invalid.
- No global mutable configuration. Pass config to constructors.
- Safe defaults for development; **no defaults for secrets**.
- Every variable is documented in an `.env.example` (no real values).

### 7.8 Database access
- PostgreSQL driver and query approach per ADR-0014 (accepted: `pgx/v5` + `sqlc`; migrations with goose).
- SQL lives only in repository code. **Parameterized queries only** — never build SQL with string
  formatting from input.
- Tenant-scoped repository methods take `tenantID` as an explicit parameter.
- Transactions via a helper that accepts `func(tx) error`, commits on success, rolls back on error.
- **No network calls to telcos or external HTTP services inside a database transaction.**

### 7.9 Testing (Go)
- Standard `testing` package; table-driven tests; `t.Parallel()` where safe; `t.Cleanup` for teardown.
- Prefer hand-written fakes over generated mocks. Inject clocks and ID generators for determinism.
- Integration tests use a real PostgreSQL (containerized), not mocks of SQL.
- Tooling (approved for Phase 3): `gofmt`, `go vet`, `golangci-lint`, `govulncheck`.

### 7.10 Other rules
- Money, rates, and credits are **never** `float32`/`float64`. Use the exact representation decided in
  ADR-0019 (currently **open** — see [financial integrity](docs/architecture/financial-integrity.md)).
- Times are `time.Time` in UTC internally; convert to `Asia/Manila` only for display.
- Use generics only where they clearly remove duplication.

---

## 8. Frontend standards

Details: [coding standards](docs/standards/coding-standards.md#frontend).

### 8.1 Component organization
- Feature-first structure: `src/features/<feature>/` contains that feature's components, hooks, API
  calls, and tests. Design tokens and primitives shared across applications live in the shared UI package
  `frontend/packages/ui` (`@sms/ui`, ADR-0013); components used by one application only stay in that
  application (`src/components/`, layouts in `src/components/layout/`) (owner decision F8, decision log §10).
- A feature MUST NOT import another feature's internal files. Share via `components/` or `lib/`.
- Function components, named exports, one component per file, explicit prop types.
- UI primitives contain no business logic; business logic lives in hooks or feature modules.

### 8.2 TypeScript
- `"strict": true` plus `noUncheckedIndexedAccess`, `noImplicitOverride`, `noFallthroughCasesInSwitch`.
- No `any` (use `unknown` and narrow). No `@ts-ignore`; `@ts-expect-error` only with a comment.
- API types are **generated from `contracts/`** (ADR-0015), not hand-written duplicates. The client portal's types
  are generated from `contracts/openapi/portal-api-v1.yaml` with `openapi-typescript`
  (`pnpm --filter @sms/client-portal generate:api`; CI fails on stale output), and all calls go through the one
  typed client in `src/lib/api/` (owner decision G3-6, decision log §20, superseding F3). The browser never uses
  the public API or API keys.

### 8.3 Tailwind CSS and design tokens
- Tailwind CSS is the **only** styling system. No CSS-in-JS, Bootstrap, Material UI, or similar.
- All colours, spacing scale extensions, radii, shadows, and typography are **design tokens** defined
  once as CSS variables (Tailwind's theme configuration) in `frontend/packages/ui/src/theme.css`;
  Tailwind's default colour palette is switched off.
- Components use **semantic** tokens (`bg-surface`, `text-muted`, `border-danger`), never raw hex values
  or arbitrary values like `bg-[#1a2b3c]`.
- Variants (size, intent) are defined in one place per component, not scattered conditionals.
- Inline `style` only for genuinely dynamic values (e.g. a computed progress width).
- Dark mode support is an open question; tokens must make it possible without rewriting components.

### 8.4 Reusable components
- Before creating a component, search `components/ui/` for an existing one. Extend it rather than copy it.
- New shared components need: typed props, all interactive states (hover, focus, disabled, loading,
  error), keyboard support, and tests.
- ADR-0013 (accepted): complex interactive behaviours (dialogs, menus, dropdowns, popovers, comboboxes)
  may later use a maintained, accessible **headless** primitives library, styled with Tailwind. No pre-styled
  UI framework. The library decision (D-140) is **deferred by the owner**; do not add one.

### 8.5 API integration
- All HTTP calls go through the single typed API client in `src/lib/api/`. Components never call `fetch`
  directly.
- A data-fetching library is **deferred to Phase 4** (ADR-0013); until then, use the typed API client and do
  not copy server data into global client state.
- Handle `401` (session expired) and `403` (forbidden) centrally. Display RFC 9457 problem details
  returned by the server.

### 8.6 Form validation
- Validate on the client for usability **and** on the server for correctness — the server is the
  authority.
- Map server field errors to the matching fields. Prevent double submission.
- Actions that send messages or spend credits MUST show a confirmation step with recipient count,
  segment count, and estimated cost.

### 8.7 Error, empty, and loading states
- Every data view handles **loading, empty, error, and success** states explicitly.
- Errors are user-friendly with a retry action; never show stack traces or raw server errors.
- Use skeletons for initial loads; avoid layout shift.

### 8.8 Responsive layouts
- Mobile-first. Verify at 360 px, 768 px, and 1280 px widths.
- Wide data tables must remain usable on small screens (responsive columns, card layout, or contained
  horizontal scroll — not page-level horizontal scroll).

### 8.9 Accessibility
- Target **WCAG 2.2 Level AA**.
- Semantic HTML first; ARIA only when semantics are insufficient.
- Every input has a visible label; errors are associated with their inputs.
- Full keyboard operability; visible focus indicators; dialogs trap and restore focus.
- Colour contrast meets AA; never convey meaning by colour alone (e.g. delivery status uses icon + text).

### 8.10 Frontend security
- **Never put secrets in frontend code.** Any variable exposed to the Vite client bundle (e.g.
  `VITE_*`) is public.
- Never render user-supplied content as HTML (`dangerouslySetInnerHTML` is prohibited for user data).
- Never perform monetary arithmetic with JavaScript numbers; display server-provided amounts only.
- Store session state in secure, HttpOnly cookies managed by the server (ADR pending), not in
  `localStorage`.

---

## 9. API standards

(ADR-0015 and ADR-0018 (idempotency) accepted 2026-11-25. ADR-0019 (monetary representation) remains
Proposed; its item below is marked as proposed.)

- REST over HTTPS, JSON bodies, OpenAPI specification in `contracts/openapi/` as the source of truth.
  Go types and routes are generated from it (`backend/internal/publicapi`, oapi-codegen); never hand-write
  duplicates of contract types. Generated parameter binding is **not** request validation or
  authentication: both MUST be added before any `/v1` endpoint serves traffic (decision log §9, D6).
- Versioned paths: `/v1/...`. No breaking changes within a published version.
- JSON field naming: `snake_case`. Timestamps: RFC 3339 in UTC. Phone numbers: E.164
  (`+639171234567`).
- Errors: RFC 9457 Problem Details (`application/problem+json`) with a stable machine-readable `type`.
- Pagination: cursor-based for large collections.
- Message-submission endpoints MUST require an **idempotency key** supplied by the client, following the
  accepted contract in [API idempotency](docs/architecture/api-idempotency.md) (ADR-0018).
- Monetary amounts are transmitted as strings with documented currency and scale (proposed, ADR-0019).
- Resource IDs are opaque strings; never expose sequential database IDs.
- Every endpoint documents: authentication, required permission, rate limit, and error cases.
- Request body size limits and server timeouts are always set.

---

## 10. Database standards

### 10.1 Migrations
- All schema changes are **versioned migration files** in `backend/migrations/` (tool per ADR-0014).
- Migration file names use a **timestamp** prefix to avoid numbering collisions between parallel agents.
- A merged migration is **immutable**. Fix mistakes with a new migration.
- Production migrations are forward-only; every PR with a migration describes its rollback plan.
- Breaking changes use **expand → migrate → contract** across separate releases.
- Never drop or rename columns/tables holding data without explicit owner approval.
- Create indexes on large tables `CONCURRENTLY`; set `lock_timeout` in migrations touching hot tables.

### 10.2 Naming conventions
- `snake_case` everywhere. Tables **plural** (`messages`), columns **singular**.
- Primary key `id`; foreign keys `<referenced_singular>_id` (`tenant_id`).
- Timestamps end in `_at` and use `timestamptz`. Booleans start with `is_` / `has_`.
- Constraint/index names: `pk_<table>`, `fk_<table>__<column>`, `ux_<table>__<columns>`,
  `ix_<table>__<columns>`, `ck_<table>__<rule>`.
- Status-like values: `text` + `CHECK` constraint (proposed) rather than PostgreSQL enum types, because
  enums are harder to evolve.

### 10.3 Types
- IDs: **UUIDv7 is the default for newly introduced generated entity primary keys**, generated by PostgreSQL 18's
  `uuidv7()` ([decision log §18, G1-1](docs/engineering/decision-log.md#18-owner-decisions--database-role-separation-and-tenant-registry-phase-4-group-1-2026-11-26)).
  Natural and composite primary keys remain valid where appropriate; other exceptions need a concrete technical
  justification. Never expose sequential integers publicly.
- Money/credits/rates: exact types only, with documented currency and scale. **Never**
  `float`/`real`/`double precision`. The representation (scaled integer vs. `numeric`) and scale are
  **open** (ADR-0019) — do not choose them in a migration without an accepted ADR.
- `jsonb` only for genuinely unstructured data (e.g. raw provider payloads), never to avoid modelling.

### 10.4 Transactions
- Keep transactions short. **No external network calls inside a transaction.**
- Use row locks (`SELECT ... FOR UPDATE`) or atomic conditional updates for balances and state changes.
- Retry on serialization failures and deadlocks with bounded attempts.
- A business change **and** the outbox record announcing it are written in the same transaction
  (transactional outbox, §11.1).

### 10.5 Indexes
- Every foreign key is indexed. Tenant-scoped indexes lead with `tenant_id`.
- Indexes are justified by a query; check `EXPLAIN` for hot paths.
- High-write tables (messages, delivery receipts) carry the minimum number of indexes.

### 10.6 Foreign keys and constraints
- Control-plane tables use foreign keys, `NOT NULL`, `UNIQUE`, and `CHECK` constraints wherever valid.
- For very high-volume, partitioned data-plane tables, the FK strategy is decided in the architecture
  phase; integrity must then be enforced another way and tested.

### 10.7 Tenant isolation
- Every tenant-owned table has `tenant_id NOT NULL`.
- Every query on tenant data filters by `tenant_id` derived from the authenticated principal.
- Prefer composite keys/constraints that include `tenant_id` so cross-tenant references are impossible.
- **PostgreSQL Row-Level Security is adopted** as defense-in-depth for **tenant-owned control-plane tables**
  (ADR-0010 accepted 2026-11-26; decisions RLS-1 to RLS-5, decision log §12.1). It is added **on top of** the
  application tenant filters above, never instead of them, and it never replaces application authorization.
  Conditions ([RLS evaluation §10](docs/architecture/rls-evaluation.md#10-conditions-of-adoption-adopted-by-the-owner-rls-1)):
  - **Tables and policies:** every such table has RLS **enabled and forced**, with the policy
    `tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid` in both `USING` and `WITH CHECK`.
  - **Tenant context:** it is set **transaction-locally** by one application helper, which rejects missing or
    invalid tenant IDs before any tenant query. It is **never** set as a session-level setting.
  - **Application role:** the application connects as a role that is not a superuser, not a table owner and
    has no `BYPASSRLS`. **Database role separation is a prerequisite** in its own approved group (RLS-4); the
    API must not use the bootstrap superuser. Implemented in Phase 4 Group 1: the API connects as `sms_app` and
    refuses an unsafe role; migrations run as `sms_migrator`, which owns the tables and has no `BYPASSRLS`
    ([backend README, "Database roles"](backend/README.md#database-roles-phase-4-group-1-owner-decision-rls-4)).
  - **Cross-tenant access:** it uses a dedicated reporting role with its own credentials and pool and
    explicit read-only policies, and is audited in the application. **No role receives a blanket
    `BYPASSRLS`**: the migration role has `NOBYPASSRLS` (owner decision G1-9, decision log §18, superseding the
    earlier rule that reserved it for the migration role), and a cross-tenant data migration or break-glass
    access needs its own separately designed, controlled and approved procedure.
  - **Schema rules:** unique constraints and foreign keys on tenant data include `tenant_id`; views use
    `security_invoker = true`; there is no `SECURITY DEFINER` over tenant tables without review; the
    application role is never granted `TRUNCATE`.
  - **Data-plane tables:** high-volume data-plane tables are **not covered** by this decision; their RLS
    strategy is deferred (RLS-5).
- Every tenant-scoped repository has a test proving tenant A cannot read or modify tenant B's data.
- The **RLS catalog guard** (`backend/internal/database/rlsguard`) runs in CI after the migrations and fails if
  any table with a `tenant_id` column lacks RLS enabled and forced or a tenant policy. Exceptions need owner
  approval recorded in the decision log; there are none.

### 10.8 Data retention
- Every table is assigned a retention class in its migration PR (e.g. *financial*, *audit*,
  *message content*, *message metadata*, *operational logs*). Retention periods are **not yet defined**
  — they depend on regulatory and business decisions.
- High-volume time-series data is designed for time-based partitioning so expiry is cheap.

### 10.9 Auditability
- Tables include `created_at`, `updated_at`; sensitive tables include `created_by`/`updated_by`.
- Ledger and audit tables are **append-only**: no `UPDATE`, no `DELETE` by application roles.
- Financial and audit records are never hard-deleted.

---

## 11. Messaging standards

The authoritative definitions are in [messaging reliability](docs/architecture/messaging-reliability.md)
(events, lifecycle, receipts), [API idempotency](docs/architecture/api-idempotency.md), and
[financial integrity](docs/architecture/financial-integrity.md) (see §1.1). This section summarizes them. Read the
[glossary](docs/glossary.md) if any term is unfamiliar.

### 11.1 Asynchronous processing and the transactional outbox
- Accepting a message and sending it to a telco are **separate steps**. The API accepts, validates,
  reserves credit, persists, and returns a message ID; dispatch happens asynchronously.
- A state change and its outbox record are committed **atomically**; if the transaction rolls back, the
  event is never published. A committed outbox record will eventually be published.
- The outbox publisher may publish the **same event more than once** (crash before marking, ambiguous
  acknowledgment, retries). Events carry a stable event ID. Delivery is **at-least-once**, not exactly-once.
- The queue/broker technology is deferred (ADR-0006). Code against an internal interface.

### 11.2 Idempotency
- **Consumers:** every consumer is idempotent and documents its duplicate-detection mechanism
  (processed-event record, conditional state transition, or unique business key — messaging reliability §2.4).
- **API:** message submission requires a client idempotency key per the accepted contract (ADR-0018). It
  prevents duplicate operations **inside our platform**; it does not guarantee a single telco submission.
- Our internal message ID is created once at acceptance and never changes. Telco message IDs are stored
  as a mapping, never used as our primary identity.
- **Ordering:** no global ordering is guaranteed. Where correctness depends on order, the component
  documents the scope, mechanism, and handling of out-of-order events (messaging reliability §2.5).

### 11.3 Bounded concurrency
- Worker counts, in-flight windows, and queue sizes are configured and finite.
- When downstream is saturated, apply **back-pressure** (slow intake, queue) rather than spawning more work.

### 11.4 Rate limiting
- Outbound throughput is limited **per provider, per connection, and per sender/account** according
  to contracted telco limits (to be confirmed — never assumed).
- Inbound API rate limits are per tenant and per API key.
- Exceeding a telco's limit can lead to throttling or suspension; limits are enforced centrally, not
  by each worker guessing.

### 11.5 Provider submission and retries
- Distinguish the milestones: accepted by our API → queued → submission attempted → provider
  acknowledgment → final receipt → customer-visible status. None before a final receipt proves delivery
  (messaging reliability §3).
- Classify every provider response as **success**, **retryable**, **permanent**, or **ambiguous**
  (e.g. timeout with no response). Classification lives in the provider adapter, from telco documentation.
- Retry only **retryable** outcomes, with exponential backoff and jitter, an attempt limit, and a
  **maximum message age**.
- Record a submission attempt before sending; a redelivered dispatch event must not submit again.
- **Ambiguous outcomes are never blindly retried**: the message moves to an uncertain state and is
  resolved by the per-provider policy (an owner decision, D-095).

### 11.6 Delivery receipt (DLR) processing
- Receipts may be late, missing, duplicated, conflicting, or out of order, and may arrive before our
  record of the provider acknowledgment. Processing MUST tolerate all of these.
- Store every raw receipt and every status transition in append-only history.
- No final receipt within the configured window → `UNKNOWN` (proposed name): an **uncertain** state,
  **not** proof of non-delivery. The window is configured per provider; there is no universal default.
- Late receipts are still correlated, applied by precedence rules, and eligible for reconciliation.
- State changes follow **precedence, not arrival order**; a final state is never replaced by a
  transitional one; conflicting finals are flagged, not silently overwritten.
- Duplicate receipts never cause duplicate billing, refunds, or webhooks (side effects keyed per message
  and effect type).
- Provider receipt formats and status mappings stay **inside provider adapters**.
- Final-status rules, billing treatment of uncertain states, and segment roll-up are pending owner
  decisions (D-095–D-098); do not invent them.

### 11.7 Failure handling
- Provider outages trigger circuit breaking and, where approved, failover routing — never silent loss.
- Poison messages (that always fail processing) are isolated in a dead-letter store with the reason.
- Every failure is observable: metrics, structured logs, and operator-visible state.

### 11.8 Message ordering
- SMS delivery order is **not guaranteed** by telcos. The system MUST NOT promise ordering to clients.
- Multi-part (concatenated) messages are reassembled by the handset using segment headers.

### 11.9 Billing consistency
- Credit is **reserved** at acceptance, **settled** at the point defined by the billing rules
  (submission or delivery — business decision D-080), and **released** for messages that fail before
  being chargeable.
- Reservation, settlement, release, refund, and reversal are separate ledger entries, each idempotent
  (unique key per message and entry type). Corrections are compensating entries, never edits.
- Clients are charged **per message** (partner-approved direction). The **billable quantity** of a message is
  its **segment count** — an owner proposal pending partner approval (decision log §17.4), not a settled rule.
  Segments are always computed by **one shared segmentation function** used for the estimate, usage records and
  any charge — never from the number of API requests. The charge event (provider acceptance, D-080) is also an
  owner proposal pending partner approval.
- Amounts, rates, and rounding follow [financial integrity](docs/architecture/financial-integrity.md);
  representation, scale, and rounding rules are open (ADR-0019, D-092, D-093).
- Reconciliation compares our ledger, our message states, and telco billing records; discrepancies are
  reported, never auto-corrected silently.

---

## 12. Security standards

Authoritative detail: [security baseline](docs/security/security-baseline.md) (see §1.1).

- **Secrets:** never in the repository, images, logs, or frontend bundles. Loaded from the environment
  or a secret manager at runtime. `.env` files are git-ignored; only `.env.example` with placeholders is
  committed. A leaked secret is rotated, not just deleted from git.
- **Authentication:** portal users via secure sessions (HttpOnly, Secure, SameSite cookies); passwords
  hashed with a modern adaptive algorithm (Argon2id proposed); **MFA mandatory for all human internal
  admin accounts** (Super Admin, Operations, Finance; D-021) and available for clients. API clients use
  API keys, stored only as hashes and shown once.
- **Authorization:** deny by default; role-based permissions checked server-side on every request;
  the frontend hides controls for convenience only — it is never the security boundary.
- **Input validation:** validate type, length, format, and range at every boundary (HTTP, file imports,
  webhooks, telco callbacks). Phone numbers are normalized to E.164 and validated. File uploads have
  size and row limits. Exported CSV/Excel files are protected against formula injection.
- **API security:** TLS only; per-tenant and per-key rate limits; request size limits; CORS restricted
  to known origins; CSRF protection for cookie-authenticated endpoints.
- **Webhooks:** outbound payloads are signed (HMAC with timestamp); customer webhook URLs are validated
  against SSRF (no private, loopback, or metadata addresses).
- **Tenant isolation:** see §10.7. Admin access to a tenant's data is audited.
- **Audit logging:** all security-relevant and financial actions (login, role change, API key creation,
  credit adjustment, pricing change, sender ID approval, provider configuration, impersonation) are
  recorded in an append-only audit log with actor, tenant, action, target, timestamp, and source IP.
- **Sensitive information:** phone numbers and message content are personal data under the Philippine
  Data Privacy Act of 2012 (RA 10173). Minimize collection, mask in logs, restrict access, encrypt in
  transit and at rest, and respect retention limits. OTP content is never logged or shown to admins.
- **Abuse prevention:** sender IDs must be approved before use; impersonation of banks, government, or
  other brands is a primary risk. Content and URL policies are pending (discovery register).

---

## 13. Testing requirements

Authoritative detail: [testing standards](docs/standards/testing-standards.md) (see §1.1).

- Every behaviour change ships with tests at the appropriate level (unit, integration, contract, E2E).
- **Mandatory test coverage areas** (no exceptions): ledger and credit operations (including rounding),
  API idempotency, outbox publication and duplicate event consumption, tenant isolation, authorization
  checks, segment/encoding calculation, message state transitions, ambiguous submission handling,
  DLR correlation (late, missing, duplicate, out-of-order), retry classification. Representative scenarios
  are listed in the authoritative documents and in the testing standards.
- Tests are deterministic: no real time, randomness, or network unless injected/controlled.
- **No test may send a real SMS or connect to a production telco endpoint.**
- Report the exact test commands you ran and their results in your completion report.

---

## 14. Git and pull request workflow

Full detail: [git workflow](docs/engineering/git-workflow.md).

- `main` is the default branch and is protected (owner-confirmed; see
  [git workflow §0](docs/engineering/git-workflow.md#0-repository-state)). **Never commit or push directly to
  `main`.** Never force-push shared branches. Create task branches from the latest `main`.
- One task → one branch → one pull request. Branch names: `<type>/<task-id>-<short-description>`
  (types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`). Tool-assigned branch names (e.g.
  `claude/...`) are acceptable if the PR title carries the task ID.
- Commit messages follow Conventional Commits (`feat(billing): add credit reservation`).
- PRs use the repository PR template and are opened as **drafts** until tests pass.
- Only the project owner approves and merges. Squash-merge is the default.

---

## 15. Definition of Done

A task is done only when **all** apply:

- [ ] The task's acceptance criteria are met — no more, no less.
- [ ] Code follows this file and the standards documents.
- [ ] Tests are written and pass locally; commands and results are reported.
- [ ] Lint/format checks pass (once tooling exists).
- [ ] No secrets, real personal data, or real phone numbers added.
- [ ] Schema, contract, and dependency changes are declared in the PR.
- [ ] Affected documentation is updated in the same PR.
- [ ] The completion report (§16) is provided.

---

## 16. Completion report format

End every task with this report (also paste it into the PR description):

```markdown
## Summary
<1–3 sentences: what was done and why>

## Changed files
| File | Change |
| ---- | ------ |
| path/to/file | <what changed and why> |

## Schema / contract / dependency changes
<"None" or a list with compatibility impact>

## Tests
- Commands run: `<exact commands>`
- Result: <pass/fail counts, or "not run because ...">
- Integration status: <"verified against simulator" | "mock test only" | "not tested">

## Assumptions
- <each assumption made>

## Unresolved issues / questions for the owner
- <each open item>
```

---

## 17. Decisions: how to read and propose them

- All decisions are recorded in [docs/engineering/decision-log.md](docs/engineering/decision-log.md)
  as ADRs with status **Proposed**, **Accepted**, **Rejected**, **Superseded**, or **Deprecated**.
- Only the project owner can change a status to **Accepted**. Agents MUST NOT mark their own proposals
  accepted.
- **Pricing, product scope and launch** decisions also require **business-partner approval**, recorded by the
  owner with its date and scope before the decision is labelled approved; silence is not approval, and supporting
  records stay outside Git. Earlier owner decisions in these areas are labelled *owner-approved, pending partner
  ratification*. The technical definitions of production gates are owner engineering decisions; authorizing
  production launch needs partner approval as well ([decision log §17](docs/engineering/decision-log.md#17-owner-decisions--business-decisions-review-phase-2-ex-9-2026-11-26)).
- To propose a decision, add an ADR with status *Proposed* following the template, explaining purpose,
  alternatives, trade-offs, recommendation, operational cost, missing information.
- **Decisions reserved for the project owner** include: adopting new infrastructure components,
  adding dependencies, changing architectural constraints (§6), public API breaking changes, pricing
  and billing rules, retention periods, telco routing policy, and anything with legal or regulatory effect.
  Of these, pricing and billing rules, product scope and launch also need business-partner approval.
