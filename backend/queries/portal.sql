-- Client portal accounts and sessions (internal/portal, Phase 4 Group 3). Every query on
-- portal_users and portal_sessions runs inside the tenant's transaction-local context
-- (database.DB.InTenantTx) and also filters by tenant_id explicitly. portal_login_directory is the
-- platform-owned login lookup (decision log §20, G3-1).

-- name: LookupLoginDirectory :one
SELECT tenant_id, user_id
FROM portal_login_directory
WHERE email = @email;

-- name: CreatePortalUser :one
INSERT INTO portal_users (tenant_id, email, role, password_hash)
VALUES (@tenant_id, @email, @role, @password_hash)
RETURNING id, tenant_id, email, role, created_at;

-- name: InsertLoginDirectoryEntry :exec
INSERT INTO portal_login_directory (email, tenant_id, user_id)
VALUES (@email, @tenant_id, @user_id);

-- name: GetPortalUserCredential :one
SELECT id, tenant_id, email, role, password_hash, failed_login_count, locked_until
FROM portal_users
WHERE tenant_id = @tenant_id AND id = @id;

-- name: ListPortalUsers :many
SELECT id, tenant_id, email, role, locked_until, last_login_at, created_at
FROM portal_users
WHERE tenant_id = @tenant_id
ORDER BY email;

-- name: GetPortalUserByEmail :one
SELECT id, tenant_id, email, role
FROM portal_users
WHERE tenant_id = @tenant_id AND email = @email;

-- name: RecordFailedLogin :one
-- Counts one failed sign-in atomically (also used for failed re-authentication, G4-7). A lock that has run out starts a new count; the lock starts
-- when the count reaches @max_failures. (Every expression sees the row as it was before the update.)
UPDATE portal_users
SET failed_login_count = CASE
        WHEN locked_until IS NOT NULL AND locked_until <= @now::timestamptz THEN 1
        ELSE failed_login_count + 1
    END,
    locked_until = CASE
        WHEN locked_until IS NOT NULL AND locked_until > @now::timestamptz THEN locked_until
        WHEN locked_until IS NULL AND failed_login_count + 1 >= @max_failures::integer THEN @lock_until::timestamptz
        WHEN locked_until IS NOT NULL AND 1 >= @max_failures::integer THEN @lock_until::timestamptz
        ELSE NULL
    END,
    updated_at = GREATEST(updated_at, @now::timestamptz)
WHERE tenant_id = @tenant_id AND id = @id
RETURNING failed_login_count, locked_until,
          -- true only for the update that started this lock (it stored this call's @lock_until)
          COALESCE(locked_until = @lock_until::timestamptz, false)::boolean AS newly_locked;

-- name: RecordSuccessfulLogin :exec
UPDATE portal_users
SET failed_login_count = 0,
    locked_until = NULL,
    last_login_at = @now::timestamptz,
    updated_at = GREATEST(updated_at, @now::timestamptz)
WHERE tenant_id = @tenant_id AND id = @id;

-- name: UnlockPortalUser :execrows
UPDATE portal_users
SET failed_login_count = 0,
    locked_until = NULL,
    updated_at = GREATEST(updated_at, @now::timestamptz)
WHERE tenant_id = @tenant_id AND id = @id;

-- name: CreatePortalSession :one
INSERT INTO portal_sessions (tenant_id, user_id, secret_hash, created_at, last_seen_at, expires_at)
VALUES (@tenant_id, @user_id, @secret_hash, @now::timestamptz, @now::timestamptz, @expires_at)
RETURNING id;

-- name: GetPortalSession :one
SELECT s.id, s.tenant_id, s.user_id, s.secret_hash, s.created_at, s.last_seen_at, s.expires_at,
       s.revoked_at, u.email, u.role
FROM portal_sessions s
JOIN portal_users u ON u.tenant_id = s.tenant_id AND u.id = s.user_id
WHERE s.tenant_id = @tenant_id AND s.id = @id;

-- name: TouchPortalSession :exec
UPDATE portal_sessions
SET last_seen_at = GREATEST(last_seen_at, @now::timestamptz)
WHERE tenant_id = @tenant_id AND id = @id AND revoked_at IS NULL;

-- name: RevokePortalSession :execrows
UPDATE portal_sessions
SET revoked_at = GREATEST(created_at, @now::timestamptz)
WHERE tenant_id = @tenant_id AND id = @id AND revoked_at IS NULL;

-- name: RevokeUserPortalSessions :execrows
UPDATE portal_sessions
SET revoked_at = GREATEST(created_at, @now::timestamptz)
WHERE tenant_id = @tenant_id AND user_id = @user_id AND revoked_at IS NULL;
