-- Per-source sign-in rate limit of the client portal (internal/signinlimit, Phase 4 Group 6).
-- portal_sign_in_rate_limits is platform-owned (no tenant data) and shared by every API instance.

-- name: CountSignInAttempt :one
-- Counts one sign-in attempt from @source atomically and returns the window's count and start. A
-- window that has ended (window_start_at <= @window_ended_before) starts again with this attempt.
-- Concurrent attempts from the same source are serialized on the row (INSERT ... ON CONFLICT DO
-- UPDATE), so no attempt is lost or counted twice. The count stops growing at the integer maximum.
INSERT INTO portal_sign_in_rate_limits AS l (source, window_start_at, attempt_count)
VALUES (@source, @now::timestamptz, 1)
ON CONFLICT (source) DO UPDATE
SET window_start_at = CASE WHEN l.window_start_at <= @window_ended_before::timestamptz THEN @now::timestamptz
                           ELSE l.window_start_at END,
    attempt_count = CASE WHEN l.window_start_at <= @window_ended_before::timestamptz THEN 1
                         ELSE LEAST(l.attempt_count, 2147483646) + 1 END
RETURNING attempt_count, window_start_at;

-- name: DeleteEndedSignInWindows :execrows
-- Removes the counters of windows that have ended (their sources start a new window anyway).
DELETE FROM portal_sign_in_rate_limits
WHERE window_start_at <= @window_ended_before::timestamptz;
