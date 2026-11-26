#!/usr/bin/env bash
# Local stack check — verifies the Docker Compose development stack end to end (Phase 3 Group 6).
#
# Run by the "Local stack check" CI job; developers can run it on Linux, macOS, or WSL from the
# repository root:
#   bash scripts/local-stack-check.sh
#
# Safety:
# - It uses its own Compose project, "sms-local-check" (override with COMPOSE_PROJECT_NAME), and
#   refuses to run as "sms-local", so it never touches your own database volume. It deletes only that
#   project's containers, volume and locally built image.
# - It reads DB_PASSWORD and the other settings from infra/local/.env, like the stack itself.
# - The failed-migration test writes one deliberately invalid migration into backend/migrations and
#   always removes it again (also on failure or Ctrl+C).
# - Uses only docker, curl and standard shell tools. Extra Compose files can be added with the
#   standard COMPOSE_FILE variable (colon-separated); the default is infra/local/compose.yaml.
set -euo pipefail

cd "$(dirname "$0")/.."

export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-sms-local-check}"
export COMPOSE_FILE="${COMPOSE_FILE:-infra/local/compose.yaml}"
if [[ "$COMPOSE_PROJECT_NAME" == "sms-local" ]]; then
  echo "refusing to run as project sms-local: this check deletes its project's volume" >&2
  exit 2
fi
if [[ ! -f infra/local/.env ]]; then
  echo "infra/local/.env is missing: copy infra/local/.env.example and set DB_PASSWORD" >&2
  exit 2
fi

P="$COMPOSE_PROJECT_NAME"
API="http://127.0.0.1:${API_PORT:-8080}"
BROKEN_MIGRATION="backend/migrations/29991231235959_local_stack_check_broken.sql"
BASELINE_VERSION="20261125041208"
failures=0

compose() { docker compose "$@"; }
pass() { echo "  ok    $*"; }
fail() { echo "  FAIL  $*"; failures=$((failures + 1)); }
expect() { # expect <description> <command...>
  local description="$1"; shift
  if "$@" >/dev/null 2>&1; then pass "$description"; else fail "$description"; fi
}
http_code() { curl -s -o /dev/null -w '%{http_code}' -m 5 "$API$1" || true; }
psql_value() { compose exec -T postgres sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atc "$1"' _ "$1"; }
db_version() { psql_value "SELECT max(version_id) FROM goose_db_version WHERE is_applied"; }
equals() { [[ "$1" == "$2" ]]; }
inspect() { docker inspect -f "$2" "$P-$1-1"; }

cleanup() {
  rm -f "$BROKEN_MIGRATION"
  if [[ "${KEEP_STACK:-}" != "1" ]]; then
    compose down -v --rmi local --remove-orphans >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

echo "== Configuration"
compose config --quiet
pass "compose config is valid"
compose down -v --remove-orphans >/dev/null 2>&1 || true

echo "== Fresh start (new volume): postgres -> migrate -> api"
compose up -d --build --wait
pass "up --build --wait succeeded"
expect "migrate exited 0" equals "$(inspect migrate '{{.State.ExitCode}}')" 0
expect "database version is the baseline" equals "$(db_version)" "$BASELINE_VERSION"
expect "GET /healthz is 200" equals "$(http_code /healthz)" 200
expect "GET /readyz is 200" equals "$(http_code /readyz)" 200
expect "api health status is healthy" equals "$(inspect api '{{.State.Health.Status}}')" healthy

echo "== Security settings"
for svc in api migrate; do
  expect "$svc runs as user 65532" equals "$(inspect "$svc" '{{.Config.User}}')" "65532:65532"
  expect "$svc has a read-only root filesystem" equals "$(inspect "$svc" '{{.HostConfig.ReadonlyRootfs}}')" true
  expect "$svc drops all capabilities" equals "$(inspect "$svc" '{{.HostConfig.CapDrop}}')" "[ALL]"
  expect "$svc has no-new-privileges" equals "$(inspect "$svc" '{{.HostConfig.SecurityOpt}}')" "[no-new-privileges:true]"
done
expect "postgres has no-new-privileges" equals "$(inspect postgres '{{.HostConfig.SecurityOpt}}')" "[no-new-privileges:true]"
expect "api process runs as uid 65532" equals "$(docker top "$P-api-1" -o uid,pid | awk 'NR > 1 { print $1 }' | sort -u)" 65532

echo "== Published ports: loopback only, metrics never published"
bindings="$(for c in $(compose ps -aq); do docker inspect -f '{{range $p, $b := .HostConfig.PortBindings}}{{range $b}}{{$p}}={{.HostIp}} {{end}}{{end}}' "$c"; done | tr ' ' '\n' | sed '/^$/d' | sort | tr '\n' ' ')"
expect "only 5432 and 8080 are published, both on 127.0.0.1 (got: $bindings)" \
  equals "$bindings" "5432/tcp=127.0.0.1 8080/tcp=127.0.0.1 "
expect "metrics port 9090 is not reachable from the host" \
  bash -c '! curl -s -m 3 -o /dev/null http://127.0.0.1:9090/metrics'

echo "== Repeated start is idempotent"
compose up -d --wait
pass "second up --wait succeeded"
expect "migrate reported no pending migrations" bash -c "docker compose logs --no-log-prefix migrate | grep -q 'no pending migrations'"
expect "database version unchanged" equals "$(db_version)" "$BASELINE_VERSION"

echo "== Rollback is refused inside the container (DB_HOST is not a loopback IP)"
expect "migrate down in a container exits non-zero" bash -c '! docker compose run --rm --no-deps migrate down'

echo "== A failed migration stops the API from starting"
printf -- '-- Temporary, written by scripts/local-stack-check.sh; never commit.\n-- +goose Up\nCREATE TABLE local_stack_check_partial (id integer);\nTHIS IS NOT SQL;\n' > "$BROKEN_MIGRATION"
compose down >/dev/null 2>&1
if compose up -d --build --wait >/dev/null 2>&1; then
  fail "up --wait should fail when a migration fails"
else
  pass "up --wait failed"
fi
expect "migrate exited 1" equals "$(inspect migrate '{{.State.ExitCode}}')" 1
expect "api container was created but never started" \
  equals "$(inspect api '{{.State.Status}} {{.State.StartedAt}}')" "created 0001-01-01T00:00:00Z"
expect "API is not serving" equals "$(http_code /healthz)" 000
expect "the failed migration was rolled back (version unchanged)" equals "$(db_version)" "$BASELINE_VERSION"
expect "no partial table was left behind" \
  equals "$(psql_value "SELECT to_regclass('public.local_stack_check_partial') IS NULL")" t
rm -f "$BROKEN_MIGRATION"
compose up -d --build --wait
pass "after removing the broken migration, up --build --wait succeeded"
expect "GET /readyz is 200 again" equals "$(http_code /readyz)" 200

echo "== Data persists across stop/start and down/up"
psql_value "CREATE TABLE local_stack_check_persistence (id integer); INSERT INTO local_stack_check_persistence VALUES (42)" >/dev/null
compose stop
expect "api stopped gracefully (exit code 0, not killed)" equals "$(inspect api '{{.State.ExitCode}}')" 0
expect "api closed its database pool on shutdown" bash -c "docker compose logs --no-log-prefix api | grep -q 'database pool closed'"
compose up -d --wait
expect "data survives stop/start" equals "$(psql_value 'SELECT id FROM local_stack_check_persistence')" 42
compose down
expect "volume is kept by down" docker volume inspect "${P}_postgres-data"
compose up -d --wait
expect "data survives down/up" equals "$(psql_value 'SELECT id FROM local_stack_check_persistence')" 42

echo "== Volume reset"
compose down -v
expect "volume is removed by down -v" bash -c "! docker volume inspect ${P}_postgres-data"
compose up -d --wait
expect "fresh database after reset has only the baseline" equals "$(db_version)" "$BASELINE_VERSION"
expect "previous data is gone" \
  equals "$(psql_value "SELECT to_regclass('public.local_stack_check_persistence') IS NULL")" t

echo
if (( failures > 0 )); then
  echo "Local stack check: $failures check(s) FAILED"
  compose ps -a || true
  compose logs --no-color --tail 50 || true
  exit 1
fi
echo "Local stack check: all checks passed"
