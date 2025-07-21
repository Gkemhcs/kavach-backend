-- name: AddEnvironmentMember :exec
INSERT INTO environment_members (environment_id, user_id, role)
VALUES ($1, $2, $3);

-- name: RemoveEnvironmentMember :exec
DELETE FROM environment_members
WHERE environment_id = $1 AND user_id = $2;

-- name: GetEnvironmentMember :one
SELECT * FROM environment_members
WHERE environment_id = $1 AND user_id = $2;

-- name: ListEnvironmentMembers :many
SELECT * FROM environment_members
WHERE environment_id = $1;
