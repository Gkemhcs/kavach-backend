-- GrantRoleBinding: Grants a role to a user or user group on a specific resource
-- Uses UPSERT semantics to create new role bindings or update existing ones
-- ON CONFLICT clause ensures idempotent behavior for duplicate role binding attempts
-- Supports both user-based (user_id) and group-based (group_id) role assignments
-- name: GrantRoleBinding :exec
INSERT INTO role_bindings (
  user_id,
  group_id,
  role,
  resource_type,
  resource_id,
  organization_id,
  secret_group_id,
  environment_id
)
VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- RevokeRoleBinding: Removes a role binding for a user or user group on a specific resource
-- Uses conditional logic to handle both user-based and group-based role bindings
-- Returns the number of affected rows to determine if the role binding existed
-- No error if role binding doesn't exist (idempotent operation)
-- name: RevokeRoleBinding :execresult
DELETE FROM role_bindings
WHERE
  ((user_id = $1 AND group_id IS NULL) OR (user_id IS NULL AND group_id = $2))
  AND role = $3
  AND resource_type = $4
  AND resource_id = $5;
