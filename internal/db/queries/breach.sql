-- name: CreateBreach :one
INSERT INTO breaches (
    contract_id,
    breach_type,
    status,
    affected_element,
    expected_value,
    actual_value,
    acknowledged_at,
    resolved_at,
    acknowledged_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetBreach :one
SELECT * FROM breaches
WHERE id = $1;

-- name: ListBreachesByContract :many
SELECT * FROM breaches
WHERE contract_id = $1
ORDER BY detected_at DESC;

-- name: ListBreachesByStatus :many
SELECT * FROM breaches
WHERE status = $1
ORDER BY detected_at DESC;

-- name: ListBreachesByContractAndStatus :many
SELECT * FROM breaches
WHERE contract_id = $1
  AND status = $2
ORDER BY detected_at DESC;

-- name: UpdateBreach :one
UPDATE breaches
SET
    status = $2,
    affected_element = $3,
    expected_value = $4,
    actual_value = $5,
    acknowledged_at = $6,
    resolved_at = $7,
    acknowledged_by = $8
WHERE id = $1
RETURNING *;

-- name: DeleteBreach :exec
DELETE FROM breaches
WHERE id = $1;
