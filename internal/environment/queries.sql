-- name: CreateEnvironment :one
INSERT INTO environments (name, secret_group_id)
VALUES ( $1, $2)
RETURNING *;

-- name: GetEnvironmentByID :one
SELECT * FROM environments WHERE id = $1;

-- name: UpdateEnvironment :one
UPDATE environments
SET name = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteEnvironment :exec
DELETE FROM environments WHERE id = $1;

-- name: ListEnvironmentsBySecretGroup :many
SELECT * FROM environments WHERE secret_group_id = $1 ORDER BY created_at DESC;
