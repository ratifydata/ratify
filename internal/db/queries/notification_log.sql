-- name: CreateNotificationLog :one
INSERT INTO notification_logs (
    org_id,
    channel,
    recipient,
    subject,
    status,
    failure_reason,
    related_entity_id,
    related_entity_type,
    sent_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetNotificationLog :one
SELECT * FROM notification_logs
WHERE id = $1;

-- name: ListNotificationLogsByOrg :many
SELECT * FROM notification_logs
WHERE org_id = $1
ORDER BY created_at DESC;

-- name: ListNotificationLogsByRelatedEntity :many
SELECT * FROM notification_logs
WHERE related_entity_type = $1
  AND related_entity_id = $2
ORDER BY created_at DESC;

-- name: UpdateNotificationLogStatus :one
UPDATE notification_logs
SET
    status = $2,
    failure_reason = $3,
    sent_at = $4
WHERE id = $1
RETURNING *;
