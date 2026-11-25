# Testing Standards

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — pending owner review | Project owner | 2026-11-24 |

**Authority:** this document is authoritative for test levels, required scenarios, and testing
expectations, as defined in [`AGENTS.md` §1.1](../../AGENTS.md#11-document-authority-and-conflicts); `AGENTS.md` §13 summarizes it. Scenario lists owned by specialist
documents (messaging reliability, API idempotency, financial integrity) are referenced here, not
redefined. It does not grant permissions, widen task scope, or override restrictions in `AGENTS.md`. If it genuinely conflicts with another document, stop the affected work, report the conflict, and request owner resolution ([`AGENTS.md` §1.1](../../AGENTS.md#11-document-authority-and-conflicts)).

## 1. Why testing matters more here

Most code will be written by AI agents. Automated tests are the owner's main evidence that work is
correct. In this platform, bugs can **send real messages to real people**, **charge customers wrongly**,
or **leak one customer's data to another** — so the most important behaviours have mandatory tests.

## 2. Universal rules

1. Every behaviour change includes tests. Bug fixes include a test that fails before the fix.
2. Tests are **deterministic**: inject clocks, ID generators, and randomness; no `sleep`-based timing.
3. Tests are **independent**: no reliance on execution order or shared mutable state.
4. Test data is **synthetic**. No real phone numbers, customer names, or message content.
5. **No test sends a real SMS or connects to a production telco endpoint.** Telco tests use the
   simulator; telco *test-environment* tests are manual, explicitly labelled, and never in CI by default.
6. Agents report exact commands and results. A test that was not run is reported as not run.
7. Never weaken, skip, or delete a failing test to make a build pass. Fix the cause or report it.

## 3. Test levels

### 3.1 Unit tests

- **Scope:** a function or small group of types, no I/O.
- **Go:** standard `testing`, table-driven, `t.Parallel()` where safe, run with `-race`.
- **Frontend:** a Vite-native test runner (Vitest proposed) + Testing Library for components (test
  behaviour the user sees, not implementation details).
- **Must cover:** segment and encoding calculation (GSM-7 basic/extended, UCS-2, boundaries 160/161,
  70/71, multi-part), phone normalization, pricing calculation, state transitions, retry classification,
  permission checks, validation rules.

### 3.2 Integration tests

- **Scope:** a module with its real dependencies — PostgreSQL in a container, the HTTP router, the
  simulator provider.
- **Tooling (proposed):** containerized PostgreSQL started per test package (e.g. via Testcontainers for Go
  or Docker Compose in CI) with migrations applied.
- **Must cover:**
  - Ledger: concurrent reservations cannot overspend (parallel test), idempotent settlement, release on
    failure, balance equals ledger sum, rounding per the decided rules, no update/delete of entries
    (scenarios FI-1 to FI-10 in [financial integrity §7](../architecture/financial-integrity.md#7-representative-test-scenarios-documentation-not-code)).
  - Tenant isolation: tenant A cannot read/update/delete tenant B's resources via repository **and** API.
    If RLS is adopted: missing tenant context returns no rows, and a reused pooled connection cannot see
    the previous tenant's data (ADR-0010).
  - API idempotency: same key + same body replays; same key + different body is rejected; concurrent
    duplicates produce one operation and one reservation (scenarios IK-1 to IK-11 in
    [API idempotency §4](../architecture/api-idempotency.md#4-representative-test-scenarios-documentation-not-code)).
  - Outbox and event consumption: a rolled-back transaction publishes nothing; a committed event is
    published after a publisher crash; **duplicate publication occurs and is tolerated**; duplicate and
    concurrent consumption produce one business effect (scenarios OB-1 to OB-9 in
    [messaging reliability §6.1](../architecture/messaging-reliability.md#61-outbox-and-event-delivery)).
    Tests must not assert that an event is published exactly once.
  - Provider submission: timeouts and crashes after sending lead to an uncertain state, not automatic
    resubmission (scenarios PS-1 to PS-7).
  - DLR handling: late (after `UNKNOWN`), missing, duplicate, out-of-order, conflicting, and unmatched
    receipts; no duplicate billing side effects (scenarios DR-1 to DR-10).
  - Migrations: apply cleanly to an empty database and to the previous schema version.
- Run with `go test -short` excluded or a build tag (decided at scaffolding) so unit tests stay fast.

### 3.3 Contract tests

- **API contracts:** server responses are validated against the OpenAPI specification in `contracts/`;
  generated frontend types must compile against the current spec.
- **Event contracts:** producers and consumers are tested against versioned event schemas.
- **Provider contract suite:** one shared suite in `tests/provider-contract/` that every provider adapter
  must pass against a simulator configured for that provider's **documented** capabilities (successful
  submit, throttling, permanent errors, ambiguous timeouts, receipt variations including late, duplicate,
  and out-of-order receipts, status mapping, reconnect after session loss). Simulator profiles must cite the
  telco documentation they emulate; undocumented behaviour is not simulated as fact.
- **Webhook contracts:** payload shape and signature verification.

### 3.4 End-to-end tests

- **Scope:** browser tests of critical user journeys against a locally running stack with the simulator.
- **Tooling (proposed):** Playwright.
- **Initial critical journeys:** login with MFA (admin); create sender ID and approve; top up credit;
  send a message and see its delivered status; import contacts; create and approve a campaign;
  opt-out enforcement; API key creation and use.
- Keep E2E tests few and stable; push detail down to lower levels.

### 3.5 Load and performance tests

- **Purpose:** verify the system meets **verified** targets and find bottlenecks — not to guess capacity.
- **Prerequisite:** targets from [traffic capacity discovery](../requirements/traffic-capacity-discovery.md).
  Until then, load tests establish *baselines* (e.g. the ceiling of the PostgreSQL-backed queue, ADR-0006).
- **Tooling (proposed):** k6 for HTTP, plus Go-based load generators for the data plane; simulator
  configured with realistic latency and telco TPS limits.
- **Scenarios:** steady state, campaign burst, provider outage and recovery, DLR flood, slow database,
  soak (hours) to detect leaks.
- **Measured:** throughput, latency percentiles (p50/p95/p99), queue age, error rates, resource usage,
  ledger correctness after the run.
- Never run load tests against shared or production environments without owner approval.

### 3.6 Security testing

- **Static analysis:** `govulncheck`, `golangci-lint` security linters, dependency audit for frontend
  packages (`pnpm audit` in CI, since pnpm is the accepted package manager — ADR-0013).
- **Secret scanning:** a pre-commit or CI secret scanner (tool to be chosen) plus GitHub secret scanning
  where available.
- **Automated security tests:** authorization matrix tests (every role × every endpoint), tenant isolation
  tests, input validation (oversized payloads, malformed phone numbers, CSV formula injection), webhook
  SSRF protection, rate limits.
- **Before production:** an independent penetration test and a review against the security baseline.

## 4. Mandatory test areas (summary)

| Area | Minimum levels |
| ---- | -------------- |
| Ledger / credits / rounding | Unit + integration (including concurrency) |
| Idempotency (API and consumers) | Unit + integration (including concurrent duplicates) |
| Outbox publication and duplicate events | Integration (including crash/restart simulation) |
| Ambiguous provider submissions | Unit + provider contract |
| DLR anomalies (late, missing, duplicate, out-of-order, conflicting) | Unit + integration |
| Tenant isolation | Integration + API |
| Authorization (RBAC) | Unit + API matrix |
| Segmentation / encoding | Unit (exhaustive boundaries) |
| Message state machine and DLR correlation | Unit + integration |
| Retry classification and backoff | Unit + provider contract |
| Provider adapters | Provider contract suite |
| Public API | Contract + integration |

## 5. Coverage

Line-coverage percentage is reported but is **not** the goal. The mandatory areas above must be
thoroughly covered (all branches of financial and authorization logic). Coverage targets may be set per
module once code exists.

## 6. Continuous integration (to be built at scaffolding)

Every PR runs: formatting check, lint, unit tests (with `-race` for Go), integration tests, contract
checks, frontend type check and tests, dependency vulnerability scan, secret scan. A PR cannot merge
while required checks fail.
