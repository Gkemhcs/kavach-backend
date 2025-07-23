-- CreateSecretGroup inserts a new secret group into the secret_groups table.
-- Used when a user creates a new secret group within an organization.
-- name: CreateSecretGroup :one
INSERT INTO secret_groups ( name, organization_id,description)
VALUES ($1, $2, $3)
RETURNING *;

-- GetSecretGroupByID fetches a secret group by its unique ID.
-- Used for secret group detail views and internal lookups.
-- name: GetSecretGroupByID :one
SELECT * FROM secret_groups WHERE id = $1;

-- GetSecretGroupByName fetches a secret group by name and organization.
-- Used to ensure secret group name uniqueness within an organization and for lookups.
-- name: GetSecretGroupByName :one
SELECT * FROM secret_groups WHERE name = $1 and organization_id = $2 ;

-- UpdateSecretGroup updates the name and updated_at timestamp of a secret group.
-- Used to rename secret groups and track modification time.
-- name: UpdateSecretGroup :one
UPDATE secret_groups
SET name = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- DeleteSecretGroup removes a secret group by its ID.
-- Used for secret group deletion and cleanup.
-- name: DeleteSecretGroup :exec
DELETE FROM secret_groups WHERE id = $1;

-- ListSecretGroupsByOrg returns all secret groups for a given organization, ordered by creation time.
-- Used to display secret groups within an organization context.
-- name: ListSecretGroupsByOrg :many
SELECT * FROM secret_groups WHERE organization_id = $1 ORDER BY created_at DESC;
