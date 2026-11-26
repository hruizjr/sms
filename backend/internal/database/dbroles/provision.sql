-- Database role provisioning (Phase 4 Group 1, owner decision RLS-4; decision log §18).
--
-- Creates and enforces the two application roles and the privileges of the CURRENT database:
--
--   sms_migrator  runs migrations and owns the application tables. Not a superuser; no BYPASSRLS,
--                 CREATEROLE, CREATEDB or REPLICATION. Database privilege: CONNECT. Schema public:
--                 USAGE and CREATE (needed to create tables). It does not own the database or the
--                 schema.
--   sms_app       used by the API. Not a superuser; no BYPASSRLS, CREATEROLE, CREATEDB or
--                 REPLICATION. Owns nothing. Database privilege: CONNECT. Schema public: USAGE only
--                 (it cannot create objects). Table privileges are granted table by table, in the
--                 migration that creates each table, and never include TRUNCATE.
--
-- Neither role is a member of any role, and no role is a member of them, so neither can inherit
-- or SET ROLE to the other's privileges.
--
-- How to run it: as a SUPERUSER (the bootstrap role of a local or test server), connected to the
-- application database. It is idempotent: running it again changes nothing that is already
-- correct, and it never drops a role, a table or any data. Passwords are not set here; the caller
-- sets them separately (infra/local/postgres-init/10-sms-roles.sh), so this file contains no secret.
--
--   - Fresh local volume: run automatically by the official PostgreSQL image initialisation
--     (/docker-entrypoint-initdb.d), through infra/local/postgres-init/10-sms-roles.sh.
--   - Existing local volume: run by the documented upgrade procedure (infra/local/README.md).
--   - Tests: run by dbtest.NewDatabase in every integration-test database.
--
-- Production provisioning is NOT covered by this group; it needs its own approved procedure.
--
-- It fails (and changes nothing, because the whole file is one statement) when:
--   - it is not run by a superuser;
--   - either role is a member of another role, or has members;
--   - either role owns the current database or any schema in it;
--   - sms_app owns any object in the database;
--   - an object in schema public is owned by a role other than sms_migrator, except goose's own
--     version table goose_db_version, which is transferred to sms_migrator (local volumes created
--     before role separation ran migrations as the bootstrap superuser).

DO $provision$
DECLARE
    role_name text;
    object_list text;
BEGIN
    IF NOT (SELECT rolsuper FROM pg_roles WHERE rolname = current_user) THEN
        RAISE EXCEPTION 'role provisioning must run as a superuser; current user is %', current_user;
    END IF;

    -- 1. The roles and their attributes. ALTER ROLE runs only when an attribute differs, so that
    --    concurrent runs (parallel integration tests) do not update the same catalog row.
    FOREACH role_name IN ARRAY ARRAY['sms_migrator', 'sms_app'] LOOP
        IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = role_name) THEN
            BEGIN
                EXECUTE format('CREATE ROLE %I LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT '
                               'NOREPLICATION NOBYPASSRLS', role_name);
            EXCEPTION WHEN duplicate_object OR unique_violation THEN
                NULL; -- created by a concurrent run; its attributes are checked below
            END;
        END IF;
        IF EXISTS (SELECT FROM pg_roles
                   WHERE rolname = role_name
                     AND (rolsuper OR rolcreatedb OR rolcreaterole OR rolinherit OR rolreplication
                          OR rolbypassrls OR NOT rolcanlogin)) THEN
            EXECUTE format('ALTER ROLE %I LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT '
                           'NOREPLICATION NOBYPASSRLS', role_name);
        END IF;
        IF EXISTS (SELECT FROM pg_auth_members m
                   WHERE m.member = to_regrole(role_name) OR m.roleid = to_regrole(role_name)) THEN
            RAISE EXCEPTION 'role % is a member of another role, or has members; remove these '
                            'memberships (REVOKE) before provisioning', role_name;
        END IF;
    END LOOP;

    -- 2. Neither role may own the database or a schema in it.
    IF (SELECT datdba FROM pg_database WHERE datname = current_database())
           IN (to_regrole('sms_migrator'), to_regrole('sms_app')) THEN
        RAISE EXCEPTION 'database % is owned by an application role; it must be owned by the '
                        'bootstrap role', current_database();
    END IF;
    SELECT string_agg(nspname, ', ' ORDER BY nspname) INTO object_list
    FROM pg_namespace
    WHERE nspowner IN (to_regrole('sms_migrator'), to_regrole('sms_app'));
    IF object_list IS NOT NULL THEN
        RAISE EXCEPTION 'schemas owned by an application role: %', object_list;
    END IF;

    -- 3. Database privileges: CONNECT for the two roles; nothing for PUBLIC (no CONNECT, no
    --    TEMPORARY, no CREATE). The superuser is not affected.
    EXECUTE format('REVOKE ALL ON DATABASE %I FROM PUBLIC, sms_migrator, sms_app', current_database());
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO sms_migrator, sms_app', current_database());

    -- 4. Schema public: only sms_migrator may create objects; sms_app may only use them.
    REVOKE ALL ON SCHEMA public FROM PUBLIC, sms_migrator, sms_app;
    GRANT USAGE, CREATE ON SCHEMA public TO sms_migrator;
    GRANT USAGE ON SCHEMA public TO sms_app;

    -- 5. Functions that sms_migrator creates later are not executable by PUBLIC by default;
    --    migrations grant EXECUTE explicitly where needed.
    ALTER DEFAULT PRIVILEGES FOR ROLE sms_migrator REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;

    -- 6. Existing volumes: goose's version table, created by the bootstrap superuser before role
    --    separation, is transferred to sms_migrator (with its identity sequence). Its rows are kept.
    IF to_regclass('public.goose_db_version') IS NOT NULL
       AND (SELECT relowner FROM pg_class WHERE oid = to_regclass('public.goose_db_version'))
           <> to_regrole('sms_migrator') THEN
        ALTER TABLE public.goose_db_version OWNER TO sms_migrator;
    END IF;

    -- 7. Every other object in schema public must already belong to sms_migrator. Unknown objects
    --    are reported, never silently taken over or dropped.
    SELECT string_agg(format('%s (owner %s)', c.relname, pg_get_userbyid(c.relowner)), ', ' ORDER BY c.relname)
    INTO object_list
    FROM pg_class c
    WHERE c.relnamespace = 'public'::regnamespace
      AND c.relkind IN ('r', 'p', 'v', 'm', 'S', 'f')
      AND c.relowner <> to_regrole('sms_migrator');
    IF object_list IS NOT NULL THEN
        RAISE EXCEPTION 'objects in schema public are not owned by sms_migrator: %. Transfer them '
                        '(ALTER ... OWNER TO sms_migrator) or remove them, then provision again', object_list;
    END IF;

    -- 8. sms_app must own nothing in this database.
    SELECT string_agg(c.relname, ', ' ORDER BY c.relname) INTO object_list
    FROM pg_class c
    WHERE c.relowner = to_regrole('sms_app');
    IF object_list IS NOT NULL THEN
        RAISE EXCEPTION 'sms_app owns objects (%); the application role must own nothing', object_list;
    END IF;
END
$provision$;
