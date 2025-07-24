-- CreateRoleBinding: Creates a new role binding for a user on a specific resource
-- Used internally for creating role bindings with explicit user IDs and resource references
-- Returns the complete role binding record including generated ID and timestamps
-- name: CreateRoleBinding :one
INSERT INTO role_bindings (
  user_id,
  role,
  resource_type,
  resource_id,
  organization_id,
  secret_group_id,
  environment_id
)
VALUES (
  $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- GetRoleBinding: Retrieves a specific role binding by its composite key
-- Used for validation and lookup operations when all binding parameters are known
-- Returns complete role binding record if found
-- name: GetRoleBinding :one
SELECT * FROM role_bindings
WHERE user_id = $1
  AND role = $2
  AND resource_type = $3
  AND resource_id = $4;

-- UpdateUserRole: Updates the role level for an existing role binding
-- Changes the role while preserving the binding relationship and updating the timestamp
-- name: UpdateUserRole :exec
UPDATE role_bindings
SET role = $5,
    updated_at = now()
WHERE user_id = $1
  AND resource_type = $2
  AND resource_id = $3
  AND role = $4;

-- DeleteRoleBinding: Removes a specific role binding from the system
-- Deletes the exact role binding identified by the composite key
-- name: DeleteRoleBinding :exec
DELETE FROM role_bindings
WHERE user_id = $1
  AND role = $2
  AND resource_type = $3
  AND resource_id = $4;

-- ListAccessibleOrganizations: Retrieves all organizations that a user has access to
-- Joins with organizations table to get organization details along with user's role
-- Filters for organization-level permissions (no secret_group_id or environment_id)
-- name: ListAccessibleOrganizations :many
SELECT 
  rb.organization_id AS id,
  o.name as org_name,
  rb.role AS role      
FROM role_bindings AS rb 
INNER JOIN organizations AS o ON rb.organization_id = o.id
WHERE 
  rb.user_id = $1 
  AND rb.environment_id IS NULL 
  AND rb.secret_group_id IS NULL;

-- ListAccessibleSecretGroups: Retrieves all secret groups within an organization that a user has access to
-- Joins with secret_groups and organizations tables to get group and org details
-- Filters for secret group-level permissions within the specified organization
-- name: ListAccessibleSecretGroups :many
SELECT 
  rb.secret_group_id AS id,
  sg.name as name,
  o.name AS organization_name,
  rb.role AS role      
FROM role_bindings AS rb 
INNER JOIN secret_groups AS sg ON rb.secret_group_id = sg.id
INNER JOIN organizations AS o ON rb.organization_id = o.id
WHERE 
  rb.user_id = $1 
  AND rb.environment_id IS NULL 
  AND rb.organization_id = $2;

-- ListAccessibleEnvironments: Retrieves all environments within a secret group that a user has access to
-- Joins with environments and secret_groups tables to get environment and group details
-- Filters for environment-level permissions within the specified secret group
-- name: ListAccessibleEnvironments :many
SELECT 
  rb.environment_id AS id,
  e.name,
  sg.name AS secret_group_name,
  rb.role AS role        
FROM role_bindings AS rb 
INNER JOIN environments AS e ON rb.environment_id = e.id
INNER JOIN secret_groups AS sg ON rb.secret_group_id = sg.id
WHERE 
  rb.user_id = $1 
  AND rb.organization_id = $2
  AND rb.secret_group_id = $3;
