-- name: CreateContractVersion :one
INSERT INTO contract_versions (
    contract_id,
    created_by,
    version_number,
    schema_snapshot,
    change_summary
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetContractVersion :one
SELECT * FROM contract_versions
WHERE id = $1;

-- name: GetContractVersionByNumber :one
SELECT * FROM contract_versions
WHERE contract_id = $1
  AND version_number = $2;

-- name: ListContractVersions :many
SELECT * FROM contract_versions
WHERE contract_id = $1
ORDER BY version_number DESC;

-- name: ListContractVersionsByCreator :many
SELECT * FROM contract_versions
WHERE created_by = $1
ORDER BY created_at DESC;
