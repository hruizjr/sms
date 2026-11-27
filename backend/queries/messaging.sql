-- Messaging (internal/messaging, Phase 4 Group 2). Every query on tenant data runs inside the
-- tenant's transaction-local context (database.DB.InTenantTx) and also filters by tenant_id
-- explicitly; row-level security is the second line of defence. Times come from the database clock.

-- name: ApproveSenderID :exec
INSERT INTO sender_ids (tenant_id, sender_id)
VALUES (@tenant_id, @sender_id)
ON CONFLICT (tenant_id, sender_id) DO NOTHING;

-- name: IsSenderIDApproved :one
SELECT EXISTS (
    SELECT 1 FROM sender_ids WHERE tenant_id = @tenant_id AND sender_id = @sender_id
) AS approved;

-- name: ListSenderIDs :many
SELECT sender_id
FROM sender_ids
WHERE tenant_id = @tenant_id
ORDER BY sender_id;

-- name: GetLiveIdempotencyRecord :one
SELECT request_fingerprint, message_id
FROM idempotency_keys
WHERE tenant_id = @tenant_id AND operation = @operation AND idempotency_key = @idempotency_key
  AND expires_at > now();

-- name: RecordIdempotencyKey :one
-- Inserts the record, or replaces an expired one (an expired key is treated as new, ADR-0018).
-- Returns no row if a live record exists (a concurrent request committed it first).
INSERT INTO idempotency_keys (tenant_id, operation, idempotency_key, request_fingerprint, message_id, expires_at)
VALUES (@tenant_id, @operation, @idempotency_key, @request_fingerprint, @message_id,
        now() + make_interval(secs => @retention_seconds::double precision))
ON CONFLICT (tenant_id, operation, idempotency_key) DO UPDATE
SET request_fingerprint = EXCLUDED.request_fingerprint,
    message_id = EXCLUDED.message_id,
    created_at = now(),
    expires_at = EXCLUDED.expires_at
WHERE idempotency_keys.expires_at <= now()
RETURNING message_id;

-- name: InsertMessage :one
INSERT INTO messages (tenant_id, client_reference, destination, sender_id, encoding, segment_count)
VALUES (@tenant_id, @client_reference, @destination, @sender_id, @encoding, @segment_count)
RETURNING id, tenant_id, client_reference, destination, sender_id, encoding, segment_count, status,
          status_reason, accepted_at, status_updated_at;

-- name: InsertMessageBody :exec
INSERT INTO message_bodies (tenant_id, message_id, body) VALUES (@tenant_id, @message_id, @body);

-- name: InsertDispatchJob :exec
INSERT INTO dispatch_jobs (tenant_id, message_id) VALUES (@tenant_id, @message_id);

-- name: InsertStatusHistory :exec
INSERT INTO message_status_history (tenant_id, message_id, from_status, to_status, status_reason, source, attempt_id, receipt_id)
VALUES (@tenant_id, @message_id, @from_status, @to_status, @status_reason, @source, @attempt_id, @receipt_id);

-- name: GetMessage :one
SELECT id, tenant_id, client_reference, destination, sender_id, encoding, segment_count, status,
       status_reason, accepted_at, status_updated_at
FROM messages
WHERE tenant_id = @tenant_id AND id = @id;

-- name: ListMessages :many
-- Newest acceptance first; keyset pagination on (accepted_at, id). Optional filters are NULL when
-- not given.
SELECT id, tenant_id, client_reference, destination, sender_id, encoding, segment_count, status,
       status_reason, accepted_at, status_updated_at
FROM messages
WHERE tenant_id = @tenant_id
  AND (sqlc.narg(client_reference)::text IS NULL OR client_reference = sqlc.narg(client_reference)::text)
  AND (sqlc.narg(states)::text[] IS NULL OR status = ANY (sqlc.narg(states)::text[]))
  AND (sqlc.narg(accepted_after)::timestamptz IS NULL OR accepted_at >= sqlc.narg(accepted_after)::timestamptz)
  AND (sqlc.narg(accepted_before)::timestamptz IS NULL OR accepted_at < sqlc.narg(accepted_before)::timestamptz)
  AND (sqlc.narg(cursor_accepted_at)::timestamptz IS NULL
       OR (accepted_at, id) < (sqlc.narg(cursor_accepted_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
ORDER BY accepted_at DESC, id DESC
LIMIT @page_limit;

-- name: ListStatusHistory :many
SELECT from_status, to_status, status_reason, source, attempt_id, receipt_id, occurred_at
FROM message_status_history
WHERE tenant_id = @tenant_id AND message_id = @message_id
ORDER BY occurred_at, id;

-- name: ListSubmissionAttempts :many
SELECT id, attempt_number, provider_id, state, provider_message_ref, provider_result_code, started_at, finished_at
FROM submission_attempts
WHERE tenant_id = @tenant_id AND message_id = @message_id
ORDER BY attempt_number;

-- name: ListMessageReceipts :many
SELECT provider_receipt_id, identity_key, receipt_status, effect, processed_at
FROM message_receipts
WHERE tenant_id = @tenant_id AND message_id = @message_id
ORDER BY processed_at, id;

-- Dispatcher -----------------------------------------------------------------------------------

-- name: ClaimDispatchJobs :many
-- Claims up to batch_size due jobs of the tenant with a lease. SKIP LOCKED lets several
-- dispatchers run at once; an expired lease (a crashed dispatcher) makes a job claimable again.
WITH due AS (
    SELECT j.message_id
    FROM dispatch_jobs j
    WHERE j.tenant_id = @tenant_id AND j.completed_at IS NULL AND j.available_at <= now()
      AND (j.lease_expires_at IS NULL OR j.lease_expires_at <= now())
    ORDER BY j.available_at
    LIMIT @batch_size
    FOR UPDATE SKIP LOCKED
)
UPDATE dispatch_jobs AS j
SET lease_expires_at = now() + make_interval(secs => @lease_seconds::double precision), updated_at = now()
FROM due
WHERE j.tenant_id = @tenant_id AND j.message_id = due.message_id
RETURNING j.message_id;

-- name: LockMessageForDispatch :one
SELECT m.status, m.accepted_at, m.destination, m.sender_id, m.encoding, m.segment_count, b.body,
       (SELECT count(*) FROM submission_attempts a WHERE a.tenant_id = m.tenant_id AND a.message_id = m.id)::integer AS attempts
FROM messages m
JOIN message_bodies b ON b.tenant_id = m.tenant_id AND b.message_id = m.id
WHERE m.tenant_id = @tenant_id AND m.id = @id
FOR UPDATE OF m;

-- name: StartSubmissionAttempt :one
INSERT INTO submission_attempts (tenant_id, message_id, attempt_number, provider_id)
VALUES (@tenant_id, @message_id, @attempt_number, @provider_id)
RETURNING id;

-- name: FinishSubmissionAttempt :execrows
UPDATE submission_attempts
SET state = @state, provider_message_ref = @provider_message_ref, provider_result_code = @provider_result_code,
    finished_at = now()
WHERE tenant_id = @tenant_id AND id = @id AND state = 'in_progress';

-- name: TransitionMessage :execrows
-- Changes the status only if the message is still in from_status (compare-and-set).
UPDATE messages
SET status = @to_status, status_reason = @status_reason, status_updated_at = now(), updated_at = now(),
    provider_id = COALESCE(sqlc.narg(provider_id)::text, provider_id)
WHERE tenant_id = @tenant_id AND id = @id AND status = @from_status;

-- name: CompleteDispatchJob :exec
UPDATE dispatch_jobs
SET completed_at = now(), lease_expires_at = NULL, updated_at = now()
WHERE tenant_id = @tenant_id AND message_id = @message_id;

-- name: RescheduleDispatchJob :exec
UPDATE dispatch_jobs
SET available_at = now() + make_interval(secs => @delay_seconds::double precision), lease_expires_at = NULL,
    completed_at = NULL, updated_at = now()
WHERE tenant_id = @tenant_id AND message_id = @message_id;

-- name: ClaimStaleAttempts :many
-- Attempts still in progress long after the submission timeout: their dispatcher crashed or lost
-- the result. They become uncertain; they are never resubmitted (messaging reliability §4).
SELECT id, message_id
FROM submission_attempts
WHERE tenant_id = @tenant_id AND state = 'in_progress'
  AND started_at < now() - make_interval(secs => @stale_after_seconds::double precision)
ORDER BY started_at
LIMIT @batch_size
FOR UPDATE SKIP LOCKED;

-- name: ClaimOverdueSubmitted :many
-- Submitted messages of a provider without a final report within its reporting window.
SELECT id
FROM messages
WHERE tenant_id = @tenant_id AND status = 'submitted' AND provider_id = @provider_id
  AND status_updated_at < now() - make_interval(secs => @window_seconds::double precision)
ORDER BY status_updated_at
LIMIT @batch_size
FOR UPDATE SKIP LOCKED;

-- Receipts -------------------------------------------------------------------------------------

-- name: InsertProviderReceipt :one
INSERT INTO provider_receipts (provider_id, provider_message_ref, identity_key, receipt_status,
                               provider_status_code, provider_reported_at, raw)
VALUES (@provider_id, @provider_message_ref, @identity_key, @receipt_status,
        @provider_status_code, @provider_reported_at, @raw)
RETURNING id;

-- name: ClaimMatchableReceipts :many
-- Unmatched receipts that correlate with THIS tenant's submission attempts. Row-level security on
-- submission_attempts restricts the join to the tenant in the transaction's context.
SELECT r.id, r.identity_key, r.receipt_status, a.message_id
FROM provider_receipts r
JOIN submission_attempts a
  ON a.tenant_id = @tenant_id AND a.provider_id = r.provider_id AND a.provider_message_ref = r.provider_message_ref
WHERE r.matched_at IS NULL
ORDER BY r.received_at, r.id
LIMIT @batch_size
FOR UPDATE OF r SKIP LOCKED;

-- name: MarkProviderReceiptMatched :exec
UPDATE provider_receipts SET matched_at = now() WHERE id = @id;

-- name: LockMessageStatus :one
SELECT status FROM messages WHERE tenant_id = @tenant_id AND id = @id FOR UPDATE;

-- name: ReceiptIdentitySeen :one
SELECT EXISTS (
    SELECT 1 FROM message_receipts
    WHERE tenant_id = @tenant_id AND message_id = @message_id AND identity_key = @identity_key
) AS seen;

-- name: InsertMessageReceipt :exec
INSERT INTO message_receipts (tenant_id, message_id, provider_receipt_id, identity_key, receipt_status, effect)
VALUES (@tenant_id, @message_id, @provider_receipt_id, @identity_key, @receipt_status, @effect);

-- name: FlagConflictingReports :exec
UPDATE messages SET has_conflicting_reports = true, updated_at = now()
WHERE tenant_id = @tenant_id AND id = @id;
