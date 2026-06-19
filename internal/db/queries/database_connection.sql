-- name: CreateDatabaseConnection :one
INSERT INTO database_connections (
    org_id,
    display_name,
    host,
    port,
    database_name,
    username,
    password_encrypted,
    ssl_enabled,
    ssl_mode,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetDatabaseConnection :one
SELECT * FROM database_connections
WHERE id = $1;

-- name: ListDatabaseConnectionsByOrg :many
SELECT * FROM database_connections
WHERE org_id = $1
ORDER BY display_name;

-- name: UpdateDatabaseConnection :one
UPDATE database_connections
SET
    display_name = $2,
    host = $3,
    port = $4,
    database_name = $5,
    username = $6,
    password_encrypted = $7,
    ssl_enabled = $8,
    ssl_mode = $9,
    status = $10,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateDatabaseConnectionTestResult :one
UPDATE database_connections
SET
    status = $2,
    last_tested_at = NOW(),
    last_test_passed = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteDatabaseConnection :exec
DELETE FROM database_connections
WHERE id = $1;
