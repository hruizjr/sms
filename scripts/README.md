# scripts/

Developer and CI helper scripts (e.g. "run all tests", "regenerate API types", "start local stack").

## Rules

- Scripts must be **idempotent** and **non-interactive** so that AI agents and CI can run them safely.
- Start Bash scripts with `#!/usr/bin/env bash` and `set -euo pipefail`.
- Each script begins with a comment describing what it does, its inputs, and its side effects.
- Scripts must never contain secrets, never contact real telco endpoints, and never send SMS.
- Prefer a single documented entry point (a `Makefile` or task runner, to be decided in scaffolding)
  over many ad-hoc scripts.

## Scripts

| Script | What it does | Side effects |
| ------ | ------------ | ------------ |
| [`local-stack-check.sh`](local-stack-check.sh) | End-to-end check of the Docker Compose development stack (Phase 3 Group 6): fresh start, security settings, loopback-only ports, idempotent restart, refused in-container rollback, a failed migration blocking the API, persistence, volume reset. Run by the "Local stack check" CI job; run it locally with `bash scripts/local-stack-check.sh` (Linux, macOS, WSL) | Uses its own Compose project `sms-local-check` (refuses `sms-local`) and deletes that project's containers, volume and image when it finishes. Temporarily writes one invalid migration into `backend/migrations/` and always removes it. Reads `infra/local/.env` |
| [`rls-evaluation.sh`](rls-evaluation.sh) | Runs the PostgreSQL Row-Level Security evaluation tests (Phase 3 Group 7, [evaluation](../docs/architecture/rls-evaluation.md)) against a disposable PostgreSQL started from the pinned image, then checks that no test role or database remains. `RUN_BENCHMARKS=1` adds the manual, indicative benchmark (`RLS_EVAL_BENCH_ROWS`, `BENCH_TIME`, `BENCH_COUNT`). Run with `bash scripts/rls-evaluation.sh` (Linux, macOS, WSL; needs Docker and Go 1.27.4) | Starts one container `sms-rlseval-<random>` (data on tmpfs, port on 127.0.0.1 only, random throwaway password, not part of any Compose project) and always removes it. Never touches the local Compose stack or its volume |

A single task runner is still not decided; until then, each script is documented here and in the README
it supports.
