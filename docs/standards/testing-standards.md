# Testing Standards

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — pending owner review | Project owner | 2026-11-24 |

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
    failure, balance equals ledger sum.
  - Tenant isolation: tenant A cannot read/update/delete tenant B's resources via repository **and** API.
  - Idempotency: repeated API submission with the same key yields one message and one charge.
  - Outbox: events are published exactly for committed changes.
  - DLR handling: late, duplicate, out-of-order, unmatched receipts.
  - Migrations: apply cleanly to an empty database and to the previous schema version.
- Run with `go test -short` excluded or a build tag (decided at scaffolding) so unit tests stay fast.

### 3.3 Contract tests

- **API contracts:** server responses are validated against the OpenAPI specification in `contracts/`;
  generated frontend types must compile against the current spec.
- **Event contracts:** producers and consumers are tested against versioned event schemas.
- **Provider contract suite:** one shared suite in `tests/provider-contract/` that every provider adapter
  must pass against a simulator configured for that provider (successful submit, throttling, permanent
  errors, timeouts, DLR variations, reconnect after session loss).
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

- **Static analysis:** `govulncheck`, `golangci-lint` security linters, dependency audit for npm packages
  (proposed: `npm audit`/`pnpm audit` in CI).
- **Secret scanning:** a pre-commit or CI secret scanner (tool to be chosen) plus GitHub secret scanning
  where available.
- **Automated security tests:** authorization matrix tests (every role × every endpoint), tenant isolation
  tests, input validation (oversized payloads, malformed phone numbers, CSV formula injection), webhook
  SSRF protection, rate limits.
- **Before production:** an independent penetration test and a review against the security baseline.

## 4. Mandatory test areas (summary)

| Area | Minimum levels |
| ---- | -------------- |
| Ledger / credits | Unit + integration (including concurrency) |
| Idempotency (API and consumers) | Unit + integration |
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
