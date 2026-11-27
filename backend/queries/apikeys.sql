-- API keys (internal/apikey, Phase 4 Group 2). Every query runs inside the tenant's
-- transaction-local context (database.DB.InTenantTx) and also filters by tenant_id explicitly.

-- name: CreateAPIKey :one
INSERT INTO api_keys (tenant_id, label, secret_hash, scopes)
VALUES (@tenant_id, @label, @secret_hash, @scopes)
RETURNING id, tenant_id, label, scopes, created_at, revoked_at;

-- name: GetAPIKeyCredential :one
SELECT id, tenant_id, secret_hash, scopes, revoked_at
FROM api_keys
WHERE tenant_id = @tenant_id AND id = @id;

-- name: RevokeAPIKey :one
UPDATE api_keys
SET revoked_at = now()
WHERE tenant_id = @tenant_id AND id = @id AND revoked_at IS NULL
RETURNING id, tenant_id, label, scopes, created_at, revoked_at;

-- name: ListAPIKeys :many
SELECT id, tenant_id, label, scopes, created_at, revoked_at
FROM api_keys
WHERE tenant_id = @tenant_id
ORDER BY created_at, id;
