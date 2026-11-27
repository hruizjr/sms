#!/usr/bin/env bash
# Provisions the application database roles sms_migrator and sms_app (Phase 4 Group 1, RLS-4;
# decision log §18). Local development only; never used in production.
#
# Two ways it runs, both inside the postgres container:
#
#   1. Fresh volume: the official PostgreSQL image runs every script in /docker-entrypoint-initdb.d
#      once, when it initialises an empty data directory. Nothing else is needed.
#   2. Existing volume (created before role separation): the image never runs init scripts again,
#      so run this script yourself, as documented in infra/local/README.md ("Upgrading an existing
#      database volume"):
#        docker compose -f infra/local/compose.yaml exec postgres bash /docker-entrypoint-initdb.d/10-sms-roles.sh
#
# It is idempotent and never deletes data: it applies /sms/provision-roles.sql (mounted read-only
# from backend/internal/database/dbroles/provision.sql) and then sets both role passwords from
# SMS_MIGRATOR_PASSWORD and SMS_APP_PASSWORD (set by compose.yaml from infra/local/.env).
#
# It connects through the container's Unix socket as the bootstrap superuser (POSTGRES_USER). The
# passwords are read by psql from the environment (\getenv), so they never appear on a command line.
# The same file works when sourced by the image's entrypoint (no `exit`, no changed shell options).

: "${POSTGRES_USER:?POSTGRES_USER is not set}"
: "${POSTGRES_DB:?POSTGRES_DB is not set}"
: "${SMS_MIGRATOR_PASSWORD:?SMS_MIGRATOR_PASSWORD is not set: add DB_MIGRATOR_PASSWORD to infra/local/.env and recreate the postgres container}"
: "${SMS_APP_PASSWORD:?SMS_APP_PASSWORD is not set: add DB_APP_PASSWORD to infra/local/.env and recreate the postgres container}"

psql --no-psqlrc -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<'SQL'
\set QUIET on
\i /sms/provision-roles.sql
\getenv migrator_password SMS_MIGRATOR_PASSWORD
\getenv app_password SMS_APP_PASSWORD
ALTER ROLE sms_migrator PASSWORD :'migrator_password';
ALTER ROLE sms_app PASSWORD :'app_password';
\echo sms-roles: sms_migrator and sms_app provisioned
SQL
