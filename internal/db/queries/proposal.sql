-- name: CreateProposal :one
INSERT INTO proposals (
    contract_id,
    raised_by,
    title,
    description,
    status,
    deadline,
    requires_unanimous_approval
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetProposal :one
SELECT * FROM proposals
WHERE id = $1;

-- name: ListProposalsByContract :many
SELECT * FROM proposals
WHERE contract_id = $1
ORDER BY created_at DESC;

-- name: ListProposalsByRaiser :many
SELECT * FROM proposals
WHERE raised_by = $1
ORDER BY created_at DESC;

-- name: ListProposalsByStatus :many
SELECT * FROM proposals
WHERE status = $1
ORDER BY deadline;

-- name: UpdateProposal :one
UPDATE proposals
SET
    title = $2,
    description = $3,
    status = $4,
    deadline = $5,
    requires_unanimous_approval = $6,
    resolved_at = $7
WHERE id = $1
RETURNING *;

-- name: DeleteProposal :exec
DELETE FROM proposals
WHERE id = $1;
