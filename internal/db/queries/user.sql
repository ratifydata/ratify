-- name: CreateUser :one
INSERT INTO users (
    org_id,
    email,
    display_name,
    is_active
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetUser :one
SELECT * FROM users
WHERE id = $1;

-- name: ListUsersByOrg :many
SELECT * FROM users
WHERE org_id = $1
ORDER BY display_name;

-- name: UpdateUser :one
UPDATE users
SET
    email = $2,
    display_name = $3,
    is_active = $4,
    last_login_at = $5
WHERE id = $1
RETURNING *;

-- name: UpdateUserLastLogin :one
UPDATE users
SET last_login_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;
