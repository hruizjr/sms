# Windows (Docker Desktop) validation — Phase 4 Group 1 (PR #17)

This checklist validates the database role separation and the tenant registry
([decision log §18](../../docs/engineering/decision-log.md#18-owner-decisions--database-role-separation-and-tenant-registry-phase-4-group-1-2026-11-26))
on Windows with Docker Desktop. It has two parts:

- **Part A — fresh database.** Runs in a separate, throwaway Compose project (`sms-winval`) on other ports
  (PostgreSQL `55432`, API `18080`). It never touches your own stack or its data.
- **Part B — upgrade of your existing database volume** (`sms-local_postgres-data`), keeping all data.

**Safety rules for this checklist:**

- No command in it deletes a volume or data. It contains **no `down -v`**, no `docker volume rm` and no
  `migrate down`.
- Changes to role attributes (A8) happen only in the throwaway project.
- If an "Expected" result does not match, **stop**, do not try to repair anything, and report the step ID with
  its output.
- **Never paste password values** into the report. The commands below print variable **names** only. The API
  and `migrate` never print passwords.

## Validation record

| Date | Commit | Environment | Part A (fresh database) | Part B (existing-volume upgrade) |
| ---- | ------ | ----------- | ----------------------- | -------------------------------- |
| 2026-11-26 | `1ce253a` | The owner's Windows computer: Windows PowerShell 5.1.26100.9818, Docker Desktop 4.98.0 (245422), Docker Engine 29.9.2, Docker Compose v5.6.1 | **Passed**: steps 0.1–0.5 and A1–A11 | **Not tested.** The computer had no `sms-local_postgres-data` volume; no project database was modified |

Notes from the owner's run:

- **A7, fifth statement** (`SELECT count(*) FROM goose_db_version` as `sms_app`): it showed the expected
  `permission denied` error, but its exit code was not separately echoed. The other four statements are
  recorded as passed.
- **Throwaway volume:** `sms-winval_postgres-data` was retained on purpose. It holds only the synthetic
  validation tenant.
- **Part B on Windows:** the existing-volume upgrade remains **unverified on Windows**. It is verified on
  Linux only, by `scripts/local-stack-check.sh` in the "Local stack check" CI job and by the agent's
  simulation described below. Run Part B on a Windows computer that has an existing volume before relying on
  it there.

## How to run

Run everything in **PowerShell** (Windows PowerShell 5.1 or PowerShell 7), from the **repository root**, in one
window, in order.

**How this checklist was checked:** every Docker command, SQL query and expected result was run on Linux with
Docker Engine and the same PostgreSQL 18.7 image, through the equivalent bash commands. Part B used a simulated
pre-PR volume. The agent could not execute the PowerShell syntax itself, because no PowerShell was available in its
environment. The owner's run on 2026-11-26 (Windows PowerShell 5.1) executed steps 0.1–0.5 and A1–A11
without a syntax error. **Part B's PowerShell steps have not been executed.** Report each step as *pass* or *fail*, using the results table at the end.

## New environment variables (PR #17)

| Variable | Where you set it | Used by | Notes |
| -------- | ---------------- | ------- | ----- |
| `DB_MIGRATOR_PASSWORD` | `infra/local/.env` | password of `sms_migrator`; Compose passes it to `migrate` as `DB_PASSWORD` (with `DB_USER=sms_migrator`) and to the `postgres` container as `SMS_MIGRATOR_PASSWORD` | **Required, no default.** Local-only random value |
| `DB_APP_PASSWORD` | `infra/local/.env` | password of `sms_app`; Compose passes it to `api` as `DB_PASSWORD` (with `DB_USER=sms_app`) and to `postgres` as `SMS_APP_PASSWORD` | **Required, no default.** Local-only random value |
| `SMS_MIGRATOR_PASSWORD`, `SMS_APP_PASSWORD` | nowhere (set by Compose inside the `postgres` container) | the role provisioning script `postgres-init/10-sms-roles.sh` | Do not set them yourself |
| `DB_MIGRATOR_PASSWORD`, `DB_APP_PASSWORD` | your shell or `backend/.env` (optional) | native integration tests only (`go test -tags integration`) | Same values as `infra/local/.env`; not needed for this checklist |

**Unchanged:** `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_PORT` and `API_PORT` in `infra/local/.env`.
`DB_USER` / `DB_PASSWORD` are now only the bootstrap superuser of the `postgres` container (default `sms`). The
`migrate` and `api` services no longer use them; Compose sets their role per service.

The commands below assume the default `DB_USER=sms` and `DB_NAME=sms`.

---

## 0. Preparation

**0.1 Check out PR #17 and record the commit.**

```powershell
git fetch origin claude/ecstatic-gates-ji0o5b-phase4-group1
git switch claude/ecstatic-gates-ji0o5b-phase4-group1
git pull --ff-only
git rev-parse --short HEAD
```

Expected: the branch is checked out. Report the commit ID.

**0.2 Check the Compose version, and that the role script has Unix line endings.**

```powershell
docker compose version
(Get-Content infra/local/postgres-init/10-sms-roles.sh -Raw) -match "`r"
```

Expected: the Compose version is v2.7.0 or later, and the line-ending check prints `False`.

**0.3 Add the two new passwords to `infra/local/.env` without displaying them.**

This adds only the variables that are missing, and never changes existing lines. If you have no
`infra/local/.env` yet, first do section 1 of [`README.md`](README.md).

```powershell
$envFile = 'infra/local/.env'
function New-LocalPassword {
    $bytes = New-Object byte[] 24
    [System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
    -join ($bytes | ForEach-Object { $_.ToString('x2') })
}
if ((Get-Content $envFile -Raw) -notmatch "`n$") { Add-Content -Path $envFile -Value '' -Encoding ascii }
foreach ($name in 'DB_MIGRATOR_PASSWORD', 'DB_APP_PASSWORD') {
    if (-not (Select-String -Path $envFile -Pattern "^$name=" -Quiet)) {
        Add-Content -Path $envFile -Value "$name=$(New-LocalPassword)" -Encoding ascii
    }
}
Get-Content $envFile | Where-Object { $_ -match '^[A-Z_]+=' } | ForEach-Object { ($_ -split '=', 2)[0] }
```

Expected: the last command prints only **names**. Each of `DB_NAME`, `DB_USER`, `DB_PASSWORD`,
`DB_MIGRATOR_PASSWORD`, `DB_APP_PASSWORD`, `DB_PORT` and `API_PORT` appears exactly once.

**0.4 Define the helpers used below** (for this PowerShell window only).

```powershell
$Compose = @('-f', 'infra/local/compose.yaml')                        # your existing stack (project sms-local)
$Check   = @('-p', 'sms-winval', '-f', 'infra/local/compose.yaml')    # throwaway project for Part A

# SQL as the bootstrap superuser, through the container's Unix socket. The SQL is sent on stdin.
function Invoke-AdminSql([string[]]$Project, [string]$Sql) {
    $Sql | docker compose @Project exec -T postgres sh -c 'psql -U $POSTGRES_USER -d $POSTGRES_DB -qAt -v ON_ERROR_STOP=1'
}
# SQL as the application role sms_app, logging in with its password over TCP inside the container.
function Invoke-AppSql([string[]]$Project, [string]$Sql) {
    $Sql | docker compose @Project exec -T postgres sh -c 'PGPASSWORD=$SMS_APP_PASSWORD psql -h 127.0.0.1 -U sms_app -d $POSTGRES_DB -qAt -v ON_ERROR_STOP=1'
}
# The role provisioning command documented in README.md section 9 (idempotent).
function Invoke-Provisioning([string[]]$Project) {
    docker compose @Project exec -T postgres bash /docker-entrypoint-initdb.d/10-sms-roles.sh
}
```

**0.5 Check that Compose accepts the configuration, and that each service has its own role.**

```powershell
docker compose @Compose config --quiet; "exit code: $LASTEXITCODE"
$model = docker compose @Compose config --format json | ConvertFrom-Json
"migrate connects as: " + $model.services.migrate.environment.DB_USER
"api connects as:     " + $model.services.api.environment.DB_USER
"postgres has role passwords: " + [bool]$model.services.postgres.environment.SMS_MIGRATOR_PASSWORD + ' ' + [bool]$model.services.postgres.environment.SMS_APP_PASSWORD
Remove-Variable model
```

Expected:
- `exit code: 0`;
- `migrate connects as: sms_migrator`;
- `api connects as:     sms_app`;
- `postgres has role passwords: True True`.

---

## Part A — fresh database (throwaway project `sms-winval`)

**A1. Use other ports, and confirm that the throwaway volume does not exist yet.**

```powershell
$env:DB_PORT = '55432'; $env:API_PORT = '18080'
docker volume ls --filter name=sms-winval_postgres-data --format '{{.Name}}'
```

Expected: **no output**. If `sms-winval_postgres-data` is listed, an earlier run left it behind: stop and
report it. Do not delete it.

**A2. Start a fresh stack.**

The first time, this builds the image, which takes a few minutes.

```powershell
docker compose @Check up -d --build --wait; "exit code: $LASTEXITCODE"
docker compose @Check ps -a
```

Expected:
- `exit code: 0`;
- `postgres` and `api` are `(healthy)`;
- `migrate` is `Exited (0)`;
- the ports are `127.0.0.1:55432->5432` and `127.0.0.1:18080->8080` only.

**A3. Fresh initialisation created the roles.**

```powershell
docker compose @Check logs postgres | Select-String 'running /docker-entrypoint-initdb.d/10-sms-roles.sh', 'sms-roles: sms_migrator and sms_app provisioned'
```

Expected: both lines appear once.

**A4. Role attributes and memberships.**

```powershell
Invoke-AdminSql $Check "SELECT rolname, rolsuper, rolbypassrls, rolcreaterole, rolcreatedb, rolreplication, rolinherit, rolcanlogin FROM pg_roles WHERE rolname IN ('sms_app', 'sms_migrator') ORDER BY 1"
Invoke-AdminSql $Check "SELECT count(*) FROM pg_auth_members WHERE member IN ('sms_app'::regrole, 'sms_migrator'::regrole) OR roleid IN ('sms_app'::regrole, 'sms_migrator'::regrole)"
```

Expected:

```
sms_app|f|f|f|f|f|f|t
sms_migrator|f|f|f|f|f|f|t
0
```

Neither role is a superuser or has `BYPASSRLS`, `CREATEROLE`, `CREATEDB` or `REPLICATION`. Both can log in,
and neither has any role membership.

**A5. The migration ran as `sms_migrator`, and the API runs as `sms_app`.**

```powershell
docker compose @Check run --rm migrate status
Invoke-AdminSql $Check "SELECT tablename, tableowner FROM pg_tables WHERE schemaname = 'public' ORDER BY 1"
Invoke-AdminSql $Check "SELECT DISTINCT usename FROM pg_stat_activity WHERE application_name = 'sms-api'"
```

Expected:
- `migrate status` lists `20261125041208_baseline.sql` and `20261126175603_create_tenants.sql`, both `applied`;
- the table list:

  ```
  goose_db_version|sms_migrator
  tenants|sms_migrator
  ```
- the API's sessions are `sms_app`.

**A6. Tenant migration verification.**

```powershell
Invoke-AdminSql $Check "SELECT column_name, data_type, column_default, is_nullable FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'tenants' ORDER BY ordinal_position"
Invoke-AdminSql $Check "SELECT conname FROM pg_constraint WHERE conrelid = 'public.tenants'::regclass AND contype <> 'n' ORDER BY 1"
Invoke-AdminSql $Check "SELECT relrowsecurity, relforcerowsecurity, obj_description(oid, 'pg_class') FROM pg_class WHERE oid = 'public.tenants'::regclass"
Invoke-AdminSql $Check "SELECT privilege_type || ' ' || coalesce(column_name, '(table)') FROM (SELECT privilege_type, NULL::text AS column_name FROM information_schema.role_table_grants WHERE grantee = 'sms_app' AND table_name = 'tenants' UNION ALL SELECT privilege_type, column_name FROM information_schema.column_privileges WHERE grantee = 'sms_app' AND table_name = 'tenants' AND privilege_type <> 'SELECT') p ORDER BY 1"
```

Expected:

```
id|uuid|uuidv7()|NO
display_name|text||NO
status|text|'pending'::text|NO
created_at|timestamp with time zone|now()|NO
updated_at|timestamp with time zone|now()|NO
```
```
ck_tenants__display_name
ck_tenants__status
ck_tenants__updated_at
pk_tenants
```
```
f|f|Platform-owned tenant registry (no tenant_id, no RLS). Retention class: account; period undefined, subject to legal review.
```
```
INSERT display_name
SELECT (table)
UPDATE status
UPDATE updated_at
```

**A7. What `sms_app` can and cannot do.** This creates **one synthetic tenant**, and only in the throwaway
database.

```powershell
Invoke-AppSql $Check "INSERT INTO tenants (display_name) VALUES ('Windows validation tenant') RETURNING uuid_extract_version(id), status"
Invoke-AppSql $Check 'SELECT count(*) FROM tenants'
foreach ($sql in 'CREATE TABLE app_made (id int)', 'TRUNCATE tenants', 'DELETE FROM tenants', "INSERT INTO tenants (display_name, status) VALUES ('x', 'active')", 'SELECT count(*) FROM goose_db_version') {
    Invoke-AppSql $Check $sql; "exit code: $LASTEXITCODE"
}
```

Expected:
- `7|pending`: a UUIDv7 ID, generated by PostgreSQL, with the default status;
- `1`;
- for each of the five forbidden statements, an `ERROR:  permission denied for ...` line followed by
  `exit code: 3` (psql's code for a failed statement):
  - `CREATE TABLE` is refused with `permission denied for schema public`, plus a `LINE 1:` pointer;
  - the next three with `permission denied for table tenants`;
  - the last with `permission denied for table goose_db_version`.

  PowerShell may show the error lines in red.

**A8. The API refuses an unsafe role, and provisioning repairs a changed role.**

```powershell
$migratorPassword = ((Get-Content infra/local/.env | Where-Object { $_ -match '^DB_MIGRATOR_PASSWORD=' }) -split '=', 2)[1]
$out = docker compose @Check run --rm --no-deps -e DB_USER=sms_migrator -e "DB_PASSWORD=$migratorPassword" api 2>&1 | Out-String -Width 4096
$code = $LASTEXITCODE
Remove-Variable migratorPassword
"exit code: $code"; $out -match 'as the application role: owns tables'

Invoke-AdminSql $Check 'ALTER ROLE sms_app BYPASSRLS'
docker compose @Check restart api
Start-Sleep -Seconds 5
docker compose @Check ps -a api
docker compose @Check logs --no-log-prefix api | Select-String 'has BYPASSRLS' | Select-Object -Last 1
Invoke-Provisioning $Check
Invoke-AdminSql $Check "SELECT rolbypassrls FROM pg_roles WHERE rolname = 'sms_app'"
docker compose @Check up -d --wait; "exit code: $LASTEXITCODE"
```

Expected:
- `exit code: 1` and `True`: the API refused the table-owner role and printed the reason;
- after `ALTER ROLE`:
  - `api` shows `Exited (1)`;
  - the log line contains `refusing to use database role "sms_app" as the application role: has BYPASSRLS`;
- provisioning prints `sms-roles: sms_migrator and sms_app provisioned`;
- `f`: provisioning removed `BYPASSRLS`;
- `exit code: 0`: the API runs again.

**A9. API health and readiness.**

```powershell
Invoke-RestMethod http://127.0.0.1:18080/healthz
Invoke-RestMethod http://127.0.0.1:18080/readyz
```

Expected: `status: ok`, then `status: ready`.

**A10. Restart and persistence** (the volume is kept).

```powershell
docker compose @Check stop
docker inspect -f '{{.State.ExitCode}}' sms-winval-api-1
docker compose @Check up -d --wait; "exit code: $LASTEXITCODE"
docker compose @Check logs --no-log-prefix migrate | Select-String 'no pending migrations' | Select-Object -Last 1
Invoke-AppSql $Check 'SELECT count(*) FROM tenants'
docker compose @Check down
docker volume ls --filter name=sms-winval_postgres-data --format '{{.Name}}'
docker compose @Check up -d --wait; "exit code: $LASTEXITCODE"
Invoke-AppSql $Check "SELECT display_name, status FROM tenants"
Invoke-Provisioning $Check
```

Expected:
- `0`: the API stopped gracefully;
- `exit code: 0`;
- `no pending migrations`;
- `1`;
- after `down`, the volume `sms-winval_postgres-data` is still listed;
- `exit code: 0`;
- `Windows validation tenant|pending`: the data survived both `stop` and `down`;
- provisioning prints the `sms-roles` line again: it is idempotent.

**A11. Stop the throwaway project, and restore the default ports for Part B.**

```powershell
docker compose @Check down
Remove-Item Env:DB_PORT, Env:API_PORT
```

The throwaway volume `sms-winval_postgres-data` is **left in place** on purpose. It contains only the synthetic
tenant from A7. Removing it is optional and not part of this checklist.

---

## Part B — upgrade your existing database volume (data kept)

**B1. Confirm that your existing volume is there.**

```powershell
docker volume ls --filter name=sms-local_postgres-data --format '{{.Name}}'
```

Expected: `sms-local_postgres-data`. If nothing is listed, you have no existing volume: report "Part B not
applicable" and stop here.

**B2. Stop your stack, then start only PostgreSQL with the new configuration.**

Both commands keep the volume. The second recreates the `postgres` container with the new settings.

```powershell
docker compose @Compose stop
docker compose @Compose up -d --wait postgres; "exit code: $LASTEXITCODE"
docker compose @Compose logs postgres | Select-String 'Skipping initialization' | Select-Object -Last 1
```

Expected:
- `exit code: 0`;
- `Skipping initialization`: the image did **not** re-initialise your existing data, so the roles are not
  created automatically.

**B3. Record the state before the upgrade**, in your home folder, outside the repository.

```powershell
Invoke-AdminSql $Compose "SELECT count(*) FROM pg_roles WHERE rolname IN ('sms_app', 'sms_migrator')"
Invoke-AdminSql $Compose 'SELECT id, version_id, is_applied, tstamp FROM goose_db_version ORDER BY id' | Set-Content "$HOME\sms-upgrade-before-goose.txt"
Invoke-AdminSql $Compose "SELECT schemaname || '.' || tablename || ' ' || tableowner FROM pg_tables WHERE schemaname NOT IN ('pg_catalog', 'information_schema') ORDER BY 1" | Set-Content "$HOME\sms-upgrade-before-tables.txt"
Get-Content "$HOME\sms-upgrade-before-goose.txt", "$HOME\sms-upgrade-before-tables.txt"
```

Expected:
- `0`: no application roles yet;
- the goose rows (normally version `0` and `20261125041208`);
- the table list (normally only `public.goose_db_version sms`).

Report the outputs, and anything other than the normal results.

**B4. Back up the database** (recommended; non-destructive).

```powershell
docker compose @Compose exec -T postgres sh -c 'pg_dump -U $POSTGRES_USER -d $POSTGRES_DB -Fc -f /tmp/sms-before-upgrade.dump'
docker compose @Compose cp postgres:/tmp/sms-before-upgrade.dump "$HOME\sms-before-upgrade.dump"
Get-Item "$HOME\sms-before-upgrade.dump" | Select-Object Name, Length
```

Expected: the file exists with a non-zero `Length`. Keep it until you are satisfied with the upgrade.

**B5. Run the documented upgrade command** ([README.md section 9](README.md#9-database-roles-and-upgrading-an-existing-database-volume)).

```powershell
Invoke-Provisioning $Compose; "exit code: $LASTEXITCODE"
```

Expected:
- `sms-roles: sms_migrator and sms_app provisioned`;
- `exit code: 0`.

If it fails with `objects in schema public are not owned by sms_migrator: ...`, **nothing was changed**. Stop
and report the message: do not drop anything.

**B6. Start the full stack.** `migrate` applies the tenants migration as `sms_migrator`.

```powershell
docker compose @Compose up -d --build --wait; "exit code: $LASTEXITCODE"
docker compose @Compose ps -a
docker compose @Compose run --rm migrate status
```

Expected:
- `exit code: 0`;
- `postgres` and `api` are healthy;
- `migrate` is `Exited (0)`;
- both migrations are `applied`.

**B7. The existing data was kept.**

```powershell
$after = Invoke-AdminSql $Compose 'SELECT id, version_id, is_applied, tstamp FROM goose_db_version ORDER BY id'
Compare-Object (Get-Content "$HOME\sms-upgrade-before-goose.txt") $after
Invoke-AdminSql $Compose "SELECT schemaname || '.' || tablename || ' ' || tableowner FROM pg_tables WHERE schemaname NOT IN ('pg_catalog', 'information_schema') ORDER BY 1"
```

Expected:
- `Compare-Object` lists **only one** line, marked `=>`: the new row for version `20261126175603`. No line is
  marked `<=`, so every earlier row is unchanged.
- The tables:

  ```
  public.goose_db_version sms_migrator
  public.tenants sms_migrator
  ```

  `goose_db_version` was handed over to `sms_migrator`, with its rows kept.

**B8. Role verification on the upgraded database.**

```powershell
Invoke-AdminSql $Compose "SELECT rolname, rolsuper, rolbypassrls, rolcreaterole, rolcreatedb, rolreplication, rolinherit, rolcanlogin FROM pg_roles WHERE rolname IN ('sms_app', 'sms_migrator') ORDER BY 1"
Invoke-AdminSql $Compose "SELECT DISTINCT usename FROM pg_stat_activity WHERE application_name = 'sms-api'"
foreach ($sql in 'CREATE TABLE app_made (id int)', 'TRUNCATE tenants', 'DELETE FROM tenants') {
    Invoke-AppSql $Compose $sql; "exit code: $LASTEXITCODE"
}
```

Expected:
- the same two role lines as in A4;
- `sms_app`;
- three times, an `ERROR:  permission denied for ...` line followed by `exit code: 3`.

These statements are refused, so they change nothing.

**B9. Tenant migration verification**, without leaving any data behind: the insert is rolled back.

```powershell
Invoke-AdminSql $Compose "SELECT column_name, data_type, column_default FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'tenants' ORDER BY ordinal_position"
Invoke-AppSql $Compose "BEGIN; INSERT INTO tenants (display_name) VALUES ('Windows validation tenant') RETURNING uuid_extract_version(id), status; ROLLBACK;"
Invoke-AppSql $Compose 'SELECT count(*) FROM tenants'
```

Expected:
- the same five columns as in A6;
- `7|pending`;
- `0` (or the number of tenants you had before, if any): the test insert was rolled back.

**B10. API health and readiness.**

```powershell
Invoke-RestMethod http://127.0.0.1:8080/healthz
Invoke-RestMethod http://127.0.0.1:8080/readyz
```

Expected: `status: ok`, then `status: ready`.

**B11. Running the upgrade again is safe, and the data persists across a restart.**

```powershell
Invoke-Provisioning $Compose; "exit code: $LASTEXITCODE"
docker compose @Compose stop
docker compose @Compose up -d --wait; "exit code: $LASTEXITCODE"
docker compose @Compose logs --no-log-prefix migrate | Select-String 'no pending migrations' | Select-Object -Last 1
Compare-Object $after (Invoke-AdminSql $Compose 'SELECT id, version_id, is_applied, tstamp FROM goose_db_version ORDER BY id')
Invoke-RestMethod http://127.0.0.1:8080/readyz
```

Expected:
- the `sms-roles` line and `exit code: 0`;
- `exit code: 0`;
- `no pending migrations`;
- `Compare-Object` prints **nothing**: unchanged across the restart;
- `status: ready`.

**B12. Optional: the client portal.** From `frontend/`, run `pnpm dev`, then open
http://localhost:5173/readyz. Expected: `{"status":"ready"}`.

---

## Results to report

Copy this table into your reply. Fill in *pass* or *fail* for each step, and for any failure paste the command
output. **Never paste a password.**

| Step | Result | Notes |
| ---- | ------ | ----- |
| 0.1 commit ID | | |
| 0.2 Compose version / line endings | | |
| 0.3 new variables present (names only) | | |
| 0.5 configuration and per-service roles | | |
| A1 throwaway volume absent | | |
| A2 fresh start | | |
| A3 roles created at initialisation | | |
| A4 role attributes and memberships | | |
| A5 migration as `sms_migrator`, API as `sms_app` | | |
| A6 tenants table verification | | |
| A7 `sms_app` allowed and denied operations | | |
| A8 unsafe role refused; provisioning repairs | | |
| A9 health and readiness | | |
| A10 restart and persistence | | |
| A11 throwaway project stopped | | |
| B1 existing volume present | | |
| B2 PostgreSQL recreated, initialisation skipped | | |
| B3 state before upgrade (outputs) | | |
| B4 backup file created | | |
| B5 upgrade command | | |
| B6 full stack started; migrations applied | | |
| B7 existing data kept | | |
| B8 role verification | | |
| B9 tenants verification (rolled back) | | |
| B10 health and readiness | | |
| B11 idempotent upgrade; restart and persistence | | |
| B12 client portal (optional) | | |

Also report:
- the Docker Desktop, Engine and Compose versions (`docker version`, `docker compose version`);
- whether you used Windows PowerShell 5.1 or PowerShell 7 (`$PSVersionTable.PSVersion`).
