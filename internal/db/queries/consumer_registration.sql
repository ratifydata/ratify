-- name: CreateConsumerRegistration :one
INSERT INTO consumer_registrations (
    contract_id,
    consumer_team_id,
    approved_by,
    status,
    usage_description,
    approved_at
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetConsumerRegistration :one
SELECT * FROM consumer_registrations
WHERE id = $1;

-- name: GetConsumerRegistrationByContractAndTeam :one
SELECT * FROM consumer_registrations
WHERE contract_id = $1
  AND consumer_team_id = $2;

-- name: ListConsumerRegistrationsByContract :many
SELECT * FROM consumer_registrations
WHERE contract_id = $1
ORDER BY registered_at DESC;

-- name: ListConsumerRegistrationsByTeam :many
SELECT * FROM consumer_registrations
WHERE consumer_team_id = $1
ORDER BY registered_at DESC;

-- name: UpdateConsumerRegistration :one
UPDATE consumer_registrations
SET
    approved_by = $2,
    status = $3,
    usage_description = $4,
    approved_at = $5
WHERE id = $1
RETURNING *;

-- name: DeleteConsumerRegistration :exec
DELETE FROM consumer_registrations
WHERE id = $1;
