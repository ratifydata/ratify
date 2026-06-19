-- name: CreateContract :one
INSERT INTO contracts (
    org_id,
    connection_id,
    producer_team_id,
    display_name,
    schema_name,
    table_name,
    description,
    status,
    freshness_sla_hours
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetContract :one
SELECT * FROM contracts
WHERE id = $1;

-- name: GetActiveContractForTable :one
SELECT * FROM contracts
WHERE connection_id = $1
  AND schema_name = $2
  AND table_name = $3
  AND status = 'active';

-- name: ListContractsByOrg :many
SELECT * FROM contracts
WHERE org_id = $1
ORDER BY created_at DESC;

-- name: ListContractsByConnection :many
SELECT * FROM contracts
WHERE connection_id = $1
ORDER BY schema_name, table_name, created_at DESC;

-- name: ListContractsByProducerTeam :many
SELECT * FROM contracts
WHERE producer_team_id = $1
ORDER BY created_at DESC;

-- name: UpdateContract :one
UPDATE contracts
SET
    producer_team_id = $2,
    display_name = $3,
    schema_name = $4,
    table_name = $5,
    description = $6,
    status = $7,
    current_version = $8,
    freshness_sla_hours = $9,
    activated_at = $10,
    deprecated_at = $11,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

