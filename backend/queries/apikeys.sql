-- API keys (internal/apikey, Phase 4 Group 2). Every query runs inside the tenant's
-- transaction-local context (database.DB.InTenantTx) and also filters by tenant_id explicitly.

-- name: CreateAPIKey :one
INSERT INTO api_keys (tenant_id, label, secret_hash, scopes)
VALUES (@tenant_id, @label, @secret_hash, @scopes)
RETURNING id, tenant_id, label, scopes, created_at, revoked_at, last_used_at;

-- name: GetAPIKeyCredential :one
SELECT id, tenant_id, secret_hash, scopes, revoked_at, last_used_at
FROM api_keys
WHERE tenant_id = @tenant_id AND id = @id;

-- name: RecordAPIKeyUse :execrows
-- Records that an active key authenticated a request, at most once per interval: the update is
-- skipped when the stored time is newer than @stale_before, so concurrent requests on any number of
-- API instances write at most once per key and interval (the condition is re-checked on the locked
-- row). Revoked keys are never updated. GREATEST keeps the time at or after created_at, which the
-- database set with its own clock.
UPDATE api_keys
SET last_used_at = GREATEST(@now::timestamptz, created_at)
WHERE tenant_id = @tenant_id AND id = @id AND revoked_at IS NULL
  AND (last_used_at IS NULL OR last_used_at <= @stale_before::timestamptz);

-- name: RevokeAPIKey :one
UPDATE api_keys
SET revoked_at = now()
WHERE tenant_id = @tenant_id AND id = @id AND revoked_at IS NULL
RETURNING id, tenant_id, label, scopes, created_at, revoked_at, last_used_at;

-- name: ListAPIKeys :many
SELECT id, tenant_id, label, scopes, created_at, revoked_at, last_used_at
FROM api_keys
WHERE tenant_id = @tenant_id
ORDER BY created_at, id;
