#!/usr/bin/env bash
# Local stack check — verifies the Docker Compose development stack end to end (Phase 3 Group 6;
# database roles and the existing-volume upgrade procedure added in Phase 4 Group 1).
#
# Run by the "Local stack check" CI job; developers can run it on Linux, macOS, or WSL from the
# repository root:
#   bash scripts/local-stack-check.sh
#
# Safety:
# - It uses its own Compose project, "sms-local-check" (override with COMPOSE_PROJECT_NAME), and
#   refuses to run as "sms-local", so it never touches your own database volume. It deletes only that
#   project's containers, volume and locally built image.
# - It reads DB_PASSWORD, DB_MIGRATOR_PASSWORD, DB_APP_PASSWORD and the other settings from
#   infra/local/.env, like the stack itself.
# - The upgrade test recreates, in this project's own throwaway volume only, the state of a volume
#   created before role separation (it drops and recreates the application roles there).
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
  echo "infra/local/.env is missing: copy infra/local/.env.example and set the passwords" >&2
  exit 2
fi

P="$COMPOSE_PROJECT_NAME"
API="http://127.0.0.1:${API_PORT:-8080}"
BROKEN_MIGRATION="backend/migrations/29991231235959_local_stack_check_broken.sql"
BASELINE_VERSION="20261125041208"
LATEST_VERSION="20261126175603"
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
# Runs SQL as the application role over TCP inside the postgres container (password from its env).
psql_app() { compose exec -T postgres sh -c 'PGPASSWORD="$SMS_APP_PASSWORD" psql -h 127.0.0.1 -U sms_app -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 -Atc "$1"' _ "$1"; }
provision_roles() { compose exec -T postgres bash /docker-entrypoint-initdb.d/10-sms-roles.sh; }
fails() { ! "$@"; }
# Runs the API once as sms_migrator (a table owner); succeeds if the API refuses and says why.
api_refuses_migrator_role() {
  local password out
  password="$(compose exec -T postgres printenv SMS_MIGRATOR_PASSWORD)"
  out="$(DB_PASSWORD="$password" timeout 60 docker compose run --rm --no-deps -e DB_USER=sms_migrator -e DB_PASSWORD api 2>&1)" && return 1
  grep -q 'refusing to use database role "sms_migrator" as the application role: owns tables' <<<"$out"
}
api_sessions_as() { psql_value "SELECT coalesce(string_agg(DISTINCT usename, ','), '') FROM pg_stat_activity WHERE application_name = 'sms-api'"; }
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
expect "database version is the latest migration" equals "$(db_version)" "$LATEST_VERSION"
expect "GET /healthz is 200" equals "$(http_code /healthz)" 200
expect "GET /readyz is 200" equals "$(http_code /readyz)" 200
expect "api health status is healthy" equals "$(inspect api '{{.State.Health.Status}}')" healthy

echo "== Database roles (fresh volume: created by postgres-init/10-sms-roles.sh)"
expect "postgres ran the role provisioning script at initialisation" \
  bash -c "docker compose logs --no-log-prefix postgres | grep -q 'sms-roles: sms_migrator and sms_app provisioned'"
expect "sms_migrator and sms_app: login, no superuser, BYPASSRLS, CREATEROLE, CREATEDB" \
  equals "$(psql_value "SELECT string_agg(format('%s:%s/%s/%s/%s/%s', rolname, rolcanlogin, rolsuper, rolbypassrls, rolcreaterole, rolcreatedb), ' ' ORDER BY rolname) FROM pg_roles WHERE rolname IN ('sms_app', 'sms_migrator')")" \
  "sms_app:t/f/f/f/f sms_migrator:t/f/f/f/f"
expect "every application table is owned by sms_migrator" \
  equals "$(psql_value "SELECT count(*) FROM pg_tables WHERE schemaname = 'public' AND tableowner <> 'sms_migrator'")" 0
expect "the api's database sessions use sms_app" equals "$(api_sessions_as)" sms_app
expect "sms_app reads the tenant registry" equals "$(psql_app 'SELECT count(*) FROM tenants')" 0
expect "sms_app cannot create a table" fails psql_app 'CREATE TABLE app_made (id int)'
expect "sms_app cannot truncate tenants" fails psql_app 'TRUNCATE tenants'

echo "== The API refuses an unsafe role, and provisioning repairs a changed role"
expect "api run as sms_migrator (a table owner) exits non-zero with the reason" api_refuses_migrator_role
psql_value "ALTER ROLE sms_app BYPASSRLS" >/dev/null
compose stop api >/dev/null 2>&1
if compose up -d --wait api >/dev/null 2>&1; then fail "api started although sms_app has BYPASSRLS"; else pass "api does not start while sms_app has BYPASSRLS"; fi
expect "api logged the refusal" bash -c "docker compose logs --no-log-prefix api | grep -q 'as the application role: has BYPASSRLS'"
expect "provisioning again succeeds (idempotent)" provision_roles
expect "provisioning removed BYPASSRLS" equals "$(psql_value "SELECT rolbypassrls FROM pg_roles WHERE rolname = 'sms_app'")" f
compose up -d --wait
pass "api starts again after provisioning"
expect "GET /readyz is 200" equals "$(http_code /readyz)" 200

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
expect "database version unchanged" equals "$(db_version)" "$LATEST_VERSION"

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
expect "the failed migration was rolled back (version unchanged)" equals "$(db_version)" "$LATEST_VERSION"
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
expect "fresh database after reset has every migration" equals "$(db_version)" "$LATEST_VERSION"
expect "previous data is gone" \
  equals "$(psql_value "SELECT to_regclass('public.local_stack_check_persistence') IS NULL")" t

echo "== Existing volume from before role separation: documented upgrade procedure (no data loss)"
compose down -v >/dev/null 2>&1
compose up -d --wait postgres
# Recreate the state of a volume created before Phase 4 Group 1, in this check's throwaway volume: no
# application roles, default privileges, and goose's version table owned by the bootstrap superuser
# with the baseline applied (the DDL goose v3 uses).
dbname="$(psql_value 'SELECT current_database()')"
psql_value "DROP OWNED BY sms_migrator, sms_app; DROP ROLE sms_migrator; DROP ROLE sms_app;
  GRANT CONNECT, TEMPORARY ON DATABASE \"$dbname\" TO PUBLIC;
  GRANT USAGE ON SCHEMA public TO PUBLIC;
  CREATE TABLE goose_db_version (id integer PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
    version_id bigint NOT NULL, is_applied boolean NOT NULL, tstamp timestamp NOT NULL DEFAULT now());
  INSERT INTO goose_db_version (version_id, is_applied) VALUES (0, true), ($BASELINE_VERSION, true);" >/dev/null
old_rows="$(psql_value "SELECT string_agg(version_id || '@' || tstamp, ',' ORDER BY id) FROM goose_db_version")"
expect "the old volume has no application roles" \
  equals "$(psql_value "SELECT count(*) FROM pg_roles WHERE rolname IN ('sms_app', 'sms_migrator')")" 0
if compose up -d --build --wait >/dev/null 2>&1; then
  fail "up --wait should fail before the upgrade (sms_migrator does not exist)"
else
  pass "before the upgrade, up --wait fails (migrate cannot log in as sms_migrator)"
fi
expect "api was never started" equals "$(inspect api '{{.State.StartedAt}}')" "0001-01-01T00:00:00Z"
expect "documented upgrade command succeeds" provision_roles
expect "goose_db_version now belongs to sms_migrator" \
  equals "$(psql_value "SELECT tableowner FROM pg_tables WHERE tablename = 'goose_db_version'")" sms_migrator
compose up -d --build --wait
pass "after the upgrade, up --build --wait succeeded"
expect "the remaining migrations were applied" equals "$(db_version)" "$LATEST_VERSION"
expect "the existing migration history was kept" \
  equals "$(psql_value "SELECT string_agg(version_id || '@' || tstamp, ',' ORDER BY id) FROM goose_db_version WHERE version_id <= $BASELINE_VERSION")" "$old_rows"
expect "the api's database sessions use sms_app" equals "$(api_sessions_as)" sms_app
expect "GET /readyz is 200" equals "$(http_code /readyz)" 200
expect "running the upgrade command again changes nothing (idempotent)" provision_roles

echo
if (( failures > 0 )); then
  echo "Local stack check: $failures check(s) FAILED"
  compose ps -a || true
  compose logs --no-color --tail 50 || true
  exit 1
fi
echo "Local stack check: all checks passed"
