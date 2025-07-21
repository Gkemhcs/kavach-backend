-- name: AddSecretGroupMember :exec
INSERT INTO secret_group_members (secret_group_id, user_id, role)
VALUES ($1, $2, $3);

-- name: RemoveSecretGroupMember :exec
DELETE FROM secret_group_members
WHERE secret_group_id = $1 AND user_id = $2;

-- name: GetSecretGroupMember :one
SELECT * FROM secret_group_members
WHERE secret_group_id = $1 AND user_id = $2;

-- name: ListSecretGroupMembers :many
SELECT * FROM secret_group_members
WHERE secret_group_id = $1;
