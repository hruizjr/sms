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

-- name: DeleteEndedSignInFailureBudgets :execrows
-- Removes budgets whose window has ended or that hold no reserved slot (an absent row is the same as
-- an empty budget).
DELETE FROM portal_sign_in_failure_budgets
WHERE window_start_at <= @window_ended_before::timestamptz OR failure_count = 0;

-- name: DeleteFullSignInRequestBuckets :execrows
-- Removes buckets that have refilled completely (an absent row is the same as a full bucket) and
-- whose last refusal report is older than the report interval.
DELETE FROM portal_sign_in_request_buckets
WHERE refilled_at <= @full_before::timestamptz
  AND (refusal_reported_at IS NULL OR refusal_reported_at <= @report_interval_before::timestamptz);
