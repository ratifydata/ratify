-- name: CreateAuditEvent :one
INSERT INTO audit_events (
    org_id,
    actor_user_id,
    actor_type,
    event_type,
    entity_type,
    entity_id,
    event_data
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetAuditEvent :one
SELECT * FROM audit_events
WHERE id = $1;

-- name: ListAuditEventsByOrg :many
SELECT * FROM audit_events
WHERE org_id = $1
ORDER BY occurred_at DESC;

-- name: ListAuditEventsByActor :many
SELECT * FROM audit_events
WHERE actor_user_id = $1
ORDER BY occurred_at DESC;

-- name: ListAuditEventsByEntity :many
SELECT * FROM audit_events
WHERE entity_type = $1
  AND entity_id = $2
ORDER BY occurred_at DESC;
