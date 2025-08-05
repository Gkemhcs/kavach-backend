-- Ownership Transfer Queries for RBAC System
-- These queries handle ownership transfer when revoking role bindings

-- ============================================================================
-- 1. GET PARENT RESOURCE OWNER
-- ============================================================================

-- name: GetOrganizationOwner :one
-- Get the owner of an organization from role_bindings table
SELECT 
    rb.user_id,
    u.name as user_name,
    u.email as user_email
FROM role_bindings rb
INNER JOIN users u ON rb.user_id = u.id
WHERE rb.resource_type = 'organization'
  AND rb.resource_id = $1
  AND rb.role = 'owner'
  AND rb.group_id IS NULL
LIMIT 1;

-- name: GetSecretGroupOwner :one
-- Get the owner of a secret group from role_bindings table
SELECT 
    rb.user_id,
    u.name as user_name,
    u.email as user_email
FROM role_bindings rb
INNER JOIN users u ON rb.user_id = u.id
WHERE rb.resource_type = 'secret_group'
  AND rb.resource_id = $1
  AND rb.role = 'owner'
  AND rb.group_id IS NULL
LIMIT 1;

-- ============================================================================
-- 2. FIND CHILD RESOURCES WITH ROLE BINDINGS BY REVOKED USER/GROUP
-- ============================================================================

-- name: GetSecretGroupsWithUserRoleBindings :many
-- Find all secret groups where the revoked user has role bindings
SELECT 
    sg.id,
    sg.name,
    sg.description,
    sg.organization_id,
    sg.created_at,
    sg.updated_at
FROM secret_groups sg
INNER JOIN role_bindings rb ON sg.id = rb.resource_id
WHERE sg.organization_id = $1
  AND rb.resource_type = 'secret_group'
  AND rb.user_id = $2
  AND rb.group_id IS NULL;

-- name: GetSecretGroupsWithGroupRoleBindings :many
-- Find all secret groups where members of the revoked group have role bindings
SELECT DISTINCT
    sg.id,
    sg.name,
    sg.description,
    sg.organization_id,
    sg.created_at,
    sg.updated_at
FROM secret_groups sg
INNER JOIN role_bindings rb ON sg.id = rb.resource_id
INNER JOIN user_group_members ugm ON rb.user_id = ugm.user_id
WHERE sg.organization_id = $1
  AND rb.resource_type = 'secret_group'
  AND ugm.user_group_id = $2
  AND rb.group_id IS NULL;

-- name: GetEnvironmentsWithUserRoleBindings :many
-- Find all environments where the revoked user has role bindings
SELECT 
    e.id,
    e.name,
    e.description,
    e.secret_group_id,
    sg.organization_id,
    e.created_at,
    e.updated_at
FROM environments e
INNER JOIN secret_groups sg ON e.secret_group_id = sg.id
INNER JOIN role_bindings rb ON e.id = rb.resource_id
WHERE e.secret_group_id = $1
  AND rb.resource_type = 'environment'
  AND rb.user_id = $2
  AND rb.group_id IS NULL;

-- name: GetEnvironmentsWithGroupRoleBindings :many
-- Find all environments where members of the revoked group have role bindings
SELECT DISTINCT
    e.id,
    e.name,
    e.description,
    e.secret_group_id,
    sg.organization_id,
    e.created_at,
    e.updated_at
FROM environments e
INNER JOIN secret_groups sg ON e.secret_group_id = sg.id
INNER JOIN role_bindings rb ON e.id = rb.resource_id
INNER JOIN user_group_members ugm ON rb.user_id = ugm.user_id
WHERE e.secret_group_id = $1
  AND rb.resource_type = 'environment'
  AND ugm.user_group_id = $2
  AND rb.group_id IS NULL;

-- ============================================================================
-- 3. TRANSFER OWNERSHIP BY UPDATING ROLE BINDINGS
-- ============================================================================

-- name: TransferSecretGroupRoleBindingOwnership :exec
-- Transfer ownership of a secret group by updating the role binding
UPDATE role_bindings
SET user_id = $2,
    updated_at = now()
WHERE resource_type = 'secret_group'
  AND resource_id = $1
  AND role = 'owner';

-- name: TransferEnvironmentRoleBindingOwnership :exec
-- Transfer ownership of an environment by updating the role binding
UPDATE role_bindings
SET user_id = $2,
    updated_at = now()
WHERE resource_type = 'environment'
  AND resource_id = $1
  AND role = 'owner';

-- ============================================================================
-- 4. BATCH OWNERSHIP TRANSFER OPERATIONS
-- ============================================================================

-- name: BatchTransferSecretGroupRoleBindingOwnership :exec
-- Transfer ownership of multiple secret groups by updating role bindings
UPDATE role_bindings
SET user_id = $2,
    updated_at = now()
WHERE resource_type = 'secret_group'
  AND resource_id = ANY($1::uuid[])
  AND role = 'owner';

-- name: BatchTransferEnvironmentRoleBindingOwnership :exec
-- Transfer ownership of multiple environments by updating role bindings
UPDATE role_bindings
SET user_id = $2,
    updated_at = now()
WHERE resource_type = 'environment'
  AND resource_id = ANY($1::uuid[])
  AND role = 'owner';

-- ============================================================================
-- 5. CREATE OWNERSHIP ROLE BINDINGS IF THEY DON'T EXIST
-- ============================================================================

-- name: CreateSecretGroupOwnershipRoleBinding :exec
-- Create an ownership role binding for a secret group if it doesn't exist
INSERT INTO role_bindings (user_id, role, resource_type, resource_id, organization_id, secret_group_id)
SELECT $2, 'owner', 'secret_group', $1, sg.organization_id, $1
FROM secret_groups sg
WHERE sg.id = $1
  AND NOT EXISTS (
    SELECT 1 FROM role_bindings rb 
    WHERE rb.resource_type = 'secret_group' 
      AND rb.resource_id = $1 
      AND rb.role = 'owner'
  );

-- name: CreateEnvironmentOwnershipRoleBinding :exec
-- Create an ownership role binding for an environment if it doesn't exist
INSERT INTO role_bindings (user_id, role, resource_type, resource_id, organization_id, secret_group_id, environment_id)
SELECT $2, 'owner', 'environment', $1, sg.organization_id, e.secret_group_id, $1
FROM environments e
INNER JOIN secret_groups sg ON e.secret_group_id = sg.id
WHERE e.id = $1
  AND NOT EXISTS (
    SELECT 1 FROM role_bindings rb 
    WHERE rb.resource_type = 'environment' 
      AND rb.resource_id = $1 
      AND rb.role = 'owner'
  );

-- ============================================================================
-- 6. VALIDATION QUERIES
-- ============================================================================

-- name: ValidateResourceOwnership :one
-- Validate that a resource exists and has an owner
SELECT 
    CASE 
        WHEN EXISTS (
            SELECT 1 FROM role_bindings 
            WHERE resource_type = $1 
              AND resource_id = $2 
              AND role = 'owner'
        ) THEN true
        ELSE false
    END as has_owner;

-- name: GetResourceRoleBindings :many
-- Get all role bindings for a specific resource
SELECT 
    rb.user_id,
    rb.role,
    rb.resource_type,
    rb.resource_id,
    u.name as user_name,
    u.email as user_email
FROM role_bindings rb
INNER JOIN users u ON rb.user_id = u.id
WHERE rb.resource_type = $1
  AND rb.resource_id = $2
ORDER BY rb.role DESC, u.name; 