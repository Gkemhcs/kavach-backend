-- AddOrgMember adds a user as a member to an organization with a specific role.
-- Used to grant access and permissions to users for an organization.
-- name: AddOrgMember :exec
INSERT INTO org_members (org_id, user_id, role)
VALUES ($1, $2, $3);

-- RemoveOrgMember removes a user from an organization.
-- Used to revoke access and permissions for a user in an organization.
-- name: RemoveOrgMember :exec
DELETE FROM org_members
WHERE org_id = $1 AND user_id = $2;

-- GetOrgMember fetches a specific member of an organization by user ID.
-- Used to check membership and permissions for a user in an organization.
-- name: GetOrgMember :one
SELECT * FROM org_members
WHERE org_id = $1 AND user_id = $2;

-- ListOrgMembers returns all members of a given organization.
-- Used to display or manage all users with access to an organization.
-- name: ListOrgMembers :many
SELECT * FROM org_members
WHERE org_id = $1;
