-- AddSecretGroupMember adds a user as a member to a secret group with a specific role.
-- Used to grant access and permissions to users for a secret group.
-- name: AddSecretGroupMember :exec
INSERT INTO secret_group_members (secret_group_id, user_id, role)
VALUES ($1, $2, $3);

-- RemoveSecretGroupMember removes a user from a secret group.
-- Used to revoke access and permissions for a user in a secret group.
-- name: RemoveSecretGroupMember :exec
DELETE FROM secret_group_members
WHERE secret_group_id = $1 AND user_id = $2;

-- GetSecretGroupMember fetches a specific member of a secret group by user ID.
-- Used to check membership and permissions for a user in a secret group.
-- name: GetSecretGroupMember :one
SELECT * FROM secret_group_members
WHERE secret_group_id = $1 AND user_id = $2;

-- ListSecretGroupMembers returns all members of a given secret group.
-- Used to display or manage all users with access to a secret group.
-- name: ListSecretGroupMembers :many
SELECT * FROM secret_group_members
WHERE secret_group_id = $1;
