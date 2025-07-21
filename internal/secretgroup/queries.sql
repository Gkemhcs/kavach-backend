-- name: CreateSecretGroup :one
INSERT INTO secret_groups ( name, organization_id)
VALUES ($1, $2)
RETURNING *;

-- name: GetSecretGroupByID :one
SELECT * FROM secret_groups WHERE id = $1;

-- name: UpdateSecretGroup :one
UPDATE secret_groups
SET name = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteSecretGroup :exec
DELETE FROM secret_groups WHERE id = $1;

-- name: ListSecretGroupsByOrg :many
SELECT * FROM secret_groups WHERE organization_id = $1 ORDER BY created_at DESC;
