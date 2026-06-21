-- name: CreateConsumerResponse :one
INSERT INTO consumer_responses (
    proposal_id,
    consumer_team_id,
    response_token_hash,
    response_type,
    rejection_reason,
    migration_days_requested,
    migration_notes,
    submitted_via_link,
    token_expires_at,
    responded_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetConsumerResponse :one
SELECT * FROM consumer_responses
WHERE id = $1;

-- name: GetConsumerResponseByProposalAndTeam :one
SELECT * FROM consumer_responses
WHERE proposal_id = $1
  AND consumer_team_id = $2;

-- name: GetConsumerResponseByTokenHash :one
SELECT * FROM consumer_responses
WHERE response_token_hash = $1;

-- name: ListConsumerResponsesByProposal :many
SELECT * FROM consumer_responses
WHERE proposal_id = $1
ORDER BY created_at;

-- name: ListConsumerResponsesByTeam :many
SELECT * FROM consumer_responses
WHERE consumer_team_id = $1
ORDER BY created_at DESC;

-- name: UpdateConsumerResponse :one
UPDATE consumer_responses
SET
    response_token_hash = $2,
    response_type = $3,
    rejection_reason = $4,
    migration_days_requested = $5,
    migration_notes = $6,
    submitted_via_link = $7,
    token_expires_at = $8,
    responded_at = $9
WHERE id = $1
RETURNING *;
