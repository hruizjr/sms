-- Layered sign-in limits of the client portal (internal/signinlimit, Phase 4 Group 6). Both tables are
-- platform-owned (no tenant data) and shared by every API instance. Every statement is atomic on its
-- row (INSERT ... ON CONFLICT DO UPDATE, or a conditional UPDATE), so concurrent attempts, on one
-- instance or several, are serialized on the row and never lost or counted twice.

-- name: ReserveSignInFailure :one
-- Reserves one failed-attempt slot of source and returns the window's count (including this slot)
-- and start. A window that has ended (window_start_at <= window_ended_before) starts again with this
-- attempt. The caller runs it in a transaction for every budget of the attempt and rolls the
-- transaction back when any budget is over its limit, so a refused attempt reserves nothing.
INSERT INTO portal_sign_in_failure_budgets AS b (source, window_start_at, failure_count)
VALUES (@source, @now::timestamptz, 1)
ON CONFLICT (source) DO UPDATE
SET window_start_at = CASE WHEN b.window_start_at <= @window_ended_before::timestamptz THEN @now::timestamptz
                           ELSE b.window_start_at END,
    failure_count = CASE WHEN b.window_start_at <= @window_ended_before::timestamptz THEN 1
                         ELSE LEAST(b.failure_count, 2147483646) + 1 END
RETURNING failure_count, window_start_at;

-- name: RefundSignInFailure :execrows
-- Returns the slot an attempt reserved in the window starting at window_start_at (a successful
-- sign-in, or an attempt whose password was never checked). Nothing changes when that window has
-- ended and a new one started, so a refund never reduces a later window's count.
UPDATE portal_sign_in_failure_budgets
SET failure_count = failure_count - 1
WHERE source = @source AND window_start_at = @window_start_at::timestamptz AND failure_count > 0;

-- name: MarkSignInFailureRefusalReported :execrows
-- Claims the one platform event of a refused budget's window: 1 row for the first caller of the
-- window, 0 for every later one.
UPDATE portal_sign_in_failure_budgets
SET refusal_reported_for = window_start_at
WHERE source = @source AND window_start_at = @window_start_at::timestamptz
  AND refusal_reported_for IS DISTINCT FROM window_start_at;

-- name: TakeSignInRequestToken :one
-- The all-attempt token bucket of a source: it refills at rate tokens per second up to burst, and the
-- request takes one token if one is available. In the update, AVAILABLE stands for
--   LEAST(burst::float8, b.tokens + GREATEST(0, date_part('epoch', CAST(now AS timestamptz) - b.refilled_at)) * rate::float8)
-- (the tokens after refilling; an instance whose clock is behind the row's adds none, and refilled_at
-- never moves backwards). last_allowed is whether this request took a token; last_reported is whether
-- this refusal is the first since report_interval_before (the caller then records one platform event).
INSERT INTO portal_sign_in_request_buckets AS b (source, tokens, refilled_at, last_allowed)
VALUES (@source, CAST(@burst AS float8) - 1, @now::timestamptz, true)
ON CONFLICT (source) DO UPDATE
SET tokens = CASE WHEN LEAST(@burst::float8, b.tokens + GREATEST(0, date_part('epoch', CAST(@now AS timestamptz) - b.refilled_at)) * @rate::float8) >= 1 THEN LEAST(@burst::float8, b.tokens + GREATEST(0, date_part('epoch', CAST(@now AS timestamptz) - b.refilled_at)) * @rate::float8) - 1 ELSE LEAST(@burst::float8, b.tokens + GREATEST(0, date_part('epoch', CAST(@now AS timestamptz) - b.refilled_at)) * @rate::float8) END,
    refilled_at = GREATEST(b.refilled_at, @now::timestamptz),
    last_allowed = LEAST(@burst::float8, b.tokens + GREATEST(0, date_part('epoch', CAST(@now AS timestamptz) - b.refilled_at)) * @rate::float8) >= 1,
    last_reported = LEAST(@burst::float8, b.tokens + GREATEST(0, date_part('epoch', CAST(@now AS timestamptz) - b.refilled_at)) * @rate::float8) < 1
        AND (b.refusal_reported_at IS NULL OR b.refusal_reported_at <= @report_interval_before::timestamptz),
    refusal_reported_at = CASE
        WHEN LEAST(@burst::float8, b.tokens + GREATEST(0, date_part('epoch', CAST(@now AS timestamptz) - b.refilled_at)) * @rate::float8) < 1 AND (b.refusal_reported_at IS NULL OR b.refusal_reported_at <= @report_interval_before::timestamptz)
        THEN @now::timestamptz
        ELSE b.refusal_reported_at
    END
RETURNING tokens, last_allowed, last_reported;

-- Cleanup (Limiter.Cleanup). Each statement deletes one bounded batch and commits on its own. The
-- rows of a batch are chosen with FOR UPDATE SKIP LOCKED, so cleanup never waits for a row that a
-- sign-in (or another API instance's cleanup) has locked: it skips the row and a later cycle takes it.
-- A sign-in may wait for the few milliseconds of one batch, but cleanup never waits for a sign-in, so
-- the two cannot deadlock whatever order they lock rows in (a sign-in locks an IPv6 /48 before its
-- /64; one unbatched DELETE locked rows in table order and could deadlock with it). Locking re-checks
-- the latest version of each row, so a row that a sign-in moved to a new window is left alone; the
-- DELETE repeats the predicate as a second check.

-- name: DeleteEndedSignInFailureBudgets :execrows
-- Deletes up to batch_size budgets whose window has ended (window_start_at <= window_ended_before),
-- oldest first; an ended window holds nothing (the next reservation would start a new one). The batch
-- is chosen through ix_portal_sign_in_failure_budgets__window_start_at and deleted through the
-- primary key, so a batch costs the same however many rows wait. A budget whose reserved slots were
-- all refunded stays until its window ends, like any other: an empty budget and an absent one behave
-- the same, and keeping it avoids an index on the counter (which every reservation and refund
-- changes, so such an index would make every one of those updates a non-HOT update).
DELETE FROM portal_sign_in_failure_budgets AS b
WHERE b.source = ANY (ARRAY(
        SELECT c.source
        FROM portal_sign_in_failure_budgets AS c
        WHERE c.window_start_at <= @window_ended_before::timestamptz
        ORDER BY c.window_start_at
        LIMIT @batch_size
        FOR UPDATE SKIP LOCKED
    ))
  AND b.window_start_at <= @window_ended_before::timestamptz;

-- name: DeleteFullSignInRequestBuckets :execrows
-- Deletes up to batch_size buckets that have refilled completely (an absent row is the same as a full
-- bucket) and whose last refusal report is older than the report interval, least recently used first.
-- The batch is chosen through ix_portal_sign_in_request_buckets__refilled_at and deleted through the
-- primary key.
DELETE FROM portal_sign_in_request_buckets AS b
WHERE b.source = ANY (ARRAY(
        SELECT c.source
        FROM portal_sign_in_request_buckets AS c
        WHERE c.refilled_at <= @full_before::timestamptz
          AND (c.refusal_reported_at IS NULL OR c.refusal_reported_at <= @report_interval_before::timestamptz)
        ORDER BY c.refilled_at
        LIMIT @batch_size
        FOR UPDATE SKIP LOCKED
    ))
  AND b.refilled_at <= @full_before::timestamptz
  AND (b.refusal_reported_at IS NULL OR b.refusal_reported_at <= @report_interval_before::timestamptz);
