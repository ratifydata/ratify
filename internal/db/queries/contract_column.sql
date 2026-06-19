-- name: CreateContractColumn :one
INSERT INTO contract_columns (
    contract_version_id,
    column_name,
    data_type,
    is_nullable,
    is_primary_key,
    description,
    "constraints",
    ordinal_position
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetContractColumn :one
SELECT * FROM contract_columns
WHERE id = $1;

-- name: GetContractColumnByName :one
SELECT * FROM contract_columns
WHERE contract_version_id = $1
  AND column_name = $2;

-- name: ListContractColumnsByVersion :many
SELECT * FROM contract_columns
WHERE contract_version_id = $1
ORDER BY ordinal_position;

-- name: UpdateContractColumn :one
UPDATE contract_columns
SET
    column_name = $2,
    data_type = $3,
    is_nullable = $4,
    is_primary_key = $5,
    description = $6,
    "constraints" = $7,
    ordinal_position = $8
WHERE id = $1
RETURNING *;