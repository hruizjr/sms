-- Security auditability and operational controls (Phase 4 Group 6): the durable security audit log
-- (owner decision G4-8, security baseline §9, S-8), API key last-used tracking (G4-6), and the per-IP
-- sign-in rate limit of the client portal (G3-9, security baseline §3 and §7).
--
-- Tables:
--   security_audit_events     tenant-owned: security events of one organization (sign-in, sign-out,
--                             account lockout, API key lifecycle, authorization denials). Row-level
--                             security ENABLED and FORCED with the one approved policy (RLS-2).
--   platform_security_events  platform-owned: security events that belong to no identifiable
--                             organization (a sign-in for an address that is not registered, a
--                             rate-limited sign-in). It has no tenant_id column, so no tenant's data is
--                             stored outside row-level security. The application may only insert:
--                             it cannot read these rows, so no portal user (of any organization) can
--                             see them. They are read by database administrators until an approved
--                             internal administration function exists.
--   portal_sign_in_rate_limits  platform-owned operational counters: sign-in attempts per source
--                             address and window, shared by every API instance. Holds an address
--                             (or an IPv6 /64 prefix), a window start and a count; nothing else.
--
-- Append-only audit tables (AGENTS.md §10.9): the application role sms_app gets INSERT (and, for the
-- tenant table, SELECT) only: no UPDATE, no DELETE, no TRUNCATE. As a second layer, triggers reject
-- UPDATE, DELETE and TRUNCATE by any role, including the table owner sms_migrator. This is not a
-- tamper-proof guarantee: a superuser (or the table owner, by disabling the triggers) can still change
-- or remove rows; detecting that needs tamper-evidence such as hash chaining or export to write-once
-- storage, which is not part of this group (security baseline S-8).
--
-- Details: the details column of both audit tables holds only fixed, non-secret fields written by the
-- application (internal/audit.Details): never passwords, session tokens, API keys or their hashes,
-- authorization headers or message text.
--
-- Retention classes (AGENTS.md §10.8; periods are NOT defined, subject to legal review, S-4):
--   security_audit_events       audit (personal data: user IDs, source IP addresses, user agents)
--   platform_security_events    audit (personal data: source IP addresses, user agents)
--   portal_sign_in_rate_limits  operational (personal data: source IP addresses); rows are deleted by
--                               the API once their window has ended (internal/signinlimit)
--   api_keys.last_used_at       account (security credential metadata)
-- No audit row is deleted automatically: a retention policy is an open owner decision.
--
-- Rollback: Down drops the new tables, functions and column with all their rows (local development
-- and tests only; production migrations are forward-only, AGENTS.md §10.1).

-- +goose Up

-- API key last-used time (G4-6): written at most once a minute per key, after the key authenticated
-- and authorized a public API request; NULL when the key has never been used.
ALTER TABLE api_keys ADD COLUMN last_used_at timestamptz;
ALTER TABLE api_keys ADD CONSTRAINT ck_api_keys__last_used_at CHECK (last_used_at IS NULL OR last_used_at >= created_at);

CREATE TABLE security_audit_events (
    id          uuid        NOT NULL DEFAULT uuidv7(),
    tenant_id   uuid        NOT NULL,
    occurred_at timestamptz NOT NULL,
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
    CONSTRAINT pk_security_audit_events PRIMARY KEY (id),
    CONSTRAINT fk_security_audit_events__tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT ck_security_audit_events__action CHECK (action IN (
        'portal.sign_in', 'portal.sign_out', 'portal.account_locked',
        'api_key.list', 'api_key.create', 'api_key.revoke',
        'audit_log.list'
    )),
    CONSTRAINT ck_security_audit_events__outcome CHECK (outcome IN ('success', 'failure', 'denied')),
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
    )
);
-- The audit log page lists one organization's events newest first (keyset pagination).
CREATE INDEX ix_security_audit_events__tenant_id_occurred_at
    ON security_audit_events (tenant_id, occurred_at DESC, id DESC);

CREATE TABLE platform_security_events (
    id          uuid        NOT NULL DEFAULT uuidv7(),
    occurred_at timestamptz NOT NULL,
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

-- Sign-in attempts per source and window (internal/signinlimit). One row per source: an IPv4
-- address (/32) or an IPv6 /64 prefix. The window starts at the first attempt and lasts the
-- configured duration; the count includes refused attempts. Rows whose window has ended are deleted.
CREATE TABLE portal_sign_in_rate_limits (
    source          cidr        NOT NULL,
    window_start_at timestamptz NOT NULL,
    attempt_count   integer     NOT NULL,
    CONSTRAINT pk_portal_sign_in_rate_limits PRIMARY KEY (source),
    CONSTRAINT ck_portal_sign_in_rate_limits__attempt_count CHECK (attempt_count >= 1)
);
CREATE INDEX ix_portal_sign_in_rate_limits__window_start_at ON portal_sign_in_rate_limits (window_start_at);

-- Row-level security on the tenant-owned audit table: enabled, forced, the approved policy (RLS-2).
ALTER TABLE security_audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE security_audit_events FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON security_audit_events
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- Application role privileges (least privilege).
GRANT UPDATE (last_used_at) ON api_keys TO sms_app;

-- Audit: insert and read the organization's own events; no UPDATE, DELETE or TRUNCATE. The ID is
-- generated by the database.
GRANT SELECT ON security_audit_events TO sms_app;
GRANT INSERT (tenant_id, occurred_at, action, outcome, actor_type, actor_id, target_type, target_id,
              request_id, source_ip, user_agent, details) ON security_audit_events TO sms_app;
-- Platform events: insert only (no SELECT), so the application cannot read them back.
GRANT INSERT (occurred_at, action, outcome, request_id, source_ip, user_agent, details)
    ON platform_security_events TO sms_app;

-- Rate-limit counters: the atomic upsert, and deletion of expired windows. No TRUNCATE.
GRANT SELECT, DELETE ON portal_sign_in_rate_limits TO sms_app;
GRANT INSERT (source, window_start_at, attempt_count) ON portal_sign_in_rate_limits TO sms_app;
GRANT UPDATE (window_start_at, attempt_count) ON portal_sign_in_rate_limits TO sms_app;

-- +goose Down
DROP TABLE portal_sign_in_rate_limits;
DROP TABLE platform_security_events;
DROP TABLE security_audit_events;
DROP FUNCTION reject_audit_event_change();
ALTER TABLE api_keys DROP CONSTRAINT ck_api_keys__last_used_at;
ALTER TABLE api_keys DROP COLUMN last_used_at;
