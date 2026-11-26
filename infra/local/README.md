# infra/local/

The local development stack, defined in [`compose.yaml`](compose.yaml) (Phase 3 Groups 2 and 6;
[decision log §11](../../docs/engineering/decision-log.md#11-owner-decisions--local-development-stack-and-expanded-ci-phase-3-group-6-2026-11-26);
database roles from Phase 4 Group 1,
[decision log §18](../../docs/engineering/decision-log.md#18-owner-decisions--database-role-separation-and-tenant-registry-phase-4-group-1-2026-11-26)).
It is for local development only and is never used in production.

| Service | What it is | Published on your machine |
| ------- | ---------- | ------------------------- |
| `postgres` | PostgreSQL 18 (`postgres:18.7-trixie`, pinned by digest; the same image as backend CI). Data in the named volume `sms-local_postgres-data`. `DB_USER` (`sms`) is its bootstrap superuser | `127.0.0.1:${DB_PORT}` (default 5432) |
| `migrate` | One-off: runs `migrate up` as the migration role **`sms_migrator`** once PostgreSQL is healthy, then exits | Nothing |
| `api` | The Go API, connected as the application role **`sms_app`**. Starts only after `migrate` has **completed successfully**. Health check: `/readyz` | `127.0.0.1:${API_PORT}` (default 8080) |

- **Client portal:** not containerised. It runs natively with `pnpm dev` and reaches the API through
  its development proxy (see [section 4](#4-start-the-client-portal-optional)).
- **Ports:** every published port is bound to **127.0.0.1 only**, so nothing is reachable from your
  network. The API's metrics listener stays inside the container and is not published. For metrics,
  run the API natively (see [`backend/README.md`](../../backend/README.md)).
- **Images:** `migrate` and `api` use one image, `sms-local-backend:dev`, built from
  [`infra/docker/backend.Dockerfile`](../docker/backend.Dockerfile). It holds three static binaries
  (`api`, `migrate`, `healthcheck`) on a distroless base image.
- **Database roles:** `migrate` and `api` never use the bootstrap superuser. See
  [section 9](#9-database-roles-and-upgrading-an-existing-database-volume).
- **Container security:** both run as the unprivileged user 65532, with a read-only filesystem, no Linux
  capabilities and `no-new-privileges`. PostgreSQL also runs with `no-new-privileges`; Group 6 verified
  that the image still initialises, restarts and serves with it.

## Requirements

| Tool | Version |
| ---- | ------- |
| Docker | Docker Desktop (Windows, macOS) or Docker Engine with the Compose plugin (Linux) |
| Docker Compose | **v2.7.0 or later** for the documented `up --wait` start (see note below) |
| Node.js and pnpm | Only for the client portal: Node.js 26.13.1, pnpm 10.34.9 ([`frontend/README.md`](../../frontend/README.md)) |

Check your version with `docker compose version`. Docker Desktop releases from 2024 onward include a
new-enough Compose.

**Compose version note.** The stack starts the one-off `migrate` service before the API, using
`depends_on: condition: service_completed_successfully`, and is started with `up --wait`.
- **Why v2.7.0:** before Compose v2.7.0, `up --wait` expected every service to be *running or
  healthy*, which a one-off service that has exited never is. v2.7.0 (July 2022) was the first release to
  make `--wait` wait for such a service to *complete successfully* instead. This comes from Compose's own
  source history (commit `fd8538f7`, "Use appropriate dependency condition for one-shot containers when
  running `compose up --wait`"; `getDependencyCondition` in `pkg/compose/start.go` is present from
  v2.7.0 and absent in v2.6.1).
- **What was actually run:** older releases were not executed here. The stack was run with Compose
  v5.7.1 locally and with the Compose version on GitHub's `ubuntu-24.04` runner in CI.
- **Recommendation:** use a current Docker Desktop or Compose plugin.

Run every command below from the **repository root**. They are the same in Windows PowerShell and in a
Linux or macOS shell. Compose finds `infra/local/.env` by itself, because it reads the `.env` file next
to `compose.yaml`.

## 1. One-time setup

Create your local settings file and choose **three** different local-only passwords: `DB_PASSWORD` (the
bootstrap superuser), `DB_MIGRATOR_PASSWORD` (`sms_migrator`) and `DB_APP_PASSWORD` (`sms_app`):

```powershell
Copy-Item infra/local/.env.example infra/local/.env
notepad infra/local/.env      # change the three passwords; change DB_PORT or API_PORT if already in use
```

```bash
cp infra/local/.env.example infra/local/.env
${EDITOR:-nano} infra/local/.env
```

- `infra/local/.env` is ignored by Git. Never commit it, and never put a real (shared, staging or
  production) password in it.
- The passwords have **no defaults**: without any of them, Compose refuses to start and says so.
- PostgreSQL reads `DB_PASSWORD` only when the database volume is first created. If you change it
  later, role provisioning cannot authenticate until you restore it or reset the volume (section 7b).
- The two role passwords are set when a new volume is initialised. To change them later, or if your
  volume existed before role separation, follow [section 9](#9-database-roles-and-upgrading-an-existing-database-volume)
  (no data is lost).

## 2. Start the stack (one command)

```powershell
docker compose -f infra/local/compose.yaml up -d --build --wait
```

What happens:
1. The backend image is built. The first time this also downloads the base images and Go modules,
   which takes a few minutes; later builds reuse the cache.
2. `postgres` starts and becomes healthy. On a **new** volume it first runs
   [`postgres-init/10-sms-roles.sh`](postgres-init/10-sms-roles.sh), which creates `sms_migrator` and
   `sms_app` (logged as `sms-roles: sms_migrator and sms_app provisioned`).
3. `migrate` applies any pending migrations and exits.
4. `api` starts and becomes healthy.

`--wait` returns when all of this has finished, or exits with an error, for example when a migration
fails.

Rebuild after changing Go code by running the same command again; `--build` rebuilds the image.

## 3. Check the stack

```powershell
docker compose -f infra/local/compose.yaml ps -a
```

Expected:
- `postgres` and `api` show `(healthy)`;
- `migrate` shows `Exited (0)`;
- `PORTS` shows only `127.0.0.1:...` addresses.

Then:

```powershell
Invoke-RestMethod http://127.0.0.1:8080/healthz     # status: ok
Invoke-RestMethod http://127.0.0.1:8080/readyz      # status: ready
```

```bash
curl -s http://127.0.0.1:8080/healthz
```

```bash
curl -s http://127.0.0.1:8080/readyz
```

Logs and migration status:

```powershell
docker compose -f infra/local/compose.yaml logs -f api
```

```powershell
docker compose -f infra/local/compose.yaml logs migrate
```

```powershell
docker compose -f infra/local/compose.yaml run --rm migrate status
```

## 4. Start the client portal (optional)

From `frontend/`, in a second terminal (first time: `pnpm install --frozen-lockfile`):

```powershell
pnpm dev
```

Open http://localhost:5173. The Vite development server forwards **only** `/healthz` and `/readyz` to
the API on `127.0.0.1:8080`, so http://localhost:5173/readyz shows the API's readiness. The portal page
makes no API calls (owner decision L5): no API client, authentication or public API integration exists
yet.

## 5. Migrations

- **Every start runs `migrate up`** (owner decision L6). It is safe to repeat:
  - migrations already applied are skipped (`no pending migrations`);
  - a database lock stops two `migrate` runs from overlapping;
  - the API itself never migrates.
- **A new migration file** in `backend/migrations/` is picked up by the next
  `up -d --build --wait` (the image embeds the files).
- **If a migration fails:**
  - that migration is rolled back;
  - `migrate` exits with code 1, and `up --wait` reports `service "migrate" didn't complete successfully`;
  - the `api` container is created but **never started**.

  Read the error with `docker compose -f infra/local/compose.yaml logs migrate`. It names the
  migration version. Fix the migration and run the start command again.
- **Rollback (`migrate down`) is not available in a container.** It requires a loopback-IP `DB_HOST`,
  and the container connects to `postgres`, so it is refused by design. Run rollback natively against
  the published port, as described in [`backend/README.md`](../../backend/README.md) ("Migrations").

## 6. Stop the stack (keeps your data)

```powershell
docker compose -f infra/local/compose.yaml stop
```

Or remove the containers but keep the data:

```powershell
docker compose -f infra/local/compose.yaml down
```

The API shuts down gracefully: it gets 20 s, longer than its own 15 s shutdown timeout. Start again with
section 2. The named volume keeps your data through `stop`, `down`, restarts and image rebuilds.

## 7. Reset and clean up

Choose the smallest reset that solves your problem.

**a) Re-run migrations only (keeps the database).** Roll back natively with `migrate down` (see
[`backend/README.md`](../../backend/README.md)), then start the stack again, which applies them again.

**b) Delete everything and start from an empty database.**

> **WARNING — this permanently deletes all data in your local database.** The `-v` flag removes the
> named volume `sms-local_postgres-data`. There is no undo. It affects only this local Compose project
> (`sms-local`); it never touches another project's volumes or any remote database. Do not run it if
> you need anything stored locally.

First check exactly which volume will be deleted:

```powershell
docker volume ls --filter name=sms-local_postgres-data
```

Then delete it and start fresh (migrations run automatically):

```powershell
docker compose -f infra/local/compose.yaml down -v
```

```powershell
docker compose -f infra/local/compose.yaml up -d --build --wait
```

**c) Also remove the locally built image** (it is rebuilt on the next start):

```powershell
docker compose -f infra/local/compose.yaml down --rmi local
```

## 8. Database only (for running Go natively)

To run the API or migrations with `go run`, start only PostgreSQL and follow the native steps in
[`backend/README.md`](../../backend/README.md) ("Running locally"):

```powershell
docker compose -f infra/local/compose.yaml up -d --wait postgres
```

Stop the containerised API first (`docker compose -f infra/local/compose.yaml stop api`) if it is
running, because both use port 8080.

## 9. Database roles and upgrading an existing database volume

The stack uses three PostgreSQL roles (Phase 4 Group 1, owner decision RLS-4):

| Role | Used by | What it may do |
| ---- | ------- | -------------- |
| `sms` (`DB_USER`, bootstrap superuser) | the `postgres` container, role provisioning, `psql` for inspection | Everything. Never used by `migrate` or `api`. |
| `sms_migrator` | `migrate` | Owns the application tables and creates them. Not a superuser; no `BYPASSRLS`, `CREATEROLE` or `CREATEDB`. |
| `sms_app` | `api` | Reads and writes only what each migration grants. Owns nothing, cannot create objects, never `TRUNCATE`. |

Both roles are defined by one idempotent script,
[`backend/internal/database/dbroles/provision.sql`](../../backend/internal/database/dbroles/provision.sql),
run by [`postgres-init/10-sms-roles.sh`](postgres-init/10-sms-roles.sh), which also sets their passwords
from `DB_MIGRATOR_PASSWORD` and `DB_APP_PASSWORD`. The script never drops a role, a table or any data.

**New volume:** nothing to do. The PostgreSQL image runs the script once, when it initialises the empty
volume.

**Existing volume** (created before role separation, or after changing a role password in `.env`): the
image never runs initialisation scripts again, so run the script once yourself. Your data is kept: the
script only creates or corrects the two roles, sets their privileges, and hands goose's version table
(`goose_db_version`, created earlier by `sms`) to `sms_migrator`.

1. Add `DB_MIGRATOR_PASSWORD` and `DB_APP_PASSWORD` to `infra/local/.env` (see `.env.example`).
2. Recreate only the `postgres` container with the new settings (the volume and its data are kept):

   ```powershell
   docker compose -f infra/local/compose.yaml up -d --wait postgres
   ```

3. Run the provisioning script inside it (it prints `sms-roles: sms_migrator and sms_app provisioned`):

   ```powershell
   docker compose -f infra/local/compose.yaml exec postgres bash /docker-entrypoint-initdb.d/10-sms-roles.sh
   ```

4. Start the stack as usual; `migrate` applies the new migrations as `sms_migrator`:

   ```powershell
   docker compose -f infra/local/compose.yaml up -d --build --wait
   ```

Running step 3 again is always safe. If it reports objects in `public` that `sms_migrator` does not own, it
has changed nothing; see "Troubleshooting".

**Alternative: start from an empty database instead.** Only if you do not need anything in your local
database, you can reset it (section 7b); the new volume then gets the roles automatically.

> **WARNING — the reset permanently deletes all data in your local database** (the named volume
> `sms-local_postgres-data`). There is no undo. Nothing in this repository runs it for you: it happens
> only if you type the `down -v` command yourself. The upgrade above keeps your data and is the
> recommended path.

## Inspect the database

Open `psql` inside the container (no local PostgreSQL client needed):

```powershell
docker compose -f infra/local/compose.yaml exec postgres psql -U sms -d sms
```

Useful commands inside `psql`: `\dt` (tables and their owners), `\du` (roles),
`SELECT * FROM goose_db_version;` (applied migrations), `\q` (quit). This connects as the bootstrap
superuser; replace `sms` if you changed `DB_USER` or `DB_NAME`.

## Integration tests and this database

The backend integration tests run natively and connect to the published port. They **never modify your
`sms` database**: each test creates its own database named `sms_test_<random>` and drops it when the
test finishes, even if it fails. They run only with `APP_ENV=test` and `DB_HOST` set to a loopback IP
(`127.0.0.1`), and refuse to start otherwise (see [`backend/README.md`](../../backend/README.md),
"Testing"). They connect as the bootstrap superuser (`DB_USER`, `DB_PASSWORD`) to create their databases,
and as `sms_migrator` and `sms_app` with `DB_MIGRATOR_PASSWORD` and `DB_APP_PASSWORD`; load all of them
from `infra/local/.env` first (backend README, step 3). They never change a role's password or attributes:

```powershell
$env:APP_ENV = "test"; $env:DB_HOST = "127.0.0.1"
cd backend; go test -tags integration ./...
```

If a test process is killed, a leftover database can be removed safely (it only ever contains test
data). First list them:

```powershell
docker compose -f infra/local/compose.yaml exec postgres psql -U sms -d sms -c "SELECT datname FROM pg_database WHERE datname LIKE 'sms\_test\_%'"
```

Then drop one by name:

```powershell
docker compose -f infra/local/compose.yaml exec postgres psql -U sms -d sms -c "DROP DATABASE sms_test_<name> WITH (FORCE)"
```

## Automated check of the whole stack

[`scripts/local-stack-check.sh`](../../scripts/local-stack-check.sh) is what CI runs ("Local stack
check"). Run it on Linux, macOS or WSL from the repository root:

```bash
bash scripts/local-stack-check.sh
```

What it checks:
- fresh start;
- database roles: created on a new volume, table ownership, the API's role, what `sms_app` cannot do;
- the API refusing an unsafe role, and provisioning repairing a changed role;
- security settings and published ports;
- idempotent restart;
- refused in-container rollback;
- a failed migration blocking the API;
- persistence;
- volume reset;
- the upgrade of a volume created before role separation (section 9), with its data kept.

It uses its own project, `sms-local-check`, and deletes only that project's containers, volume and
image. It never touches your `sms-local` data.

## Troubleshooting

| Symptom | Cause and fix |
| ------- | ------------- |
| `required variable DB_PASSWORD is missing a value` | `infra/local/.env` is missing or has no `DB_PASSWORD`. Do section 1. |
| `required variable DB_MIGRATOR_PASSWORD is missing a value` (or `DB_APP_PASSWORD`) | Your `infra/local/.env` predates role separation. Add both passwords, then follow section 9. |
| `migrate` exits 1 with `password authentication failed for user "sms_migrator"` (or `role "sms_migrator" does not exist`) | The volume existed before role separation, or a role password changed. Run the provisioning command in section 9; no data is lost. |
| `migrate` exits 1 with `refusing to use database role ... as the migration role` | `migrate` was started with a privileged `DB_USER` (for example the bootstrap superuser). Use `sms_migrator`. |
| `api` exits 1 with `refusing to use database role ... as the application role: ...` | The API's role is unsafe (the message lists why, for example `has BYPASSRLS` or `owns tables`). Use `sms_app`; if `sms_app` itself was changed, run the provisioning command in section 9, which restores its attributes. |
| Provisioning fails with `objects in schema public are not owned by sms_migrator: ...` | Something in `public` was created by another role (for example by hand as `sms`). Nothing was changed. Transfer it (`ALTER TABLE ... OWNER TO sms_migrator`) or drop it if it is disposable, then run the command again. |
| `Bind for 127.0.0.1:5432 failed: port is already allocated` (or 8080) | Another PostgreSQL or API is running. Stop it, or set `DB_PORT` / `API_PORT` in `.env`. Changing `API_PORT` breaks the portal's development proxy, which expects 8080. |
| `up --wait` ends with `service "migrate" didn't complete successfully` | A migration failed and the API was not started. Read `logs migrate` (section 5). |
| `unknown flag: --wait` or the start never returns | Compose is older than the minimum version in "Requirements". Update Docker Desktop or the Compose plugin. |

## Windows (Docker Desktop) manual checklist

GitHub's Windows runners cannot run Linux containers, so CI verifies the stack itself on Linux only
(owner decision L8). CI does check on Windows that PowerShell finds `infra/local/.env` from the
repository root, that ports are bound to 127.0.0.1, and that a missing password is refused ("Local stack
config (Windows)").

**Last validation: 2026-11-26, by the owner. Completed with no failures reported.** Versions:

- Docker Desktop 4.100.1;
- Docker Engine 29.9.3;
- Docker Compose 5.7.1.

The boxes below are left unticked so the procedure can be reused. Repeat it after a change to the Compose file,
the Dockerfile or the PostgreSQL image, and record the new result here.

Run this once on Windows with Docker Desktop (WSL 2 backend), in PowerShell from the repository root, and
report the result in the pull request or to the owner:

1. [ ] `docker compose version` reports at least the minimum version in "Requirements".
2. [ ] Without `infra/local/.env`, `docker compose -f infra/local/compose.yaml config` fails with
       `DB_PASSWORD is not set` (or another required password).
3. [ ] After section 1, `docker compose -f infra/local/compose.yaml up -d --build --wait` exits without
       an error.
4. [ ] `docker compose -f infra/local/compose.yaml ps -a` shows `postgres` and `api` healthy, `migrate`
       `Exited (0)`, and only `127.0.0.1:` ports.
5. [ ] `Invoke-RestMethod http://127.0.0.1:8080/readyz` returns `status: ready`.
6. [ ] `docker compose -f infra/local/compose.yaml run --rm migrate status` lists every migration as applied.
7. [ ] From `frontend/`, `pnpm dev`, then http://localhost:5173/readyz in a browser shows
       `{"status":"ready"}`.
8. [ ] `docker compose -f infra/local/compose.yaml stop`, then section 2 again: the start succeeds and
       `migrate` reports `no pending migrations`.
9. [ ] `docker compose -f infra/local/compose.yaml down -v`, then section 2 again: a fresh database is
       created and migrated.
10. [ ] `docker compose -f infra/local/compose.yaml down -v --rmi local` cleans up.
11. [ ] (Phase 4 Group 1) After step 3, `docker compose -f infra/local/compose.yaml logs postgres` contains
        `sms-roles: sms_migrator and sms_app provisioned`, and
        `docker compose -f infra/local/compose.yaml exec postgres bash /docker-entrypoint-initdb.d/10-sms-roles.sh`
        prints the same line again (section 9, idempotent).

> **Re-validation needed.** Phase 4 Group 1 changed the Compose file (role passwords and the
> provisioning mount). The owner's validation above predates it; the checklist, including item 11 and an
> upgrade of an existing volume (section 9), has not been run on Windows since.
