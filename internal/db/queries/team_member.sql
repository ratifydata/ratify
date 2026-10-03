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

-- name: TeamMemberExist :one
SELECT EXISTS (
    SELECT 1 FROM team_members
    WHERE team_id = $1
      AND user_id = $2
) AS team_member_exists;

-- name: ListTeamMembersByTeam :many
SELECT tm.team_id, tm.user_id, tm.role, u.email
FROM team_members tm
JOIN users u ON u.id = tm.user_id
WHERE tm.team_id = $1 AND tm.deleted_at IS NULL
ORDER BY tm.joined_at, tm.user_id;

-- name: UpdateTeamMemberRole :one
UPDATE team_members
SET role = $3
WHERE team_id = $1
  AND user_id = $2
RETURNING *;

-- name: RemoveTeamMember :one
UPDATE team_members
SET deleted_at = NOW()
WHERE team_id = $1
  AND user_id = $2
RETURNING *;
