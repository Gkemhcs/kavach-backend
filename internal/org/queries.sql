-- CreateOrganization inserts a new organization into the organizations table.
-- Used when a user creates a new organization.
-- name: CreateOrganization :one
INSERT INTO organizations ( name, description,owner_id)
VALUES ( $1, $2, $3)
RETURNING *;

-- GetOrganizationByID fetches an organization by its unique ID.
-- Used for organization detail views and internal lookups.
-- name: GetOrganizationByID :one
SELECT * FROM organizations WHERE id = $1;

-- UpdateOrganization updates the name and updated_at timestamp of an organization.
-- Used to rename organizations and track modification time.
-- name: UpdateOrganization :one
UPDATE organizations
SET name = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- DeleteOrganization removes an organization by its ID.
-- Used for organization deletion and cleanup.
-- name: DeleteOrganization :exec
DELETE FROM organizations WHERE id = $1;

-- ListOrganizationsByOwner returns all organizations for a given owner, ordered by creation time.
-- Used to display organizations within a user's context.
-- name: ListOrganizationsByOwner :many
SELECT * FROM organizations WHERE owner_id = $1 ORDER BY created_at DESC;

-- GetOrganizationByName fetches an organization by name and owner.
-- Used to ensure organization name uniqueness within a user's organizations and for lookups.
-- name: GetOrganizationByName :one
SELECT * FROM organizations WHERE name = $1 ;