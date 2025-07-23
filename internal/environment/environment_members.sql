-- AddEnvironmentMember adds a user as a member to an environment with a specific role.
-- Used to grant access and permissions to users for an environment.
-- name: AddEnvironmentMember :exec
INSERT INTO environment_members (environment_id, user_id, role)
VALUES ($1, $2, $3);

-- RemoveEnvironmentMember removes a user from an environment.
-- Used to revoke access and permissions for a user in an environment.
-- name: RemoveEnvironmentMember :exec
DELETE FROM environment_members
WHERE environment_id = $1 AND user_id = $2;

-- GetEnvironmentMember fetches a specific member of an environment by user ID.
-- Used to check membership and permissions for a user in an environment.
-- name: GetEnvironmentMember :one
SELECT * FROM environment_members
WHERE environment_id = $1 AND user_id = $2;

-- ListEnvironmentMembers returns all members of a given environment.
-- Used to display or manage all users with access to an environment.
-- name: ListEnvironmentMembers :many
SELECT * FROM environment_members
WHERE environment_id = $1;
