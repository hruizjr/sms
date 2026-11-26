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

A single task runner is still not decided; until then, each script is documented here and in the README
it supports.
