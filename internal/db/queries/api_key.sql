-- name: CreateAPIKey :one
INSERT INTO api_keys (
    user_id,
    org_id,
    name,
    key_hash,
    key_prefix,
    scope,
    is_active,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING user_id,org_id,name,key_hash,key_prefix,scope,is_active,expires_at;

-- name: GetAPIKey :one
SELECT * FROM api_keys
WHERE id = $1;

-- name: GetAPIKeyByHash :one
SELECT * FROM api_keys
WHERE key_hash = $1;

-- name: GetAPIKeyByPrefix :one
SELECT * FROM api_keys
WHERE key_prefix = $1  AND is_active = $2;

-- name: ListAPIKeysByUser :many
SELECT * FROM api_keys
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: ListAPIKeysByOrg :many
SELECT * FROM api_keys
WHERE org_id = $1
ORDER BY created_at DESC;

-- name: UpdateAPIKey :one
UPDATE api_keys
SET
    name = $2,
    scope = $3,
    is_active = $4,
    expires_at = $5
WHERE id = $1
RETURNING *;

-- name: UpdateAPIKeyLastUsed :one
UPDATE api_keys
SET last_used_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteAPIKey :exec
UPDATE  api_keys
SET is_active = false
WHERE id = $1;
