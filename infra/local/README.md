# infra/local/

Local development stack, defined in [`compose.yaml`](compose.yaml). In Phase 3 Group 2 it contains
**PostgreSQL 18 only** (`postgres:18.7-trixie`). It is for local development and is never used in
production.

| Property | Value |
| -------- | ----- |
| Service | `postgres` (Compose project `sms-local`, container `sms-local-postgres-1`) |
| Published port | `127.0.0.1:${DB_PORT}` (default `127.0.0.1:5432`). **Loopback only**: the database is not reachable from your network. |
| Data | Named volume `sms-local_postgres-data`. Survives `stop`, `down`, and restarts. |
| Health check | `pg_isready` over TCP every 2 s. The container reports `healthy` once PostgreSQL accepts connections. |
| Credentials | From `infra/local/.env` (git-ignored). There is **no default password**. |

Requirements: Docker Desktop (Windows, macOS) or Docker Engine with the Compose plugin (Linux). Run every
command below from the **repository root**.

## 1. One-time setup

Create your local settings file and choose a local-only password:

```powershell
Copy-Item infra/local/.env.example infra/local/.env
notepad infra/local/.env      # change DB_PASSWORD; change DB_PORT if 5432 is already in use
```

```bash
cp infra/local/.env.example infra/local/.env
```

`infra/local/.env` is ignored by Git. Never commit it, and never put a real (shared, staging, or
production) password in it.

## 2. Start PostgreSQL

```powershell
docker compose -f infra/local/compose.yaml up -d --wait
```

`--wait` returns once the health check passes (usually within 10–15 seconds; the first start also
downloads the image).

## 3. Check that PostgreSQL is healthy

```powershell
docker compose -f infra/local/compose.yaml ps
```

`STATUS` should show `(healthy)` and `PORTS` should show `127.0.0.1:5432->5432/tcp`. To run the health
check yourself:

```powershell
docker compose -f infra/local/compose.yaml exec postgres pg_isready -h 127.0.0.1 -U sms -d sms
```

Expected: `127.0.0.1:5432 - accepting connections`. (Replace `sms` if you changed `DB_USER` or `DB_NAME`.)

## 4. Inspect the database

Open `psql` inside the container (no local PostgreSQL client needed):

```powershell
docker compose -f infra/local/compose.yaml exec postgres psql -U sms -d sms
```

Useful commands inside `psql`: `\dt` (tables), `SELECT * FROM goose_db_version;` (applied migrations),
`\q` (quit). To follow the server log:

```powershell
docker compose -f infra/local/compose.yaml logs -f postgres
```

## 5. Stop PostgreSQL (keeps your data)

```powershell
docker compose -f infra/local/compose.yaml stop
```

Start it again with step 2. `docker compose -f infra/local/compose.yaml down` also keeps your data: it
removes the container but **not** the volume.

## 6. Reset the local database

Choose the smallest reset that solves your problem.

**a) Re-run migrations only (keeps the database and volume).** Roll the schema back and apply it again
with the migrate command (see [`backend/README.md`](../../backend/README.md), "Migrations"). This changes
only objects created by migrations.

**b) Delete everything and start from an empty database.**

> **WARNING — this permanently deletes all data in your local database.** The `-v` flag removes the
> named volume `sms-local_postgres-data`. There is no undo. It affects only this local Compose project;
> it never touches another project's volumes or any remote database. Do not run it if you need anything
> stored locally.

First check exactly which volume will be deleted:

```powershell
docker volume ls --filter name=sms-local_postgres-data
```

Then delete it and start fresh:

```powershell
docker compose -f infra/local/compose.yaml down -v
docker compose -f infra/local/compose.yaml up -d --wait
```

After a full reset, apply migrations again.

## Integration tests and this database

The backend integration tests connect to this server but **never modify your `sms` database**. Each test
creates its own database named `sms_test_<random>` and drops it when the test finishes, even if it fails.
If a test process is killed, a leftover database can be removed safely (it only ever contains test data):

```powershell
docker compose -f infra/local/compose.yaml exec postgres psql -U sms -d sms -c "SELECT datname FROM pg_database WHERE datname LIKE 'sms\_test\_%'"
```

```powershell
docker compose -f infra/local/compose.yaml exec postgres psql -U sms -d sms -c "DROP DATABASE sms_test_<name> WITH (FORCE)"
```
