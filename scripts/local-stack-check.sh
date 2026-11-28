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
# - The unsafe-role tests change sms_app (BYPASSRLS, then a membership in sms_migrator) on this
#   project's own server only, and restore the approved state before continuing.
# - The messaging test creates a synthetic tenant and API key with smsctl in this project's database
#   and sends synthetic messages to the in-process simulator only (no SMS, no network connection).
# - The portal test creates synthetic portal users with smsctl (passwords from standard input) in this
#   project's database and signs in with curl against the API only.
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
# The expected migrations are derived from the migration files themselves (never hard-coded), so a
# new migration is expected automatically and a database that has not applied it fails the check.
migration_versions() { # every migration file's version, ascending, one per line
  local f name
  for f in backend/migrations/[0-9]*_*.sql; do
    name="$(basename "$f")"
    [[ "$f" == "$BROKEN_MIGRATION" || ! -f "$f" ]] && continue
    echo "${name%%_*}"
  done | sort -n
}
EXPECTED_VERSIONS="$(migration_versions | paste -sd ' ' -)"
LATEST_VERSION="$(migration_versions | tail -n 1)"
if [[ ! "$LATEST_VERSION" =~ ^[0-9]{14}$ ]]; then
  echo "could not determine the latest migration version from backend/migrations" >&2
  exit 2
fi
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
# Runs the provisioning script while sms_app is a member of sms_migrator; succeeds if provisioning
# refuses (fails closed, changing nothing) and names the membership as the reason. It checks
# sms_migrator first, which then "has members".
provisioning_refuses_membership() {
  local out
  out="$(provision_roles 2>&1)" && return 1
  grep -q 'role sms_migrator is a member of another role, or has members' <<<"$out"
}
# Runs the API once as sms_migrator (a table owner); succeeds if the API refuses and says why.
api_refuses_migrator_role() {
  local password out
  password="$(compose exec -T postgres printenv SMS_MIGRATOR_PASSWORD)"
  out="$(DB_PASSWORD="$password" timeout 60 docker compose run --rm --no-deps -e DB_USER=sms_migrator -e DB_PASSWORD api 2>&1)" && return 1
  grep -q 'refusing to use database role "sms_migrator" as the application role: owns tables' <<<"$out"
}
api_sessions_as() { psql_value "SELECT coalesce(string_agg(DISTINCT usename, ','), '') FROM pg_stat_activity WHERE application_name = 'sms-api'"; }
db_version() { psql_value "SELECT max(version_id) FROM goose_db_version WHERE is_applied"; }
# The applied versions (as goose records them), ascending, space-separated.
applied_versions() { psql_value "SELECT string_agg(v::text, ' ' ORDER BY v) FROM (SELECT DISTINCT version_id AS v FROM goose_db_version WHERE is_applied AND version_id > 0) a"; }
# Succeeds if `migrate status` lists every migration file as applied and none as pending.
migrate_status_all_applied() {
  local out
  out="$(docker compose run --rm --no-deps migrate status 2>&1)" || return 1
  ! grep -qi 'pending' <<<"$out" || return 1
  local v
  for v in $EXPECTED_VERSIONS; do
    grep -Eq "^$v[[:space:]]+applied" <<<"$out" || return 1
  done
}
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
expect "database version is the latest migration ($LATEST_VERSION, from the migration files)" equals "$(db_version)" "$LATEST_VERSION"
expect "every migration file is applied" equals "$(applied_versions)" "$EXPECTED_VERSIONS"
expect "migrate status lists every migration as applied, none pending" migrate_status_all_applied
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

echo "== A role membership is refused; provisioning fails closed until it is revoked"
# Provisioning never removes a membership itself: it refuses and names the role (provision.sql).
# The recovery is the one its error message gives: REVOKE the membership, then provision again.
membership_count="SELECT count(*) FROM pg_auth_members WHERE member IN ('sms_app'::regrole, 'sms_migrator'::regrole) OR roleid IN ('sms_app'::regrole, 'sms_migrator'::regrole)"
psql_value "GRANT sms_migrator TO sms_app" >/dev/null
compose stop api >/dev/null 2>&1
if compose up -d --wait api >/dev/null 2>&1; then fail "api started although sms_app is a member of sms_migrator"; else pass "api does not start while sms_app is a member of sms_migrator"; fi
expect "api logged the refusal" bash -c "docker compose logs --no-log-prefix api | grep -q 'as the application role: is a member of sms_migrator'"
expect "provisioning refuses while the membership exists" provisioning_refuses_membership
expect "the refused provisioning changed nothing (membership still present)" equals "$(psql_value "$membership_count")" 1
psql_value "REVOKE sms_migrator FROM sms_app" >/dev/null
expect "provisioning succeeds after the REVOKE" provision_roles
expect "neither role has a membership" equals "$(psql_value "$membership_count")" 0
compose up -d --wait
pass "api starts again after the membership is revoked"
expect "GET /readyz is 200" equals "$(http_code /readyz)" 200
expect "the api's database sessions use sms_app" equals "$(api_sessions_as)" sms_app

echo "== End-to-end messaging through the simulator (Phase 4 Group 2)"
bootstrap="$(compose run --rm -T smsctl bootstrap --name "Stack Check Co" --sender-id EXAMPLECO 2>&1)" || true
api_key="$(grep -E '^smsk_' <<<"$bootstrap" | head -n 1 || true)"
tenant_id="$(sed -n 's/^tenant: \([0-9a-f-]*\) (active)$/\1/p' <<<"$bootstrap")"
hdr_file="$(mktemp)"
resp_body="" resp_code=""
# post_message <idempotency key> <json body>: sets resp_code and resp_body; headers go to $hdr_file.
post_message() {
  local out
  out="$(curl -s -m 10 -w '\n%{http_code}' -D "$hdr_file" -X POST "$API/v1/messages" \
    -H "Authorization: Bearer $api_key" -H "Content-Type: application/json" -H "Idempotency-Key: $1" --data "$2")"
  resp_code="$(tail -n 1 <<<"$out")"
  resp_body="$(head -n -1 <<<"$out")"
}
message_id_of() { grep -o '"id":"msg_[A-Za-z0-9]*"' <<<"$1" | head -n 1 | cut -d'"' -f4; }
status_of() { grep -o '"status":"[a-z_]*"' <<<"$1" | head -n 1 | cut -d'"' -f4; }
get_status() { status_of "$(curl -s -m 5 "$API/v1/messages/$1" -H "Authorization: Bearer $api_key")"; }
# wait_for_status <message id> <status>: up to 30 seconds (the simulator reports receipts after 2s).
wait_for_status() {
  local i
  for i in $(seq 1 30); do
    [[ "$(get_status "$1")" == "$2" ]] && return 0
    sleep 1
  done
  return 1
}
bootstrapped() { [[ -n "$tenant_id" && -n "$api_key" ]]; }
accepted_with_id() { [[ "$resp_code" == 202 && -n "$(message_id_of "$resp_body")" && "$(status_of "$resp_body")" == accepted ]]; }
replayed_same_id() { [[ "$resp_code" == 202 && "$(message_id_of "$resp_body")" == "$message_id" ]] && grep -qi '^Idempotent-Replayed: true' "$hdr_file"; }
key_reused() { [[ "$resp_code" == 422 ]] && grep -q 'problems/idempotency-key-reused' <<<"$resp_body"; }
key_not_logged() { ! compose logs --no-log-prefix 2>&1 | grep -qF "$api_key"; }

expect "smsctl bootstrap created an active tenant and printed an API key once" bootstrapped
post_message stack-check-1 '{"to":"+639000000001","sender_id":"EXAMPLECO","body":"Your code is 000000.","client_reference":"stack-1"}'
message_id="$(message_id_of "$resp_body")"
expect "POST /v1/messages is 202 with a message id and status accepted (got $resp_code)" accepted_with_id
expect "the simulator's receipt makes it delivered" wait_for_status "$message_id" delivered
post_message stack-check-1 '{"sender_id":"EXAMPLECO","to":"+639000000001","client_reference":"stack-1","body":"Your code is 000000."}'
expect "an identical retry (other key order) is replayed with the same id" replayed_same_id
post_message stack-check-1 '{"to":"+639000000001","sender_id":"EXAMPLECO","body":"Another text"}'
expect "the same key with a different body is 422 idempotency-key-reused" key_reused
post_message stack-check-2 '{"to":"+639000000030","sender_id":"EXAMPLECO","body":"Uncertain"}'
uncertain_id="$(message_id_of "$resp_body")"
expect "a simulated timeout is reported as submission_uncertain" wait_for_status "$uncertain_id" submission_uncertain
sleep 3 # several dispatcher cycles
trace="$(compose run --rm -T smsctl message trace --tenant "$tenant_id" --message-id "$uncertain_id" 2>&1 || true)"
expect "the uncertain message was handed over exactly once (never resent)" equals "$(grep -c '^  #[0-9]* provider simulator' <<<"$trace")" 1
expect "a request without an API key is 401" equals "$(http_code /v1/messages)" 401
expect "the dispatcher's database sessions use sms_app" \
  equals "$(psql_value "SELECT coalesce(string_agg(DISTINCT usename, ','), '') FROM pg_stat_activity WHERE application_name = 'sms-dispatcher'")" sms_app
expect "no log contains the API key" key_not_logged
rm -f "$hdr_file"

echo "== Client portal: session sign-in, CSRF, submission and tenant isolation (Phase 4 Group 3)"
portal_password="stack check portal password"
printf '%s\n' "$portal_password" | compose run --rm -T smsctl portal-user create --tenant "$tenant_id" \
  --email owner@stack-check.example.invalid --role owner --password-stdin >/dev/null 2>&1 || true
cookie="" csrf="" portal_code="" portal_body=""
# portal <method> <path> [curl args...]: sets portal_code and portal_body (with the session cookie).
portal() {
  local method="$1" path="$2" out
  shift 2
  out="$(curl -s -m 10 -w '\n%{http_code}' -X "$method" "$API/portal/v1/$path" ${cookie:+-H "Cookie: $cookie"} "$@")"
  portal_code="$(tail -n 1 <<<"$out")"
  portal_body="$(head -n -1 <<<"$out")"
}
portal_sign_in() { # portal_sign_in <email> <password>: sets cookie and csrf on success
  local headers
  headers="$(curl -s -m 10 -D - -o "$hdr_file" -X POST "$API/portal/v1/session" -H "Content-Type: application/json" \
    --data "{\"email\":\"$1\",\"password\":\"$2\"}")"
  cookie="$( (grep -i '^set-cookie: __Host-sms_session=' <<<"$headers" || true) | sed -E 's/^[^:]*: ([^;]*);.*/\1/' | tr -d '\r')"
  csrf="$( (grep -o '"csrf_token":"[a-z0-9]*"' "$hdr_file" || true) | cut -d'"' -f4)"
  cookie_attributes="$( (grep -i '^set-cookie: __Host-sms_session=' <<<"$headers" || true) | tr -d '\r')"
}
portal_status() { portal GET "messages/$1"; status_of "$portal_body"; }
wait_for_portal_status() {
  local i
  for i in $(seq 1 30); do
    [[ "$(portal_status "$1")" == "$2" ]] && return 0
    sleep 1
  done
  return 1
}
signed_in() { [[ -n "$cookie" && -n "$csrf" ]]; }
hardened_cookie() { grep -q 'HttpOnly' <<<"$cookie_attributes" && grep -q 'Secure' <<<"$cookie_attributes" && grep -q 'SameSite=Strict' <<<"$cookie_attributes"; }
password_not_logged() { ! compose logs --no-log-prefix 2>&1 | grep -qF "$portal_password"; }

portal_sign_in owner@stack-check.example.invalid "wrong password, not this one"
expect "a wrong password is refused (no session)" equals "$cookie" ""
portal_sign_in owner@stack-check.example.invalid "$portal_password"
expect "smsctl created a portal user who can sign in" signed_in
expect "the session cookie is HttpOnly, Secure and SameSite=Strict" hardened_cookie
portal GET session
expect "GET /portal/v1/session names the user and organization" grep -q '"email":"owner@stack-check.example.invalid"' <<<"$portal_body"
portal POST messages -H "Content-Type: application/json" -H "Idempotency-Key: portal-1" --data '{"to":"+639000000002","sender_id":"EXAMPLECO","body":"Portal stack check"}'
expect "a submission without the CSRF token is 403 (got $portal_code)" equals "$portal_code" 403
portal POST messages -H "X-CSRF-Token: $csrf" -H "Content-Type: application/json" -H "Idempotency-Key: portal-1" --data '{"to":"+639000000002","sender_id":"EXAMPLECO","body":"Portal stack check"}'
portal_message="$(message_id_of "$portal_body")"
expect "POST /portal/v1/messages with the CSRF token is 202 (got $portal_code)" equals "$portal_code" 202
portal POST messages -H "X-CSRF-Token: $csrf" -H "Content-Type: application/json" -H "Idempotency-Key: portal-1" --data '{"to":"+639000000002","sender_id":"EXAMPLECO","body":"Portal stack check"}'
expect "a repeated portal submission returns the same message" equals "$(message_id_of "$portal_body")" "$portal_message"
expect "the portal message is delivered by the same dispatcher and simulator" wait_for_portal_status "$portal_message" delivered
expect "the portal message is visible through the public API of the same tenant" equals "$(get_status "$portal_message")" delivered
portal GET messages
expect "the portal list shows both portal and API messages" bash -c "grep -q '$portal_message' <<<'$portal_body' && grep -q '$message_id' <<<'$portal_body'"
# A second tenant's user sees nothing of the first tenant.
other="$(compose run --rm -T smsctl tenant create --name "Stack Check Other Co" 2>&1)" || true
other_tenant="$(sed -n 's/^tenant: \([0-9a-f-]*\) (active)$/\1/p' <<<"$other")"
printf '%s\n' "$portal_password" | compose run --rm -T smsctl portal-user create --tenant "$other_tenant" \
  --email other@stack-check.example.invalid --role viewer --password-stdin >/dev/null 2>&1 || true
owner_cookie="$cookie" owner_csrf="$csrf"
portal_sign_in other@stack-check.example.invalid "$portal_password"
portal GET "messages/$portal_message"
expect "another tenant's user gets 404 for the message (got $portal_code)" equals "$portal_code" 404
portal GET messages
expect "another tenant's user lists no messages" equals "$portal_body" '{"data":[],"next_cursor":null}'
portal POST messages -H "X-CSRF-Token: $csrf" -H "Content-Type: application/json" -H "Idempotency-Key: v-1" --data '{"to":"+639000000002","sender_id":"EXAMPLECO","body":"x"}'
expect "a viewer cannot submit (got $portal_code)" equals "$portal_code" 403
portal GET api-keys
expect "a viewer cannot list API keys (got $portal_code)" equals "$portal_code" 403
cookie="$owner_cookie" csrf="$owner_csrf"

echo "== Client portal: API key management (Phase 4 Group 4)"
key_request() { printf '{"name":"Stack check key","scopes":["messages:send","messages:read"],"current_password":"%s"}' "$1"; }
portal POST api-keys -H "Content-Type: application/json" --data "$(key_request "$portal_password")"
expect "creating an API key without the CSRF token is 403 (got $portal_code)" equals "$portal_code" 403
portal POST api-keys -H "X-CSRF-Token: $csrf" -H "Content-Type: application/json" --data "$(key_request "not the password")"
expect "creating an API key with a wrong current password is 403 (got $portal_code)" bash -c "[[ '$portal_code' == 403 ]] && grep -q reauthentication-failed <<<'$portal_body'"
portal POST api-keys -H "X-CSRF-Token: $csrf" -H "Content-Type: application/json" --data "$(key_request "$portal_password")"
expect "POST /portal/v1/api-keys with the current password is 201 (got $portal_code)" equals "$portal_code" 201
portal_key="$(grep -o '"key":"smsk_[0-9a-z_]*"' <<<"$portal_body" | cut -d'"' -f4)"
portal_key_id="$(grep -o '"id":"[0-9a-z]*"' <<<"$portal_body" | head -n 1 | cut -d'"' -f4)"
portal_api_send() {
  curl -s -m 10 -o /dev/null -w '%{http_code}' -X POST "$API/v1/messages" -H "Authorization: Bearer $portal_key" \
    -H "Content-Type: application/json" -H "Idempotency-Key: $1" --data '{"to":"+639000000003","sender_id":"EXAMPLECO","body":"Key check"}'
}
expect "a key created in the portal authenticates on the public API" equals "$(portal_api_send key-1)" 202
key_secret="${portal_key##*_}"
listed_without_secret() { grep -q "$portal_key_id" <<<"$portal_body" && ! grep -qF "$key_secret" <<<"$portal_body"; }
revoked_in_body() { grep -q '"status":"revoked"' <<<"$portal_body"; }
key_not_logged() { ! compose logs --no-log-prefix 2>&1 | grep -qF "$key_secret"; }
portal GET api-keys
expect "the key list shows the new key without its secret" listed_without_secret
portal DELETE "api-keys/$portal_key_id" -H "X-CSRF-Token: $csrf"
expect "DELETE /portal/v1/api-keys/{id} is 200 (got $portal_code)" equals "$portal_code" 200
expect "the key is listed as revoked" revoked_in_body
expect "the revoked key is refused by the public API" equals "$(portal_api_send key-2)" 401
expect "no log contains the new key's secret" key_not_logged
portal DELETE session -H "X-CSRF-Token: $csrf"
expect "sign-out is 204 (got $portal_code)" equals "$portal_code" 204
portal GET session
expect "the signed-out session no longer works (got $portal_code)" equals "$portal_code" 401
expect "no log contains the portal password" password_not_logged
rm -f "$hdr_file"

echo "== Security settings"
for svc in api migrate dispatcher; do
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
expect "the check detects a database that has not applied the latest migration" \
  fails equals "$(db_version)" "$LATEST_VERSION"
expect "the check detects a database that has not applied every migration" \
  fails equals "$(applied_versions)" "$EXPECTED_VERSIONS"
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
expect "every migration file is applied after the upgrade" equals "$(applied_versions)" "$EXPECTED_VERSIONS"
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
