-- Security auditability and operational controls (Phase 4 Group 6): the durable security audit log
-- (owner decision G4-8, security baseline §9, S-8), API key last-used tracking (G4-6), and the layered
-- sign-in limits of the client portal (G3-9, decision log §23 and §23.1, security baseline §3 and §7).
--
-- Tables:
--   security_audit_events     tenant-owned: security events of one organization (sign-in, sign-out,
--                             account lockout, API key lifecycle, authorization denials, portal user
--                             administration through smsctl). Row-level security ENABLED and FORCED
--                             with the one approved policy (RLS-2).
--   platform_security_events  platform-owned: security events that belong to no identifiable
--                             organization (a sign-in for an address that is not registered, a
--                             rate-limited sign-in). It has no tenant_id column, so no tenant's data is
--                             stored outside row-level security. The application may only insert:
--                             it cannot read these rows, so no portal user (of any organization) can
--                             see them. They are read by database administrators until an approved
--                             internal administration function exists.
--   portal_sign_in_failure_budgets  platform-owned operational counters: failed (or still pending)
--                             sign-in attempts per source and window, shared by every API instance.
--                             A source is an IPv4 address (/32), an IPv6 /64 or an IPv6 /48.
--   portal_sign_in_request_buckets  platform-owned operational state: the all-attempt token bucket
--                             per source (IPv4 /32 or IPv6 /64), shared by every API instance.
--
-- Append-only audit tables (AGENTS.md §10.9): the application role sms_app gets INSERT (and, for the
-- tenant table, SELECT) only: no UPDATE, no DELETE, no TRUNCATE. It cannot supply an event's id or
-- occurred_at: both are assigned by PostgreSQL (column privileges exclude them), so the application
-- can neither choose an existing event's id nor back-date an event. As a second layer, triggers reject
-- UPDATE, DELETE and TRUNCATE by any role, including the table owner sms_migrator. These controls stop
-- the application (and an attacker holding only its credentials) from changing or removing recorded
-- events. They are NOT tamper-proof: a database superuser, or the table owner after disabling the
-- triggers, can still change or remove rows, and nothing stored in this database can prove otherwise.
-- Evidence that holds against a superuser needs an external immutable export or externally anchored
-- integrity evidence (a production requirement, security baseline S-8), not part of this group.
--
-- Deduplication: an event may carry a dedup_key (set by the application from server-side values
-- only: the actor, the action and a time interval or lock). A unique index on (tenant_id, dedup_key)
-- makes a repeated event an INSERT ... ON CONFLICT DO NOTHING that writes nothing: no existing event
-- is ever modified, and no auxiliary table is needed.
--
-- Details: the details column of both audit tables holds only fixed, non-secret fields written by the
-- application (internal/audit.Details): never passwords, session tokens, API keys or their hashes,
-- authorization headers or message text.
--
-- Retention classes (AGENTS.md §10.8; periods are NOT defined, subject to legal review, S-4):
--   security_audit_events           audit (personal data: user IDs, source IP addresses, user agents)
--   platform_security_events        audit (personal data: source IP addresses, user agents)
--   portal_sign_in_failure_budgets  operational (personal data: source IP addresses or prefixes); rows
--                                   are deleted by the API once their window has ended
--   portal_sign_in_request_buckets  operational (personal data: source IP addresses or prefixes); rows
--                                   are deleted by the API once their bucket is full again
--   api_keys.last_used_at           account (security credential metadata)
-- No audit row is deleted automatically: a retention policy is an open owner decision.
--
-- Locks (AGENTS.md §10.1): api_keys is read by every public API request and tenants by every tenant
-- transaction. Adding the nullable column is a catalog-only change and the CHECK constraint scans the
-- (small) api_keys table once, but both need a brief ACCESS EXCLUSIVE lock, and the new foreign key a
-- SHARE ROW EXCLUSIVE lock on tenants. lock_timeout makes the migration fail (and roll back, changing
-- nothing) instead of queueing behind a long transaction while every later query waits behind it.
--
-- Rollback: Down drops the new tables, functions and column with all their rows (local development
-- and tests only; production migrations are forward-only, AGENTS.md §10.1).

-- +goose Up
SET LOCAL lock_timeout = '5s';

-- API key last-used time (G4-6): written at most once a minute per key, after the key authenticated
-- and authorized a public API request; NULL when the key has never been used.
ALTER TABLE api_keys ADD COLUMN last_used_at timestamptz;
ALTER TABLE api_keys ADD CONSTRAINT ck_api_keys__last_used_at CHECK (last_used_at IS NULL OR last_used_at >= created_at);

CREATE TABLE security_audit_events (
    id          uuid        NOT NULL DEFAULT uuidv7(),
    tenant_id   uuid        NOT NULL,
    -- Assigned by PostgreSQL when the row is inserted (the application has no privilege to set it).
    occurred_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    action      text        NOT NULL,
    outcome     text        NOT NULL,
    actor_type  text        NOT NULL,
    actor_id    uuid,
    target_type text,
    target_id   uuid,
    request_id  text,
    source_ip   inet,
    user_agent  text,
    details     jsonb       NOT NULL DEFAULT '{}'::jsonb,
    dedup_key   text,
    CONSTRAINT pk_security_audit_events PRIMARY KEY (id),
    CONSTRAINT fk_security_audit_events__tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT ck_security_audit_events__action CHECK (action IN (
        'portal.sign_in', 'portal.sign_out', 'portal.account_locked',
        'api_key.list', 'api_key.create', 'api_key.revoke',
        'audit_log.list',
        'portal_user.create', 'portal_user.unlock', 'portal_user.sessions_revoke',
        'sender_id.approve'
    )),
    CONSTRAINT ck_security_audit_events__outcome CHECK (outcome IN ('success', 'failure', 'denied', 'enforced')),
    -- "enforced" (a security control was applied: the account was locked) is the outcome of an
    -- account lock, and of nothing else.
    CONSTRAINT ck_security_audit_events__enforced CHECK ((outcome = 'enforced') = (action = 'portal.account_locked')),
    -- A portal user acts with its ID; an unauthenticated caller and the platform itself have none.
    CONSTRAINT ck_security_audit_events__actor CHECK (
        (actor_type = 'portal_user' AND actor_id IS NOT NULL)
        OR (actor_type IN ('anonymous', 'system') AND actor_id IS NULL)
    ),
    CONSTRAINT ck_security_audit_events__target CHECK (
        (target_type IS NULL AND target_id IS NULL)
        OR (target_type IN ('portal_user', 'portal_session', 'api_key') AND target_id IS NOT NULL)
    ),
    CONSTRAINT ck_security_audit_events__request_id CHECK (request_id ~ '^[A-Za-z0-9._-]{1,128}$'),
    CONSTRAINT ck_security_audit_events__user_agent CHECK (
        char_length(user_agent) BETWEEN 1 AND 256 AND user_agent !~ '[[:cntrl:]]'
    ),
    CONSTRAINT ck_security_audit_events__details CHECK (
        jsonb_typeof(details) = 'object' AND octet_length(details::text) <= 2048
    ),
    CONSTRAINT ck_security_audit_events__dedup_key CHECK (dedup_key ~ '^[a-z0-9_.:-]{1,160}$')
);
-- The audit log page lists one organization's events newest first (keyset pagination).
CREATE INDEX ix_security_audit_events__tenant_id_occurred_at
    ON security_audit_events (tenant_id, occurred_at DESC, id DESC);
-- One event per deduplication key and organization (INSERT ... ON CONFLICT DO NOTHING).
CREATE UNIQUE INDEX ux_security_audit_events__tenant_id_dedup_key
    ON security_audit_events (tenant_id, dedup_key) WHERE dedup_key IS NOT NULL;

CREATE TABLE platform_security_events (
    id          uuid        NOT NULL DEFAULT uuidv7(),
    -- Assigned by PostgreSQL when the row is inserted (the application has no privilege to set it).
    occurred_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    action      text        NOT NULL,
    outcome     text        NOT NULL,
    request_id  text,
    source_ip   inet,
    user_agent  text,
    details     jsonb       NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT pk_platform_security_events PRIMARY KEY (id),
    CONSTRAINT ck_platform_security_events__action CHECK (action IN ('portal.sign_in', 'portal.sign_in_rate_limited')),
    CONSTRAINT ck_platform_security_events__outcome CHECK (outcome IN ('success', 'failure', 'denied')),
    CONSTRAINT ck_platform_security_events__request_id CHECK (request_id ~ '^[A-Za-z0-9._-]{1,128}$'),
    CONSTRAINT ck_platform_security_events__user_agent CHECK (
        char_length(user_agent) BETWEEN 1 AND 256 AND user_agent !~ '[[:cntrl:]]'
    ),
    CONSTRAINT ck_platform_security_events__details CHECK (
        jsonb_typeof(details) = 'object' AND octet_length(details::text) <= 2048
    )
);
CREATE INDEX ix_platform_security_events__occurred_at ON platform_security_events (occurred_at);

-- Defense in depth for the append-only audit tables: reject every change of an existing row, by any
-- role, unless the table owner or a superuser first disables these triggers (see the header).
-- +goose StatementBegin
CREATE FUNCTION reject_audit_event_change() RETURNS trigger
    LANGUAGE plpgsql
    SET search_path = pg_catalog
AS $$
BEGIN
    RAISE EXCEPTION 'audit events are append-only: % on % is not allowed', TG_OP, TG_TABLE_NAME
        USING ERRCODE = 'insufficient_privilege';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER security_audit_events_append_only
    BEFORE UPDATE OR DELETE ON security_audit_events
    FOR EACH ROW EXECUTE FUNCTION reject_audit_event_change();
CREATE TRIGGER security_audit_events_no_truncate
    BEFORE TRUNCATE ON security_audit_events
    FOR EACH STATEMENT EXECUTE FUNCTION reject_audit_event_change();
CREATE TRIGGER platform_security_events_append_only
    BEFORE UPDATE OR DELETE ON platform_security_events
    FOR EACH ROW EXECUTE FUNCTION reject_audit_event_change();
CREATE TRIGGER platform_security_events_no_truncate
    BEFORE TRUNCATE ON platform_security_events
    FOR EACH STATEMENT EXECUTE FUNCTION reject_audit_event_change();

-- Failed-attempt budgets (internal/signinlimit). One row per source: an IPv4 address (/32), an IPv6
-- /64 or an IPv6 /48 aggregate (cidr values with different prefix lengths are different keys). Each
-- sign-in attempt reserves a slot in its source's budgets before any credential is checked; a
-- successful sign-in returns its own slot, so failure_count counts failed and still pending attempts
-- of the window. The window starts with the first reservation; rows whose window has ended, and rows
-- with nothing reserved, are deleted. refusal_reported_for is the window start for which a refusal
-- was recorded as a platform event (at most one per source and window).
CREATE TABLE portal_sign_in_failure_budgets (
    source               cidr        NOT NULL,
    window_start_at      timestamptz NOT NULL,
    failure_count        integer     NOT NULL,
    refusal_reported_for timestamptz,
    CONSTRAINT pk_portal_sign_in_failure_budgets PRIMARY KEY (source),
    CONSTRAINT ck_portal_sign_in_failure_budgets__failure_count CHECK (failure_count >= 0)
);
CREATE INDEX ix_portal_sign_in_failure_budgets__window_start_at ON portal_sign_in_failure_budgets (window_start_at);

-- All-attempt token buckets (internal/signinlimit). One row per source (an IPv4 address or an IPv6
-- /64): tokens left at refilled_at; the bucket refills at the configured rate up to the configured
-- burst. last_allowed and last_reported are the result of the latest statement on the row (read back
-- by the same statement with RETURNING). refusal_reported_at is the last time a refusal of this source
-- was recorded as a platform event. Rows that are full again are deleted.
CREATE TABLE portal_sign_in_request_buckets (
    source              cidr             NOT NULL,
    tokens              double precision NOT NULL,
    refilled_at         timestamptz      NOT NULL,
    last_allowed        boolean          NOT NULL,
    last_reported       boolean          NOT NULL DEFAULT false,
    refusal_reported_at timestamptz,
    CONSTRAINT pk_portal_sign_in_request_buckets PRIMARY KEY (source),
    CONSTRAINT ck_portal_sign_in_request_buckets__tokens CHECK (tokens >= 0 AND tokens <= 1000000)
);
CREATE INDEX ix_portal_sign_in_request_buckets__refilled_at ON portal_sign_in_request_buckets (refilled_at);

-- Row-level security on the tenant-owned audit table: enabled, forced, the approved policy (RLS-2).
ALTER TABLE security_audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE security_audit_events FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON security_audit_events
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- Application role privileges (least privilege).
GRANT UPDATE (last_used_at) ON api_keys TO sms_app;

-- Audit: insert and read the organization's own events; no UPDATE, DELETE or TRUNCATE. The ID and
-- the time are assigned by the database: there is no INSERT privilege on id or occurred_at.
GRANT SELECT ON security_audit_events TO sms_app;
GRANT INSERT (tenant_id, action, outcome, actor_type, actor_id, target_type, target_id,
              request_id, source_ip, user_agent, details, dedup_key) ON security_audit_events TO sms_app;
-- Platform events: insert only (no SELECT), so the application cannot read them back.
GRANT INSERT (action, outcome, request_id, source_ip, user_agent, details) ON platform_security_events TO sms_app;

-- Sign-in limits: the atomic upserts, refunds, and deletion of ended windows and full buckets. No TRUNCATE.
GRANT SELECT, DELETE ON portal_sign_in_failure_budgets TO sms_app;
GRANT INSERT (source, window_start_at, failure_count) ON portal_sign_in_failure_budgets TO sms_app;
GRANT UPDATE (window_start_at, failure_count, refusal_reported_for) ON portal_sign_in_failure_budgets TO sms_app;
GRANT SELECT, DELETE ON portal_sign_in_request_buckets TO sms_app;
GRANT INSERT (source, tokens, refilled_at, last_allowed) ON portal_sign_in_request_buckets TO sms_app;
GRANT UPDATE (tokens, refilled_at, last_allowed, last_reported, refusal_reported_at) ON portal_sign_in_request_buckets TO sms_app;

-- +goose Down
SET LOCAL lock_timeout = '5s';
DROP TABLE portal_sign_in_request_buckets;
DROP TABLE portal_sign_in_failure_budgets;
DROP TABLE platform_security_events;
DROP TABLE security_audit_events;
DROP FUNCTION reject_audit_event_change();
ALTER TABLE api_keys DROP CONSTRAINT ck_api_keys__last_used_at;
ALTER TABLE api_keys DROP COLUMN last_used_at;
