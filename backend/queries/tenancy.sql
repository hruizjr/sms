-- name: CreateTenant :one
-- id, status ('pending') and both timestamps come from the column defaults; the application role
-- may insert display_name only.
INSERT INTO tenants (display_name)
VALUES (@display_name)
RETURNING id, display_name, status, created_at, updated_at;

-- name: GetTenant :one
SELECT id, display_name, status, created_at, updated_at
FROM tenants
WHERE id = @id;

-- name: ListTenants :many
-- Keyset pagination in id order (UUIDv7, so roughly creation order). Pass the nil UUID for the
-- first page and the last id of the previous page afterwards.
SELECT id, display_name, status, created_at, updated_at
FROM tenants
WHERE id > @after_id
ORDER BY id
LIMIT @page_size;

-- name: TransitionTenantStatus :one
-- Atomic compare-and-set: changes the status only if it is still from_status, so concurrent
-- transitions cannot both succeed. No row is returned when the tenant does not exist or its status
-- differs.
UPDATE tenants
SET status = @to_status, updated_at = greatest(now(), updated_at)
WHERE id = @id AND status = @from_status
RETURNING id, display_name, status, created_at, updated_at;
