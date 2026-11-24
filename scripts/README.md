# scripts/

Developer and CI helper scripts (e.g. "run all tests", "regenerate API types", "start local stack").

## Rules

- Scripts must be **idempotent** and **non-interactive** so that AI agents and CI can run them safely.
- Start Bash scripts with `#!/usr/bin/env bash` and `set -euo pipefail`.
- Each script begins with a comment describing what it does, its inputs, and its side effects.
- Scripts must never contain secrets, never contact real telco endpoints, and never send SMS.
- Prefer a single documented entry point (a `Makefile` or task runner, to be decided in scaffolding)
  over many ad-hoc scripts.

No scripts exist yet.
