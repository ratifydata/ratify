-- name: CreateProposalChange :one
INSERT INTO proposal_changes (
    proposal_id,
    change_type,
    classification,
    affected_element,
    before_value,
    after_value
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetProposalChange :one
SELECT * FROM proposal_changes
WHERE id = $1;

-- name: ListProposalChangesByProposal :many
SELECT * FROM proposal_changes
WHERE proposal_id = $1
ORDER BY affected_element, change_type;

-- name: UpdateProposalChange :one
UPDATE proposal_changes
SET
    change_type = $2,
    classification = $3,
    affected_element = $4,
    before_value = $5,
    after_value = $6
WHERE id = $1
RETURNING *;

-- name: DeleteProposalChange :exec
DELETE FROM proposal_changes
WHERE id = $1;
