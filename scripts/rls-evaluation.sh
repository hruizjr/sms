#!/usr/bin/env bash
# RLS evaluation — runs the PostgreSQL Row-Level Security evaluation tests (Phase 3 Group 7,
# ADR-0010, D-141) against a DISPOSABLE PostgreSQL instance, and optionally the manual benchmark.
#
# Run from anywhere, on Linux, macOS, or WSL (needs docker and Go 1.27.4):
#   bash scripts/rls-evaluation.sh                  # functional tests (what CI runs)
#   RUN_BENCHMARKS=1 bash scripts/rls-evaluation.sh # plus the indicative benchmark (manual only)
#
# Optional settings for the benchmark: RLS_EVAL_BENCH_ROWS (default 1000000), BENCH_TIME (default
# 2s), BENCH_COUNT (default 1).
#
# Why a separate instance: the tests create PostgreSQL roles, and roles belong to the whole server,
# not to one database. This script therefore never uses the local Compose stack or its volume. It
# starts its own container from the same pinned PostgreSQL image (read from
# infra/local/compose.yaml) with:
#   - a random name (sms-rlseval-<random>), not part of any Compose project;
#   - its data directory on tmpfs (memory), so nothing is written to a volume;
#   - a random host port bound to 127.0.0.1 only;
#   - a random throwaway password generated for this run and never printed.
# The container is removed when the script exits, also on failure or Ctrl+C.
set -euo pipefail

cd "$(dirname "$0")/.."

image="$(sed -n 's/^[[:space:]]*image:[[:space:]]*\(postgres:[^[:space:]]*@sha256:[0-9a-f]\{64\}\)[[:space:]]*$/\1/p' infra/local/compose.yaml)"
if [[ -z "$image" || "$(wc -l <<<"$image")" -ne 1 ]]; then
  echo "could not read the pinned PostgreSQL image from infra/local/compose.yaml" >&2
  exit 2
fi

name="sms-rlseval-$(od -An -N4 -tx1 /dev/urandom | tr -d ' \n')"
password="$(od -An -N18 -tx1 /dev/urandom | tr -d ' \n')"

cleanup() { docker rm -f "$name" >/dev/null 2>&1 || true; }
trap cleanup EXIT

echo "== Starting disposable PostgreSQL ($image) as container $name"
docker run -d --name "$name" \
  --tmpfs /var/lib/postgresql \
  --security-opt no-new-privileges \
  -p 127.0.0.1::5432 \
  -e POSTGRES_USER=disposable_admin \
  -e POSTGRES_DB=rlseval_maintenance \
  -e POSTGRES_PASSWORD="$password" \
  "$image" >/dev/null

for _ in $(seq 1 60); do
  if docker exec "$name" pg_isready -q -h 127.0.0.1 -p 5432 -U disposable_admin -d rlseval_maintenance; then
    break
  fi
  sleep 1
done
docker exec "$name" pg_isready -q -h 127.0.0.1 -p 5432 -U disposable_admin -d rlseval_maintenance
port="$(docker port "$name" 5432/tcp | sed -n 's/^127\.0\.0\.1:\([0-9]*\)$/\1/p' | head -n 1)"
echo "   server $(docker exec "$name" psql -U disposable_admin -d rlseval_maintenance -Atc 'SHOW server_version') on 127.0.0.1:$port"

export APP_ENV=test DB_HOST=127.0.0.1 DB_PORT="$port" DB_NAME=rlseval_maintenance \
  DB_USER=disposable_admin DB_PASSWORD="$password" DB_SSLMODE=disable \
  RLS_EVAL_DISPOSABLE_SERVER=1

echo "== Functional evaluation tests (race detector)"
go -C backend test -tags integration,rlseval -race -count=1 ./internal/database/rlseval/...

if [[ "${RUN_BENCHMARKS:-}" == "1" ]]; then
  echo "== Indicative benchmark (manual only; not a CI gate)"
  go -C backend test -tags integration,rlseval -count="${BENCH_COUNT:-1}" -run '^$' \
    -bench . -benchtime="${BENCH_TIME:-2s}" -timeout 60m -v ./internal/database/rlseval/...
fi

echo "== Leftover check"
leftover="$(docker exec "$name" psql -U disposable_admin -d rlseval_maintenance -Atc \
  "SELECT count(*) FROM pg_roles WHERE starts_with(rolname, 'rlseval_')")"
leftover_dbs="$(docker exec "$name" psql -U disposable_admin -d rlseval_maintenance -Atc \
  "SELECT count(*) FROM pg_database WHERE starts_with(datname, 'sms_test_')")"
echo "   test-created roles remaining: $leftover; test databases remaining: $leftover_dbs"
if [[ "$leftover" != "0" || "$leftover_dbs" != "0" ]]; then
  echo "RLS evaluation: FAILED (objects left behind)"
  exit 1
fi
echo "RLS evaluation: all checks passed; removing container $name"
