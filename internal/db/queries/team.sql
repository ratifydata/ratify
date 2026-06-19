-- name: CreateTeam :one
INSERT INTO teams (
    org_id,
    name,
    description
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetTeam :one
SELECT * FROM teams
WHERE id = $1;

-- name: ListTeamsByOrg :many
SELECT * FROM teams
WHERE org_id = $1
ORDER BY name;

-- name: UpdateTeam :one
UPDATE teams
SET
    name = $2,
    description = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteTeam :exec
DELETE FROM teams
WHERE id = $1;
