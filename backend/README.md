# backend/

The Go backend: one Go module (`github.com/hruizjr/sms/backend`) that builds two executables, `api` and
`migrate` (ADR-0002, ADR-0003). Phase 3 is built in reviewed groups
([decision log §7](../docs/engineering/decision-log.md#7-owner-decisions--phase-3-walking-skeleton-2026-11-25)).

**Current state — Groups 1, 2 and 4:** an HTTP API with configuration, structured logging, `/healthz`,
`/readyz` (which checks PostgreSQL), Prometheus metrics on a separate loopback-only listener, graceful
shutdown, a PostgreSQL connection pool, goose migrations, sqlc configuration, Go types and Chi routes
generated from the public API contract (not mounted in production; see
[OpenAPI code generation](#openapi-code-generation-public-api-v1)), unit and integration tests, and CI.
There are **no business tables, no business functionality, no `/v1` endpoints, and no SMS sending**.

## Layout

```
backend/
├── go.mod                      Module github.com/hruizjr/sms/backend; toolchain go1.27.4
├── .golangci.yml               Linter configuration (golangci-lint v2)
├── .env.example                Every configuration variable, with defaults
├── sqlc.yaml                   sqlc configuration (pgx/v5); no queries yet
├── tools/                      Separate Go module that version-locks code generators (oapi-codegen)
├── cmd/
│   ├── api/main.go             HTTP API: load config → open pool → serve → shut down → close pool
│   └── migrate/main.go         Migrations: up | status | version | down
├── migrations/                 Versioned goose SQL migrations, embedded into cmd/migrate
├── queries/                    SQL queries for sqlc (empty until the first business feature)
└── internal/
    ├── config/                 Environment configuration: parsing, defaults, validation
    ├── database/               PostgreSQL pool (pgxpool): Open, Ping, InTx, Close
    │   └── dbtest/             Integration-test helper: throwaway databases (build tag "integration")
    ├── migration/              goose runner: advisory lock, one transaction per migration
    ├── httpserver/             Router, /healthz and /readyz, problem responses, listeners, shutdown;
    │                           mountPublicAPI (generated /v1 routes; not called by the production router)
    ├── publicapi/              GENERATED from contracts/openapi/public-api-v1.yaml — never edit by hand
    ├── middleware/             Request ID, access log, metrics, panic recovery, security headers, body limit
    └── observability/          slog logger and Prometheus metrics registry
```

Business modules (identity, tenancy, billing, messaging, …) will be added under `internal/` in later
phases, each owning its own data (AGENTS.md §5, §7.1).

## How the API uses PostgreSQL

```
cmd/api ── config.Load ──▶ database.Open (pgxpool, lazy: no connection yet)
        ── startup Ping (bounded by DB_CONNECT_TIMEOUT) ── failure is logged, not fatal
        ── httpserver.Run
              GET /healthz ──▶ 200 always while the process serves HTTP (no database check)
              GET /readyz  ──▶ listeners serving? ──▶ Ping (bounded: 2 s) ──▶ 200 ready | 503 problem
        ── SIGTERM: stop accepting → finish in-flight requests → close the pool
```

- **One pool per process**, created at startup and shared (AGENTS.md §7.5). Its size is bounded by
  `DB_MAX_CONNS`; requests beyond that wait for a free connection (bounded by their context).
- **Connections are established lazily.** The API starts even if PostgreSQL is down, reports `503` on
  `/readyz`, and recovers by itself when PostgreSQL becomes reachable. Invalid *configuration*, by contrast,
  stops the process at startup.
- **Each connection** has a bounded connect timeout (`DB_CONNECT_TIMEOUT`) and is recycled after
  `DB_MAX_CONN_LIFETIME` or `DB_MAX_CONN_IDLE_TIME`. pgx health-checks idle connections every minute.
- Connections identify themselves with `application_name` `sms-api` or `sms-migrate`
  (visible in `pg_stat_activity`).
- `database.DB.InTx` runs a function in a transaction: commit on success, rollback on error or panic.
- The API **never runs migrations**. Use `cmd/migrate`.

### Liveness vs. readiness

| Endpoint | Question it answers | Depends on PostgreSQL? | Use it for |
| -------- | ------------------- | ---------------------- | ---------- |
| `GET /healthz` | Is the process alive and serving HTTP? | **No** | Restarting a hung process. A database outage must not restart the API. |
| `GET /readyz` | Can this instance serve requests now? | **Yes** (ping, 2 s timeout) | Routing traffic. `503` while starting, shutting down, or when PostgreSQL is unreachable. |

`/readyz` returns the same generic problem document for every failure. The cause (for example
`connection refused` or an authentication error) is logged server-side once when readiness starts failing
(`readiness check failed`) and once when it recovers (`readiness check recovered`); it is never returned
to the client. Under heavy load, a readiness ping waits for a free pool connection, so `/readyz` can
report `503` when the pool is exhausted.

## Endpoints

| Listener | Endpoint | Behaviour |
| -------- | -------- | --------- |
| Application | `GET /healthz` | Liveness. `200 {"status":"ok"}` while the process runs. Checks no dependencies; reveals no configuration. |
| Application | `GET /readyz` | Readiness. `200 {"status":"ready"}` when both listeners are serving **and** PostgreSQL answers a ping within 2 s; otherwise `503` problem. |
| Metrics | `GET /metrics` | Prometheus exposition format. Not served on the application listener. |

Errors use RFC 9457 problem documents (`application/problem+json`) with type `about:blank`, the
HTTP status title, a fixed client-safe detail, and the request ID. Internal error text, panic values,
database errors, and stack traces are never returned. The public API's own problem types
(`https://api.example.invalid/problems/<slug>`, contract §5) are used only by the generated `/v1` layer,
which is not mounted in production yet.

A recovered panic is logged as one `panic recovered` entry with the request ID, method, route, the
panic value's Go **type** (`panic_type`), and a stack of function names with `file:line` positions.
The raw panic value is **never logged**, because it may contain connection strings, credentials,
tokens, or personal data; the stack omits argument values for the same reason.

## Configuration

Both executables read the same environment variables. Defaults are safe for local development (all
listeners and the database host are on `127.0.0.1`). Invalid values stop the process at startup with an
error naming each offending variable; the error never contains the database password. See also
[`.env.example`](.env.example).

| Variable | Default | Rules |
| -------- | ------- | ----- |
| `APP_ENV` | `development` | `development`, `test`, or `production` |
| `HTTP_ADDR` | `127.0.0.1:8080` | `host:port`; host empty (all interfaces), `localhost`, or an IP; port 1–65535 |
| `HTTP_READ_TIMEOUT` | `15s` | Positive Go duration (`5s`, `1m30s`) |
| `HTTP_READ_HEADER_TIMEOUT` | `5s` | Positive duration, not greater than `HTTP_READ_TIMEOUT` |
| `HTTP_WRITE_TIMEOUT` | `15s` | Positive duration |
| `HTTP_IDLE_TIMEOUT` | `60s` | Positive duration |
| `SHUTDOWN_TIMEOUT` | `15s` | Positive duration; graceful-shutdown deadline for in-flight requests |
| `METRICS_ADDR` | `127.0.0.1:9090` | `ip:port` with a **loopback IP** (`127.0.0.1`, anything in `127.0.0.0/8`, or `[::1]`); must not share the application listener |
| `LOG_FORMAT` | `text` in development, otherwise `json` | `text` or `json` |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `DB_HOST` | `127.0.0.1` | One host name or IP address (no spaces, no host lists) |
| `DB_PORT` | `5432` | 1–65535 |
| `DB_NAME` | `sms` | Database name; at most 63 bytes |
| `DB_USER` | `sms` | Role name; at most 63 bytes |
| `DB_PASSWORD` | **none — required** | Secrets have no default (AGENTS.md §7.7). Read verbatim (spaces are kept). Never logged or printed. |
| `DB_SSLMODE` | `disable` in development and test; `verify-full` in production | `disable`, `require`, `verify-ca`, `verify-full`. `disable` is **rejected in production**. `allow`/`prefer` are not accepted because they silently fall back to plaintext. For `verify-ca`/`verify-full`, provide the CA with the standard `PGSSLROOTCERT` variable. |
| `DB_MAX_CONNS` | `10` | 1–100. Maximum pooled connections **per process**; the sum over all processes must stay below the server's `max_connections`. |
| `DB_MIN_CONNS` | `0` | 0–`DB_MAX_CONNS`. Connections kept open while idle. |
| `DB_CONNECT_TIMEOUT` | `5s` | Positive duration. Time to establish one connection (TCP, TLS, authentication); also bounds the startup ping. |
| `DB_MAX_CONN_LIFETIME` | `1h` | Positive duration. Connections are replaced after this age (picks up failovers and password rotation). |
| `DB_MAX_CONN_IDLE_TIME` | `30m` | Positive duration. Idle connections are closed after this time. |

The `/readyz` ping timeout is fixed at 2 seconds (`httpserver.DefaultReadinessTimeout`).

The processes log only an explicit, non-sensitive startup summary (environment, log settings, listener
addresses, database host, port, name, TLS mode, and pool size) — never the environment wholesale, the
password, or a connection string. The password is held in a `config.Secret`, which prints and logs as
`[REDACTED]`.

## Migrations

Migrations use [goose](https://github.com/pressly/goose) v3 (ADR-0014) and live in
[`migrations/`](migrations/) as plain SQL files:

- **Naming:** `<UTC timestamp YYYYMMDDHHMMSS>_<snake_case_description>.sql` (timestamps avoid numbering
  collisions between parallel agents; AGENTS.md §10.1). Each file has `-- +goose Up` and, where
  possible, `-- +goose Down` sections. A unit test enforces the naming.
- **Embedded:** the files are compiled into the `migrate` binary (`go:embed`), so it works from any
  directory.
- **Transactions:** each migration runs in its own transaction; a failure rolls that migration back and
  stops. Statements that cannot run in a transaction (for example `CREATE INDEX CONCURRENTLY`) need
  `-- +goose NO TRANSACTION` in that file.
- **Locking:** `migrate` takes a PostgreSQL session-level advisory lock, so two `migrate` processes never
  apply migrations at the same time; the second waits (retrying every 5 seconds, up to 5 minutes).
- **Ordering:** a migration older than the latest applied one is rejected (no out-of-order apply).
- **Version table:** goose records applied versions in `goose_db_version` — the only table in the
  database during the walking skeleton.
- **Never automatic:** the API does not migrate on startup. Run `migrate` explicitly.
- **Rollback fails closed:** `migrate down` runs only when **both** hold, and is refused otherwise
  before any connection is made (there is no override flag):
  - `APP_ENV` is set **explicitly** to `development` or `test`. A missing `APP_ENV` is refused (the
    general configuration would otherwise default it to `development`), and `APP_ENV=production` is
    refused: production migrations are forward-only (AGENTS.md §10.1); fix a mistake with a new
    migration.
  - `DB_HOST` is a **loopback IP address** (`127.0.0.1`, anything in `127.0.0.0/8`, or `::1`). Host
    names, including `localhost`, and every non-loopback address are refused, so a rollback can only
    reach a local database.

  `up`, `status`, and `version` are not affected by these rules.

The only migration today is `20261125041208_baseline.sql`. It contains **no SQL**: goose refuses to run
without at least one migration, and `go:embed` needs at least one file. Applying it creates only goose's
version table. The first business feature adds the first real migration.

Commands (run from `backend/` with the `DB_*` variables set; see "Running locally"):

| Command | Effect |
| ------- | ------ |
| `go run ./cmd/migrate up` | Apply all pending migrations, print each result and the new version. |
| `go run ./cmd/migrate status` | List every migration with `applied`/`pending` and its time. |
| `go run ./cmd/migrate version` | Print the current version (`0` if none applied). |
| `go run ./cmd/migrate down` | **Rollback:** undo the most recent migration. Requires an explicit `APP_ENV=development` or `test` and a loopback-IP `DB_HOST` (see above). Run it repeatedly to go further back. |

Exit codes: `0` success, `1` failure (message on stderr), `2` usage error.

## sqlc

[`sqlc.yaml`](sqlc.yaml) compiles SQL in [`queries/`](queries/) against the schema in `migrations/` into
the dedicated package `internal/database/sqlcgen` (pgx/v5 driver; no ORM). Conventions: no JSON tags on
database models, no generated interfaces, nullable columns as Go pointers, `timestamptz` as `time.Time`,
empty slices instead of nil.

**There are no queries yet, so `sqlc generate` currently stops with "no queries contained in paths".**
This is expected: sqlc requires at least one query, and the walking skeleton deliberately has no business
tables or queries. The configuration was verified by generating code from a throwaway copy with a
temporary table and query (not committed). The first approved business feature adds its migration and
first query to `queries/`, runs `sqlc generate`, and commits the generated package; CI can then check that
generated code is up to date (`sqlc diff`).

sqlc v1.31.1 needs cgo to build from source. Easiest options:

```powershell
docker run --rm -v "${PWD}:/src" -w /src sqlc/sqlc:1.31.1 generate     # from backend/; Windows, macOS, Linux
```

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1                  # Linux/macOS with a C compiler
```

## OpenAPI code generation (public API v1)

The public API contract [`contracts/openapi/public-api-v1.yaml`](../contracts/openapi/public-api-v1.yaml)
is the source of truth (ADR-0015). Go types and Chi routes are **generated** from it by
[oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) into the package
[`internal/publicapi`](internal/publicapi/) (owner decisions D1–D7, Phase 3 Group 4;
[decision log §9](../docs/engineering/decision-log.md#9-owner-decisions--go-contract-integration-phase-3-group-4-2026-11-25)).

| File | Role |
| ---- | ---- |
| `internal/publicapi/publicapi.gen.go` | **Generated. Never edit by hand.** Models, Chi routes with parameter binding, and the typed ("strict") request/response interface |
| `internal/publicapi/oapi-codegen.yaml` | Generator configuration (`models`, `chi-server`, `strict-server`) |
| `internal/publicapi/generate.go` | The `go:generate` directive |
| `internal/publicapi/publicapi_test.go` | Hand-written tests of the generated types |
| `tools/go.mod`, `tools/go.sum` | A separate module that locks oapi-codegen **v2.8.2** and all its dependencies by checksum, so they never enter the API module or the `api` binary |

The generated code needs one library at run time: `github.com/oapi-codegen/runtime` **v1.9.0** (in `go.mod`).

**Regenerate** after any change to the specification or the configuration, from `backend/`. The same
command works in Linux and macOS shells and in Windows PowerShell or `cmd` (it uses only the Go tool and
forward-slash relative paths):

```bash
go generate ./internal/publicapi/...
```

Then run `go test ./...` and commit the regenerated file together with the change that caused it. Never
fix generated code by editing it: change the specification (through its own review) or the configuration.
CI regenerates the code twice and fails if the result differs from the committed file or between runs
(step "Check generated code is up to date" in the **Go test and build** job; also run on Windows in the
**Go generate check (Windows)** job).

**What is generated, and what is not.** The generated layer routes requests, **binds** path, query and
header parameters to their declared Go types (a value that cannot be parsed, such as `limit=ten`, gives a
`400 validation-error`), reports a missing `Idempotency-Key` (`400 idempotency-key-missing`), and decodes
JSON bodies (invalid JSON gives `400 malformed-request`). It does **not** perform full OpenAPI request
validation — patterns, lengths, ranges, enum membership, required body fields, `additionalProperties:
false`, rejecting `null` where the contract does not allow it, `Content-Type` — and it does **not**
authenticate or authorize. Both are deferred to the first endpoint implementation (owner decision D6).
`TestPublicAPIBindingIsNotValidation` documents exactly which invalid requests still reach a handler.

**Absent, null, and value.** Response fields that the contract makes required and nullable
(`Message.client_reference`, `Message.status_reason`, `MessageList.next_cursor`) are Go pointers without
`omitempty`: they are always present, `null` when nil. Optional fields (`MessageCreateRequest.client_reference`,
optional `Problem` members, query parameters) are pointers with `omitempty`: absent is `nil`, never a zero
value. Go's JSON decoder also turns an explicit `null` into `nil`; rejecting `null` where the contract
forbids it is part of the deferred validation.

**Not mounted in production.** `httpserver.NewRouter` does not register any `/v1` route, so `/v1/*` still
returns the existing `404` problem (owner decision D4). `httpserver.mountPublicAPI` is the integration
point for the first endpoint implementation; today only tests call it. It always replaces the generator's
default error handlers, which would otherwise return raw Go error text as `text/plain`.

**Dependency updates.** Change the generator version only with owner approval, in `tools/`:
`go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@<version>`, then `go mod tidy`,
regenerate, and review the diff. `tools/go.mod` also raises `github.com/getkin/kin-openapi` (the
generator's spec parser) to v0.148.0, the first release fixing GO-2026-6789 and GO-2026-6795; remove that
line when the generator itself requires a fixed version.

## Running locally

### Toolchain

The module pins Go **1.27.4** (`toolchain go1.27.4` in `go.mod`). Install any Go release ≥ 1.21; with the
default `GOTOOLCHAIN=auto`, the `go` command downloads and verifies Go 1.27.4 automatically the first time
you run it in `backend/`. To install 1.27.4 directly instead, use the installer from
<https://go.dev/dl/>. You also need Docker Desktop for the local database.

### Windows (PowerShell), step by step

Run from the **repository root** unless a step says otherwise.

**1. Start PostgreSQL** (first time: create `infra/local/.env` — see
[`infra/local/README.md`](../infra/local/README.md)):

```powershell
Copy-Item infra/local/.env.example infra/local/.env   # first time only; then edit DB_PASSWORD
docker compose -f infra/local/compose.yaml up -d --wait
```

**2. Check that PostgreSQL is healthy** (`STATUS` must show `(healthy)`):

```powershell
docker compose -f infra/local/compose.yaml ps
```

**3. Configure the Go application** for this PowerShell session by loading the same `.env` file:

```powershell
Get-Content infra/local/.env | Where-Object { $_ -match '^\s*[A-Z_][A-Z0-9_]*=' } | ForEach-Object {
    $name, $value = $_ -split '=', 2
    Set-Item -Path "Env:$($name.Trim())" -Value $value
}
$env:DB_HOST = "127.0.0.1"
$env:APP_ENV = "development"
```

The variables last until you close the window. Setting `APP_ENV` explicitly matters: `migrate down`
(step 9) is refused without it. Use the IP address `127.0.0.1`, not `localhost`: rollback and the
integration tests accept only a loopback IP. Other settings (`DB_SSLMODE`, pool sizes, …) use their
defaults; see "Configuration".

**4. Run migrations:**

```powershell
cd backend
go run ./cmd/migrate up
go run ./cmd/migrate status
```

**5. Start the API** (Ctrl+C stops it gracefully):

```powershell
go run ./cmd/api
```

**6. Test readiness** in a second PowerShell window:

```powershell
Invoke-RestMethod http://127.0.0.1:8080/healthz      # status: ok
Invoke-RestMethod http://127.0.0.1:8080/readyz       # status: ready
```

With PostgreSQL stopped (step 8), `/readyz` returns `503`:

```powershell
try { Invoke-RestMethod http://127.0.0.1:8080/readyz } catch { $_.Exception.Response.StatusCode.value__ }
```

**7. Run the integration tests** (needs steps 1–3 in this window; from `backend/`). They require
`APP_ENV=test` and refuse to run otherwise:

```powershell
$env:APP_ENV = "test"
go test -tags integration ./...
$env:APP_ENV = "development"    # back to development for running the API and migrate down
```

**8. Stop PostgreSQL** (keeps your data; from the repository root):

```powershell
docker compose -f infra/local/compose.yaml stop
```

**9. Reset the local database.** To re-run migrations without deleting the database (needs
`APP_ENV=development` and `DB_HOST=127.0.0.1` from step 3; otherwise `down` is refused):

```powershell
go run ./cmd/migrate down       # repeat until "database version: 0"
go run ./cmd/migrate up
```

> **WARNING — the next command permanently deletes ALL data in your local database** (it removes the
> Docker volume `sms-local_postgres-data`). It affects only this local Compose project. Use it only when
> you want a completely empty database.

```powershell
docker compose -f infra/local/compose.yaml down -v
docker compose -f infra/local/compose.yaml up -d --wait
```

### macOS / Linux

```bash
cp infra/local/.env.example infra/local/.env      # first time only; then edit DB_PASSWORD
docker compose -f infra/local/compose.yaml up -d --wait
set -a; . infra/local/.env; set +a                # export DB_* into this shell
cd backend
go run ./cmd/migrate up
go run ./cmd/api
```

```bash
curl -s http://127.0.0.1:8080/readyz
```

```bash
curl -s http://127.0.0.1:9090/metrics | grep sms_http
```

## Testing

| Kind | Command (from `backend/`) | Needs PostgreSQL? |
| ---- | ------------------------- | ----------------- |
| Unit tests | `go test ./...` | No |
| Unit + integration tests | `APP_ENV=test go test -tags integration ./...` | Yes (`APP_ENV=test`, loopback `DB_HOST`, `DB_*` set) |

- **Unit tests** use fakes and in-memory listeners only; they never read the process environment or need
  Docker.
- **Integration tests** are the files ending in `_integration_test.go`, compiled only with the
  `integration` build tag. They connect with the `DB_*` variables, but **never use or modify `DB_NAME`
  itself**: every test creates its own empty database `sms_test_<random>` and drops it in cleanup, even
  when the test fails. Your local `sms` database and its data are never touched.
- **Safeguards (fail closed).** Because the tests create and drop databases, `internal/database/dbtest`
  refuses to start unless **all** of these hold, and fails the test with a setup message (never
  containing the password):
  - `APP_ENV=test` is set explicitly — missing, `development`, `production`, or any other value is refused;
  - the `DB_*` configuration is valid (for example, `DB_PASSWORD` is set);
  - `DB_HOST` is a **loopback IP address** (`127.0.0.1` or `::1`) — host names, including `localhost`,
    and non-loopback addresses are refused.

  Only generated `sms_test_<16 hex>` names are ever created or dropped, and never the configured
  `DB_NAME`, even if it starts with `sms_test_`. The tests are not silently skipped when the setup is wrong.
- Integration tests cover: connecting and pinging (PostgreSQL 18 verified), wrong password and missing
  database errors (without leaking the password), pool limits and closing, transaction commit and
  rollback, migrations (up, status, down, idempotency, failure rollback, the advisory lock, no business
  tables), `/readyz` with the database available and unavailable, and the `migrate` command end to end.
- CI runs both, against a PostgreSQL 18.7 service container, plus smoke tests of the built `migrate`
  and `api` binaries.

## Checks (same commands as CI)

Run from `backend/`. CI uses the pinned versions below (see `.github/workflows/backend-ci.yml`).

```bash
gofmt -l .                                   # must print nothing
go vet ./... && go vet -tags integration ./...
go test ./...
go test -race ./...                          # needs a C compiler (cgo); on Windows, run in CI or WSL
APP_ENV=test go test -tags integration -race ./...   # needs local PostgreSQL and the DB_* variables
go build -o bin/api ./cmd/api && go build -o bin/migrate ./cmd/migrate
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.10.0 ./...
```

`govulncheck` analyses the standard library of the `go` found on your `PATH`, so run it with Go 1.27.4
itself on `PATH` (as CI does). If your `PATH` points to an older Go that only delegates to 1.27.4,
govulncheck reports type errors in the standard library rather than results.

## Known follow-ups (not part of Group 2)

- **sqlc:** generate code and add a `sqlc diff` CI check once the first real query exists.
- **Windows:** the owner validates the PowerShell steps above with Docker Desktop.
- **Readiness logging:** review the access-log severity of `/readyz` 503 responses (currently `ERROR`,
  per the Group 1 rule for status ≥ 500).
- **Database roles:** before any production business schema, separate the migration role (DDL) from the
  runtime role used by the API (DML only, least privilege). Today both executables use the same
  `DB_USER`.

## Rules

Follow [`AGENTS.md`](../AGENTS.md) §7 (Go standards), §10 (database standards), and
[`docs/standards/coding-standards.md`](../docs/standards/coding-standards.md). Create packages and
migrations only within an approved task. Walking-skeleton migrations contain no business tables
(decision log §7.3); a merged migration is never edited.
