-- Security audit log (internal/audit, Phase 4 Group 6). Tenant events are written and read inside
-- the tenant's transaction-local context (database.DB.InTenantTx) and filter by tenant_id
-- explicitly; the application can only insert them (no UPDATE or DELETE privilege). Platform events
-- belong to no organization and can only be inserted: the application cannot read them.

-- name: InsertSecurityAuditEvent :exec
INSERT INTO security_audit_events (
    tenant_id, occurred_at, action, outcome, actor_type, actor_id, target_type, target_id,
    request_id, source_ip, user_agent, details
) VALUES (
    @tenant_id, @occurred_at, @action, @outcome, @actor_type, sqlc.narg(actor_id), sqlc.narg(target_type),
    sqlc.narg(target_id), sqlc.narg(request_id), sqlc.narg(source_ip), sqlc.narg(user_agent), @details
);

-- name: InsertPlatformSecurityEvent :exec
INSERT INTO platform_security_events (occurred_at, action, outcome, request_id, source_ip, user_agent, details)
VALUES (@occurred_at, @action, @outcome, sqlc.narg(request_id), sqlc.narg(source_ip), sqlc.narg(user_agent), @details);

-- name: ListSecurityAuditEvents :many
-- Newest first; keyset pagination on (occurred_at, id). Optional filters are NULL when not given.
-- The email addresses of the portal users involved are read from portal_users (same tenant), so
-- the audit log itself stores user IDs, not copies of email addresses.
SELECT e.id, e.occurred_at, e.action, e.outcome, e.actor_type, e.actor_id, e.target_type, e.target_id,
       e.request_id, e.source_ip, e.user_agent, e.details,
       actor_user.email AS actor_email, target_user.email AS target_email
FROM security_audit_events e
LEFT JOIN portal_users actor_user
       ON e.actor_type = 'portal_user' AND actor_user.tenant_id = e.tenant_id AND actor_user.id = e.actor_id
LEFT JOIN portal_users target_user
       ON e.target_type = 'portal_user' AND target_user.tenant_id = e.tenant_id AND target_user.id = e.target_id
WHERE e.tenant_id = @tenant_id
  AND (sqlc.narg(action)::text IS NULL OR e.action = sqlc.narg(action)::text)
  AND (sqlc.narg(outcome)::text IS NULL OR e.outcome = sqlc.narg(outcome)::text)
  AND (sqlc.narg(occurred_from)::timestamptz IS NULL OR e.occurred_at >= sqlc.narg(occurred_from)::timestamptz)
  AND (sqlc.narg(occurred_before)::timestamptz IS NULL OR e.occurred_at < sqlc.narg(occurred_before)::timestamptz)
  AND (sqlc.narg(cursor_occurred_at)::timestamptz IS NULL
       OR (e.occurred_at, e.id) < (sqlc.narg(cursor_occurred_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
ORDER BY e.occurred_at DESC, e.id DESC
LIMIT @page_limit;
