-- name: CreateOrganization :one
INSERT INTO organizations (
    name,
    slug,
    smtp_host,
    smtp_port,
    smtp_username,
    smtp_password_encrypted,
    smtp_from_address,
    auto_approve_additive
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetOrganization :one
SELECT * FROM organizations
WHERE id = $1;

-- name: ListOrganizations :many
SELECT * FROM organizations
ORDER BY name;

-- name: UpdateOrganization :one
UPDATE organizations
SET
    name = $2,
    slug = $3,
    smtp_host = $4,
    smtp_port = $5,
    smtp_username = $6,
    smtp_password_encrypted = $7,
    smtp_from_address = $8,
    auto_approve_additive = $9,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteOrganization :exec
DELETE FROM organizations
WHERE id = $1;
