# backend/

The Go backend: one Go module (`github.com/hruizjr/sms/backend`) that builds one or more executables
(ADR-0002, ADR-0003). Phase 3 is built in reviewed groups
([decision log §7](../docs/engineering/decision-log.md#7-owner-decisions--phase-3-walking-skeleton-2026-11-25)).

**Current state — Group 1 (Go API skeleton):** an HTTP API process with configuration, structured
logging, `/healthz`, `/readyz`, Prometheus metrics on a separate non-public listener, graceful shutdown,
tests, and CI. There is **no database, no business functionality, and no SMS sending**.

## Layout

```
backend/
├── go.mod                      Module github.com/hruizjr/sms/backend; toolchain go1.27.4
├── .golangci.yml               Linter configuration (golangci-lint v2)
├── .env.example                Every configuration variable, with defaults
├── cmd/
│   └── api/main.go             Entry point: load config → build dependencies → run → shut down
└── internal/
    ├── config/                 Environment configuration: parsing, defaults, validation
    ├── httpserver/             Router, /healthz and /readyz, problem responses, listeners, shutdown
    ├── middleware/             Request ID, access log, metrics, panic recovery, security headers, body limit
    └── observability/          slog logger and Prometheus metrics registry
```

Business modules (identity, tenancy, billing, messaging, …) will be added under `internal/` in later
phases, each owning its own data (AGENTS.md §5, §7.1). `cmd/migrate` and the database arrive in Group 2.

## How a request flows

```
client ──▶ application listener (HTTP_ADDR)
            └─ chi router
                ├─ RequestID        accept a safe X-Request-ID or generate one; echo it in the response
                ├─ SecurityHeaders  nosniff, restrictive CSP, no-referrer, no-store
                ├─ AccessLog        one structured log line: request_id, method, route, status, duration_ms
                ├─ Metrics          sms_http_requests_total / sms_http_request_duration_seconds by route pattern
                ├─ Recover          panic → logged server-side, generic 500 problem to the client
                ├─ LimitRequestBody 1 MiB maximum
                └─ handler          /healthz, /readyz; anything else → 404/405 problem (RFC 9457)

Prometheus ──▶ metrics listener (METRICS_ADDR, loopback by default) ──▶ GET /metrics only
```

## Endpoints

| Listener | Endpoint | Behaviour |
| -------- | -------- | --------- |
| Application | `GET /healthz` | Liveness. `200 {"status":"ok"}` while the process runs. Checks no dependencies; reveals no configuration. |
| Application | `GET /readyz` | Readiness. `200 {"status":"ready"}` once both listeners are serving; `503` problem while starting or shutting down. **No database check yet** — Group 2 adds one. |
| Metrics | `GET /metrics` | Prometheus exposition format. Not served on the application listener. |

Errors use RFC 9457 problem documents (`application/problem+json`) with type `about:blank`, the
HTTP status title, a fixed client-safe detail, and the request ID. Internal error text, panic values,
and stack traces are never returned. Application-specific problem types come with the OpenAPI contract
(Group 3).

## Configuration

All variables are optional; defaults are safe for local development (both listeners bind to
`127.0.0.1`). Invalid values stop the process at startup with an error naming each offending variable.
See also [`.env.example`](.env.example).

| Variable | Default | Rules |
| -------- | ------- | ----- |
| `APP_ENV` | `development` | `development`, `test`, or `production` |
| `HTTP_ADDR` | `127.0.0.1:8080` | `host:port`; host empty (all interfaces), `localhost`, or an IP; port 1–65535 |
| `HTTP_READ_TIMEOUT` | `15s` | Positive Go duration (`5s`, `1m30s`) |
| `HTTP_READ_HEADER_TIMEOUT` | `5s` | Positive duration, not greater than `HTTP_READ_TIMEOUT` |
| `HTTP_WRITE_TIMEOUT` | `15s` | Positive duration |
| `HTTP_IDLE_TIMEOUT` | `60s` | Positive duration |
| `SHUTDOWN_TIMEOUT` | `15s` | Positive duration; graceful-shutdown deadline for in-flight requests |
| `METRICS_ADDR` | `127.0.0.1:9090` | `host:port`; must be loopback unless `METRICS_ALLOW_NON_LOOPBACK=true`; must not share the application listener |
| `METRICS_ALLOW_NON_LOOPBACK` | `false` | Explicit opt-in for container deployments; the metrics listener must never be publicly reachable |
| `LOG_FORMAT` | `text` in development, otherwise `json` | `text` or `json` |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

The process logs only an explicit, non-sensitive startup summary (environment, log level, log format,
listener addresses) — never the environment wholesale. No secrets are configured yet.

## Running locally

### Toolchain

The module pins Go **1.27.4** (`toolchain go1.27.4` in `go.mod`). Install any Go release ≥ 1.21; with the
default `GOTOOLCHAIN=auto`, the `go` command downloads and verifies Go 1.27.4 automatically the first time
you run it in `backend/`. To install 1.27.4 directly instead, use the installer from
<https://go.dev/dl/>.

### Windows (PowerShell)

```powershell
cd backend
go version                       # should report go1.27.4 (downloaded automatically if needed)
go run ./cmd/api                 # Ctrl+C to stop gracefully
```

In a second PowerShell window:

```powershell
Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz
(Invoke-WebRequest http://127.0.0.1:9090/metrics -UseBasicParsing).Content | Select-String sms_http
```

To change configuration for one session:

```powershell
$env:LOG_FORMAT = "json"; $env:LOG_LEVEL = "debug"
go run ./cmd/api
```

### macOS / Linux

```bash
cd backend && go run ./cmd/api
```

```bash
curl -s http://127.0.0.1:8080/healthz
```

```bash
curl -s http://127.0.0.1:9090/metrics | grep sms_http
```

## Checks (same commands as CI)

Run from `backend/`. CI uses the pinned versions below (see `.github/workflows/backend-ci.yml`).

```bash
gofmt -l .                                   # must print nothing
go vet ./...
go test ./...
go test -race ./...                          # needs a C compiler (cgo); on Windows, run in CI or WSL
go build -o bin/api ./cmd/api
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.10.0 ./...
```

`govulncheck` analyses the standard library of the `go` found on your `PATH`, so run it with Go 1.27.4
itself on `PATH` (as CI does). If your `PATH` points to an older Go that only delegates to 1.27.4,
govulncheck reports type errors in the standard library rather than results.

## Rules

Follow [`AGENTS.md`](../AGENTS.md) §7 (Go standards) and
[`docs/standards/coding-standards.md`](../docs/standards/coding-standards.md). Create packages and
migrations only within an approved Phase 3 group task. Walking-skeleton migrations contain no business
tables.
