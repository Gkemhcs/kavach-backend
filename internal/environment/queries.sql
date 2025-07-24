-- CreateEnvironment inserts a new environment into the environments table.
-- Used when a user creates a new environment within a secret group.
-- name: CreateEnvironment :one
INSERT INTO environments (name, secret_group_id,description)
VALUES ( $1, $2, $3)
RETURNING *;

-- GetEnvironmentByID fetches an environment by its unique ID.
-- Used for environment detail views and internal lookups.
-- name: GetEnvironmentByID :one
SELECT * FROM environments WHERE id = $1;

-- GetEnvironmentByName fetches an environment by name and secret group.
-- Used to ensure environment name uniqueness within a group and for lookups.
-- name: GetEnvironmentByName :one
SELECT 
    e.id AS id,
    e.name AS name,
    e.secret_group_id AS secret_group_id,
    sg.organization_id AS organization_id,
    e.created_at AS created_at,
    e.updated_at AS updated_at,
    e.description AS description
FROM environments e
INNER JOIN secret_groups sg ON e.secret_group_id = sg.id
WHERE  e.secret_group_id = $1 and e.name = $2;


-- UpdateEnvironment updates the name and updated_at timestamp of an environment.
-- Used to rename environments and track modification time.
-- name: UpdateEnvironment :one
UPDATE environments
SET name = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- DeleteEnvironment removes an environment by its ID.
-- Used for environment deletion and cleanup.
-- name: DeleteEnvironment :exec
DELETE FROM environments WHERE id = $1;

-- ListEnvironmentsBySecretGroup returns all environments for a given secret group, ordered by creation time.
-- Used to display environments within a group context.
-- name: ListEnvironmentsBySecretGroup :many
SELECT * FROM environments WHERE secret_group_id = $1 ORDER BY created_at DESC;
