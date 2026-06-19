-- name: CreateTeamMember :one
INSERT INTO team_members (
    team_id,
    user_id,
    role
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetTeamMember :one
SELECT * FROM team_members
WHERE team_id = $1
  AND user_id = $2;

-- name: ListTeamMembersByTeam :many
SELECT * FROM team_members
WHERE team_id = $1
ORDER BY joined_at;

-- name: UpdateTeamMemberRole :one
UPDATE team_members
SET role = $3
WHERE team_id = $1
  AND user_id = $2
RETURNING *;

-- name: DeleteTeamMember :exec
DELETE FROM team_members
WHERE team_id = $1
  AND user_id = $2;
