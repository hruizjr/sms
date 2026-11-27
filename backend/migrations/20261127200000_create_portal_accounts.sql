-- Client portal accounts (Phase 4 Group 3): portal users, their server-side sessions, and the login
-- directory.
--
-- Tenant isolation (ADR-0010, AGENTS.md §10.7): portal_users and portal_sessions are tenant-owned and
-- have row-level security ENABLED and FORCED with the one approved policy (RLS-2).
--
-- portal_login_directory is platform-owned and has no RLS. It is the one owner-approved exception to the
-- RLS catalog guard (decision log §20, G3-1): signing in with only an email address and a password needs
-- one lookup before any tenant is known ("which tenant does this email belong to?"). The directory holds
-- only that mapping (normalized email -> tenant and user). Password hashes, roles, lockout state and
-- sessions stay in the tenant-owned tables and are read only inside the tenant's transaction-local
-- context. A composite foreign key ties each directory row to its user, so the email, tenant and user
-- can never disagree, and an email address is unique across the platform.
--
-- Retention classes (AGENTS.md §10.8; periods are NOT defined, subject to legal review):
--   portal_users            account (personal data: email address; password stored only as an Argon2id hash)
--   portal_login_directory  account (personal data: email address)
--   portal_sessions         operational (security; the session secret is stored only as a SHA-256 hash)
--
-- Privileges: sms_app gets only the columns the portal and smsctl need; no DELETE, no TRUNCATE.
-- Expired and revoked sessions therefore stay until a retention job exists (not in this group).
--
-- Rollback: Down drops these tables and all their rows (local development and tests only; production
-- migrations are forward-only, AGENTS.md §10.1).

-- +goose Up

CREATE TABLE portal_users (
    id                 uuid        NOT NULL DEFAULT uuidv7(),
    tenant_id          uuid        NOT NULL,
    email              text        NOT NULL,
    role               text        NOT NULL,
    password_hash      text        NOT NULL,
    failed_login_count integer     NOT NULL DEFAULT 0,
    locked_until       timestamptz,
    last_login_at      timestamptz,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT pk_portal_users PRIMARY KEY (id),
    CONSTRAINT ux_portal_users__tenant_id_id UNIQUE (tenant_id, id),
    CONSTRAINT ux_portal_users__tenant_id_id_email UNIQUE (tenant_id, id, email),
    CONSTRAINT ux_portal_users__tenant_id_email UNIQUE (tenant_id, email),
    CONSTRAINT fk_portal_users__tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    -- Stored normalized (trimmed, lower case); the application validates the address in detail.
    CONSTRAINT ck_portal_users__email CHECK (
        email = lower(btrim(email)) AND char_length(email) BETWEEN 3 AND 254 AND email ~ '^[^@[:space:][:cntrl:]]+@[^@[:space:][:cntrl:]]+$'
    ),
    -- MVP client roles (MVP-03): Owner, Sender, Viewer.
    CONSTRAINT ck_portal_users__role CHECK (role IN ('owner', 'sender', 'viewer')),
    CONSTRAINT ck_portal_users__password_hash CHECK (password_hash LIKE '$argon2id$%'),
    CONSTRAINT ck_portal_users__failed_login_count CHECK (failed_login_count >= 0),
    CONSTRAINT ck_portal_users__updated_at CHECK (updated_at >= created_at)
);

-- Platform-owned login directory (see the header): one row per portal user.
CREATE TABLE portal_login_directory (
    email      text        NOT NULL,
    tenant_id  uuid        NOT NULL,
    user_id    uuid        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT pk_portal_login_directory PRIMARY KEY (email),
    CONSTRAINT ux_portal_login_directory__user UNIQUE (tenant_id, user_id),
    CONSTRAINT fk_portal_login_directory__user FOREIGN KEY (tenant_id, user_id, email)
        REFERENCES portal_users (tenant_id, id, email)
);

-- Server-side sessions. The cookie carries the tenant, the session ID and a random secret; only the
-- SHA-256 hash of the secret is stored. Times are written by the application (one clock for checks).
CREATE TABLE portal_sessions (
    id           uuid        NOT NULL DEFAULT uuidv7(),
    tenant_id    uuid        NOT NULL,
    user_id      uuid        NOT NULL,
    secret_hash  bytea       NOT NULL,
    created_at   timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL,
    expires_at   timestamptz NOT NULL,
    revoked_at   timestamptz,
    CONSTRAINT pk_portal_sessions PRIMARY KEY (id),
    CONSTRAINT fk_portal_sessions__user FOREIGN KEY (tenant_id, user_id) REFERENCES portal_users (tenant_id, id),
    CONSTRAINT ck_portal_sessions__secret_hash CHECK (octet_length(secret_hash) = 32),
    CONSTRAINT ck_portal_sessions__last_seen_at CHECK (last_seen_at >= created_at),
    CONSTRAINT ck_portal_sessions__expires_at CHECK (expires_at > created_at),
    CONSTRAINT ck_portal_sessions__revoked_at CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);
CREATE INDEX ix_portal_sessions__user ON portal_sessions (tenant_id, user_id);

-- Row-level security: enabled, forced, and the approved tenant policy (RLS-2) on every tenant table.
ALTER TABLE portal_users ENABLE ROW LEVEL SECURITY;
ALTER TABLE portal_users FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON portal_users
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE portal_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE portal_sessions FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON portal_sessions
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- Application role privileges (least privilege; no DELETE, no TRUNCATE).
GRANT SELECT ON portal_users TO sms_app;
GRANT INSERT (tenant_id, email, role, password_hash) ON portal_users TO sms_app;
GRANT UPDATE (failed_login_count, locked_until, last_login_at, updated_at) ON portal_users TO sms_app;

GRANT SELECT ON portal_login_directory TO sms_app;
GRANT INSERT (email, tenant_id, user_id) ON portal_login_directory TO sms_app;

GRANT SELECT ON portal_sessions TO sms_app;
GRANT INSERT (tenant_id, user_id, secret_hash, created_at, last_seen_at, expires_at) ON portal_sessions TO sms_app;
GRANT UPDATE (last_seen_at, revoked_at) ON portal_sessions TO sms_app;

-- +goose Down
DROP TABLE portal_sessions;
DROP TABLE portal_login_directory;
DROP TABLE portal_users;
