-- CreateGroup: Creates a new user group within an organization
-- Returns the complete group record including generated ID and timestamps
-- Enforces unique constraint on (organization_id, name) combination
-- name: CreateGroup :one
INSERT INTO user_groups (organization_id, name, description)
VALUES ($1, $2, $3)
RETURNING *;

-- DeleteGroup: Removes a user group from an organization
-- Only deletes if the group exists and belongs to the specified organization
-- Cascading deletes will remove all group memberships automatically
-- name: DeleteGroup :exec
DELETE FROM user_groups
WHERE id = $1 AND organization_id = $2;

-- ListGroupsByOrg: Retrieves all user groups within an organization
-- Returns minimal fields needed for listing: id, name, description, created_at
-- Ordered by creation date (newest first) for consistent pagination
-- name: ListGroupsByOrg :many
SELECT id, name, description, created_at
FROM user_groups
WHERE organization_id = $1
ORDER BY created_at DESC;

-- GetGroupByName: Retrieves a specific user group by name within an organization
-- Used for validation and lookup operations when group name is known
-- Returns complete group record including all metadata fields
-- name: GetGroupByName :one
SELECT * FROM user_groups
WHERE name = $1 AND organization_id = $2;

-- AddGroupMember: Creates a membership relationship between a user and a user group
-- Enforces unique constraint on (user_group_id, user_id) to prevent duplicate memberships
-- Returns the created membership record for confirmation
-- name: AddGroupMember :exec
INSERT INTO user_group_members (user_group_id, user_id)
VALUES ($1, $2)
RETURNING *;

-- RemoveGroupMember: Removes a user's membership from a user group
-- Returns the number of affected rows to determine if membership existed
-- No error if membership doesn't exist (idempotent operation)
-- name: RemoveGroupMember :execresult
DELETE FROM user_group_members
WHERE user_group_id = $1 AND user_id = $2;

-- ListGroupMembers: Retrieves all members of a specific user group
-- Joins with users table to get member details: id, name, email, membership date
-- Ordered by membership creation date (newest first) for consistent display
-- name: ListGroupMembers :many
SELECT u.id, u.name, u.email, ugm.created_at
FROM user_group_members ugm
JOIN users u ON ugm.user_id = u.id
WHERE ugm.user_group_id = $1
ORDER BY ugm.created_at DESC;
