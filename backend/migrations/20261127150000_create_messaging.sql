-- Messaging (Phase 4 Group 2): API keys, approved sender IDs, messages, idempotency records, the
-- durable dispatch queue, submission attempts, status history and delivery receipts.
--
-- Tenant isolation (ADR-0010, AGENTS.md §10.7): every table with a tenant_id column has row-level
-- security ENABLED and FORCED, with the one approved policy (RLS-2) in USING and WITH CHECK. The
-- application role (sms_app) gets only the column privileges the API and the dispatcher need; it
-- never gets DELETE or TRUNCATE. Child tables reference messages through (tenant_id, message_id), so
-- a row can never point at another tenant's message.
--
-- provider_receipts is the one platform-owned table: it holds delivery receipts exactly as a
-- provider adapter reported them, BEFORE they are correlated with a tenant's message (a receipt
-- carries only the provider's message reference). It has no tenant_id and no RLS. Correlation runs
-- inside each tenant's transaction-local context, so it only ever matches that tenant's own
-- submission attempts; the matched copy lives in the tenant-owned message_receipts table.
--
-- Retention classes (AGENTS.md §10.8; periods are NOT defined, subject to legal review):
--   api_keys              account (security credential; secret stored only as a SHA-256 hash)
--   sender_ids            account
--   messages              message metadata
--   message_bodies        message content (personal data; never returned by the API or logged)
--   idempotency_keys      operational (a key is honoured for seven days, ADR-0018 / D-094d)
--   dispatch_jobs         operational
--   submission_attempts   message metadata
--   message_status_history  message metadata (append-only audit trail)
--   provider_receipts     message metadata (raw receipts may contain personal data)
--   message_receipts      message metadata (append-only)
--
-- Internal message states (messaging reliability §3.1, ADR-0021); the customer-visible status is
-- derived in Go (contracts/openapi/README.md §3). ACCEPTED is not stored: a message and its dispatch
-- job are committed together, so a committed message is already QUEUED.
--
-- Rollback: Down drops these tables and all their rows (local development and tests only;
-- production migrations are forward-only, AGENTS.md §10.1).

-- +goose Up

-- API keys. The key itself is shown once at creation; only the SHA-256 hash of its random secret
-- is stored. The key's public part names the tenant, so authentication sets the tenant context
-- before it looks the key up (row-level security then applies to the lookup itself).
CREATE TABLE api_keys (
    id          uuid        NOT NULL DEFAULT uuidv7(),
    tenant_id   uuid        NOT NULL,
    label       text        NOT NULL,
    secret_hash bytea       NOT NULL,
    scopes      text[]      NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    revoked_at  timestamptz,
    CONSTRAINT pk_api_keys PRIMARY KEY (id),
    CONSTRAINT fk_api_keys__tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT ck_api_keys__label CHECK (
        label = btrim(label) AND char_length(label) BETWEEN 1 AND 100 AND label !~ '[[:cntrl:]]'
    ),
    CONSTRAINT ck_api_keys__secret_hash CHECK (octet_length(secret_hash) = 32),
    CONSTRAINT ck_api_keys__scopes CHECK (
        cardinality(scopes) >= 1 AND scopes <@ ARRAY['messages:send', 'messages:read']::text[]
    ),
    CONSTRAINT ck_api_keys__revoked_at CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);
CREATE INDEX ix_api_keys__tenant_id ON api_keys (tenant_id);

-- Sender IDs approved for a tenant (contract: 422 sender-id-not-approved otherwise). Natural key.
CREATE TABLE sender_ids (
    tenant_id   uuid        NOT NULL,
    sender_id   text        NOT NULL,
    approved_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT pk_sender_ids PRIMARY KEY (tenant_id, sender_id),
    CONSTRAINT fk_sender_ids__tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT ck_sender_ids__sender_id CHECK (
        char_length(sender_id) BETWEEN 1 AND 32 AND sender_id !~ '[[:cntrl:]]'
    )
);

CREATE TABLE messages (
    id                      uuid        NOT NULL DEFAULT uuidv7(),
    tenant_id               uuid        NOT NULL,
    client_reference        text,
    destination             text        NOT NULL,
    sender_id               text        NOT NULL,
    encoding                text        NOT NULL,
    segment_count           integer     NOT NULL,
    status                  text        NOT NULL DEFAULT 'queued',
    status_reason           text,
    provider_id             text,
    has_conflicting_reports boolean     NOT NULL DEFAULT false,
    accepted_at             timestamptz NOT NULL DEFAULT now(),
    status_updated_at       timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT pk_messages PRIMARY KEY (id),
    CONSTRAINT ux_messages__tenant_id_id UNIQUE (tenant_id, id),
    CONSTRAINT fk_messages__tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT ck_messages__destination CHECK (destination ~ '^\+[1-9][0-9]{6,14}$'),
    CONSTRAINT ck_messages__client_reference CHECK (client_reference ~ '^[A-Za-z0-9._:-]{1,64}$'),
    CONSTRAINT ck_messages__sender_id CHECK (char_length(sender_id) BETWEEN 1 AND 32),
    CONSTRAINT ck_messages__encoding CHECK (encoding IN ('gsm7', 'ucs2')),
    CONSTRAINT ck_messages__segment_count CHECK (segment_count >= 1),
    CONSTRAINT ck_messages__status CHECK (status IN (
        'queued', 'submitting', 'submit_uncertain', 'submitted',
        'rejected', 'failed', 'delivered', 'undelivered', 'unknown'
    )),
    -- Every status has exactly the reasons the contract allows for it (D-099).
    CONSTRAINT ck_messages__status_reason CHECK (
        (status IN ('queued', 'submitting', 'submitted', 'delivered') AND status_reason IS NULL)
        OR (status = 'rejected' AND status_reason IN ('recipient_opted_out', 'policy_violation', 'provider_rejected'))
        OR (status = 'failed' AND status_reason IN ('retries_exhausted', 'expired_before_submission', 'permanent_submission_error'))
        OR (status = 'undelivered' AND status_reason IN ('provider_reported_undelivered', 'provider_reported_expired'))
        OR (status = 'submit_uncertain' AND status_reason = 'submission_outcome_unknown')
        OR (status = 'unknown' AND status_reason = 'no_final_report_received')
    ),
    CONSTRAINT ck_messages__status_updated_at CHECK (status_updated_at >= accepted_at)
);
-- GET /v1/messages: newest acceptance first, keyset pagination.
CREATE INDEX ix_messages__tenant_id_accepted_at_id ON messages (tenant_id, accepted_at DESC, id DESC);
CREATE INDEX ix_messages__tenant_id_client_reference ON messages (tenant_id, client_reference)
    WHERE client_reference IS NOT NULL;
-- The reporting-window sweep looks for submitted messages without a final report.
CREATE INDEX ix_messages__tenant_id_status_updated_at ON messages (tenant_id, status_updated_at)
    WHERE status = 'submitted';

-- Message text, kept apart from the metadata so that it is read only by the dispatcher and can be
-- expired on its own retention schedule.
CREATE TABLE message_bodies (
    tenant_id  uuid NOT NULL,
    message_id uuid NOT NULL,
    body       text NOT NULL,
    CONSTRAINT pk_message_bodies PRIMARY KEY (tenant_id, message_id),
    CONSTRAINT fk_message_bodies__message FOREIGN KEY (tenant_id, message_id) REFERENCES messages (tenant_id, id),
    CONSTRAINT ck_message_bodies__body CHECK (char_length(body) BETWEEN 1 AND 1600)
);

-- Idempotency records (ADR-0018): scope tenant + operation + key; the fingerprint is the SHA-256 of
-- the normalized request body. A record is honoured until expires_at (seven days); after that the
-- key is treated as new and the row is replaced.
CREATE TABLE idempotency_keys (
    tenant_id           uuid        NOT NULL,
    operation           text        NOT NULL,
    idempotency_key     text        NOT NULL,
    request_fingerprint bytea       NOT NULL,
    message_id          uuid        NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    expires_at          timestamptz NOT NULL,
    CONSTRAINT pk_idempotency_keys PRIMARY KEY (tenant_id, operation, idempotency_key),
    CONSTRAINT fk_idempotency_keys__message FOREIGN KEY (tenant_id, message_id) REFERENCES messages (tenant_id, id),
    CONSTRAINT ck_idempotency_keys__operation CHECK (operation IN ('create_message')),
    CONSTRAINT ck_idempotency_keys__idempotency_key CHECK (char_length(idempotency_key) BETWEEN 1 AND 255),
    CONSTRAINT ck_idempotency_keys__request_fingerprint CHECK (octet_length(request_fingerprint) = 32),
    CONSTRAINT ck_idempotency_keys__expires_at CHECK (expires_at > created_at)
);
CREATE INDEX ix_idempotency_keys__message ON idempotency_keys (tenant_id, message_id);

-- Durable dispatch queue (transactional outbox): one job per message, committed in the same
-- transaction as the message. Workers claim jobs with FOR UPDATE SKIP LOCKED and a lease.
CREATE TABLE dispatch_jobs (
    tenant_id        uuid        NOT NULL,
    message_id       uuid        NOT NULL,
    available_at     timestamptz NOT NULL DEFAULT now(),
    lease_expires_at timestamptz,
    completed_at     timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT pk_dispatch_jobs PRIMARY KEY (tenant_id, message_id),
    CONSTRAINT fk_dispatch_jobs__message FOREIGN KEY (tenant_id, message_id) REFERENCES messages (tenant_id, id)
);
CREATE INDEX ix_dispatch_jobs__tenant_id_available_at ON dispatch_jobs (tenant_id, available_at)
    WHERE completed_at IS NULL;

-- One row per hand-off attempt, recorded BEFORE the provider is called (messaging reliability §4).
-- At most one attempt per message can be in progress.
CREATE TABLE submission_attempts (
    id                   uuid        NOT NULL DEFAULT uuidv7(),
    tenant_id            uuid        NOT NULL,
    message_id           uuid        NOT NULL,
    attempt_number       integer     NOT NULL,
    provider_id          text        NOT NULL,
    state                text        NOT NULL DEFAULT 'in_progress',
    provider_message_ref text,
    provider_result_code text,
    started_at           timestamptz NOT NULL DEFAULT now(),
    finished_at          timestamptz,
    CONSTRAINT pk_submission_attempts PRIMARY KEY (id),
    CONSTRAINT ux_submission_attempts__tenant_id_id UNIQUE (tenant_id, id),
    CONSTRAINT ux_submission_attempts__message_attempt UNIQUE (tenant_id, message_id, attempt_number),
    CONSTRAINT fk_submission_attempts__message FOREIGN KEY (tenant_id, message_id) REFERENCES messages (tenant_id, id),
    CONSTRAINT ck_submission_attempts__attempt_number CHECK (attempt_number >= 1),
    CONSTRAINT ck_submission_attempts__state CHECK (state IN (
        'in_progress', 'accepted', 'rejected', 'not_sent_retryable', 'not_sent_permanent', 'uncertain'
    )),
    CONSTRAINT ck_submission_attempts__finished CHECK ((state = 'in_progress') = (finished_at IS NULL)),
    CONSTRAINT ck_submission_attempts__provider_message_ref CHECK (
        (state = 'accepted') = (provider_message_ref IS NOT NULL)
    )
);
CREATE UNIQUE INDEX ux_submission_attempts__in_progress ON submission_attempts (tenant_id, message_id)
    WHERE state = 'in_progress';
CREATE INDEX ix_submission_attempts__provider_message_ref
    ON submission_attempts (tenant_id, provider_id, provider_message_ref)
    WHERE provider_message_ref IS NOT NULL;
CREATE INDEX ix_submission_attempts__in_progress_started_at ON submission_attempts (tenant_id, started_at)
    WHERE state = 'in_progress';

-- Platform-owned receipt inbox (see the header): every receipt as the adapter reported it.
CREATE TABLE provider_receipts (
    id                   uuid        NOT NULL DEFAULT uuidv7(),
    provider_id          text        NOT NULL,
    provider_message_ref text        NOT NULL,
    identity_key         text        NOT NULL,
    receipt_status       text        NOT NULL,
    provider_status_code text,
    provider_reported_at timestamptz,
    raw                  bytea,
    received_at          timestamptz NOT NULL DEFAULT now(),
    matched_at           timestamptz,
    CONSTRAINT pk_provider_receipts PRIMARY KEY (id),
    CONSTRAINT ck_provider_receipts__receipt_status CHECK (receipt_status IN ('delivered', 'undelivered', 'expired')),
    CONSTRAINT ck_provider_receipts__matched_at CHECK (matched_at IS NULL OR matched_at >= received_at)
);
CREATE INDEX ix_provider_receipts__unmatched ON provider_receipts (provider_id, provider_message_ref)
    WHERE matched_at IS NULL;

-- Append-only status history (messaging reliability §5.2).
CREATE TABLE message_status_history (
    id            uuid        NOT NULL DEFAULT uuidv7(),
    tenant_id     uuid        NOT NULL,
    message_id    uuid        NOT NULL,
    from_status   text,
    to_status     text        NOT NULL,
    status_reason text,
    source        text        NOT NULL,
    attempt_id    uuid,
    receipt_id    uuid,
    occurred_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT pk_message_status_history PRIMARY KEY (id),
    CONSTRAINT fk_message_status_history__message FOREIGN KEY (tenant_id, message_id) REFERENCES messages (tenant_id, id),
    CONSTRAINT fk_message_status_history__attempt FOREIGN KEY (tenant_id, attempt_id) REFERENCES submission_attempts (tenant_id, id),
    CONSTRAINT fk_message_status_history__receipt FOREIGN KEY (receipt_id) REFERENCES provider_receipts (id),
    CONSTRAINT ck_message_status_history__source CHECK (source IN ('api', 'dispatcher', 'recovery', 'receipt', 'sweeper'))
);
CREATE INDEX ix_message_status_history__message ON message_status_history (tenant_id, message_id, occurred_at);

-- Receipts correlated with a tenant's message, with the effect each one had (append-only).
CREATE TABLE message_receipts (
    id                  uuid        NOT NULL DEFAULT uuidv7(),
    tenant_id           uuid        NOT NULL,
    message_id          uuid        NOT NULL,
    provider_receipt_id uuid        NOT NULL,
    identity_key        text        NOT NULL,
    receipt_status      text        NOT NULL,
    effect              text        NOT NULL,
    processed_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT pk_message_receipts PRIMARY KEY (id),
    CONSTRAINT ux_message_receipts__provider_receipt_id UNIQUE (provider_receipt_id),
    CONSTRAINT fk_message_receipts__message FOREIGN KEY (tenant_id, message_id) REFERENCES messages (tenant_id, id),
    CONSTRAINT fk_message_receipts__provider_receipt FOREIGN KEY (provider_receipt_id) REFERENCES provider_receipts (id),
    CONSTRAINT ck_message_receipts__receipt_status CHECK (receipt_status IN ('delivered', 'undelivered', 'expired')),
    -- applied: changed the status; duplicate: same receipt already processed; superseded: recorded
    -- without effect by precedence; conflict: a different final report, kept and flagged (D-098).
    CONSTRAINT ck_message_receipts__effect CHECK (effect IN ('applied', 'duplicate', 'superseded', 'conflict'))
);
CREATE INDEX ix_message_receipts__message_identity ON message_receipts (tenant_id, message_id, identity_key);

-- Row-level security: enabled, forced, and the approved tenant policy (RLS-2) on every tenant table.
ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE api_keys FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON api_keys
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE sender_ids ENABLE ROW LEVEL SECURITY;
ALTER TABLE sender_ids FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON sender_ids
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON messages
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE message_bodies ENABLE ROW LEVEL SECURITY;
ALTER TABLE message_bodies FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON message_bodies
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE idempotency_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE idempotency_keys FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON idempotency_keys
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE dispatch_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE dispatch_jobs FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON dispatch_jobs
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE submission_attempts ENABLE ROW LEVEL SECURITY;
ALTER TABLE submission_attempts FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON submission_attempts
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE message_status_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE message_status_history FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON message_status_history
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE message_receipts ENABLE ROW LEVEL SECURITY;
ALTER TABLE message_receipts FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON message_receipts
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- Privileges of the application role: only what the API and the dispatcher use. No DELETE, no
-- TRUNCATE; history and receipt tables are insert-only.
GRANT SELECT ON api_keys TO sms_app;
GRANT INSERT (tenant_id, label, secret_hash, scopes) ON api_keys TO sms_app;
GRANT UPDATE (revoked_at) ON api_keys TO sms_app;

GRANT SELECT, INSERT ON sender_ids TO sms_app;

GRANT SELECT ON messages TO sms_app;
GRANT INSERT (tenant_id, client_reference, destination, sender_id, encoding, segment_count) ON messages TO sms_app;
GRANT UPDATE (status, status_reason, status_updated_at, provider_id, has_conflicting_reports, updated_at) ON messages TO sms_app;

GRANT SELECT, INSERT ON message_bodies TO sms_app;

GRANT SELECT ON idempotency_keys TO sms_app;
GRANT INSERT (tenant_id, operation, idempotency_key, request_fingerprint, message_id, expires_at) ON idempotency_keys TO sms_app;
GRANT UPDATE (request_fingerprint, message_id, created_at, expires_at) ON idempotency_keys TO sms_app;

GRANT SELECT ON dispatch_jobs TO sms_app;
GRANT INSERT (tenant_id, message_id) ON dispatch_jobs TO sms_app;
GRANT UPDATE (available_at, lease_expires_at, completed_at, updated_at) ON dispatch_jobs TO sms_app;

GRANT SELECT ON submission_attempts TO sms_app;
GRANT INSERT (tenant_id, message_id, attempt_number, provider_id) ON submission_attempts TO sms_app;
GRANT UPDATE (state, provider_message_ref, provider_result_code, finished_at) ON submission_attempts TO sms_app;

GRANT SELECT ON provider_receipts TO sms_app;
GRANT INSERT (provider_id, provider_message_ref, identity_key, receipt_status, provider_status_code, provider_reported_at, raw)
    ON provider_receipts TO sms_app;
GRANT UPDATE (matched_at) ON provider_receipts TO sms_app;

GRANT SELECT, INSERT ON message_status_history TO sms_app;

GRANT SELECT, INSERT ON message_receipts TO sms_app;

-- +goose Down
DROP TABLE message_receipts;
DROP TABLE message_status_history;
DROP TABLE provider_receipts;
DROP TABLE submission_attempts;
DROP TABLE dispatch_jobs;
DROP TABLE idempotency_keys;
DROP TABLE message_bodies;
DROP TABLE messages;
DROP TABLE sender_ids;
DROP TABLE api_keys;
