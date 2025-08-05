-- Enhanced RBAC Queries for Hierarchical Access Control
-- Handles: User + Group access, Parent-child inheritance, Role conflict resolution

-- Get user's effective role for a specific resource
-- Considers: Direct user access, Group membership, Parent resource inheritance
-- name: GetUserEffectiveRole :one
WITH user_direct_roles AS (
    -- Direct user role bindings
    SELECT role, resource_type, resource_id
    FROM role_bindings rb1
    WHERE rb1.user_id = $1 
      AND rb1.resource_type = $2 
      AND rb1.resource_id = $3
      AND rb1.group_id IS NULL
),
user_group_roles AS (
    -- User's role through group membership
    SELECT rb.role, rb.resource_type, rb.resource_id
    FROM role_bindings rb
    INNER JOIN user_group_members ugm ON rb.group_id = ugm.user_group_id
    WHERE ugm.user_id = $1 
      AND rb.resource_type = $2 
      AND rb.resource_id = $3
      AND rb.user_id IS NULL
),
parent_inherited_roles AS (
    -- Inherited roles from parent resources
    SELECT 
        CASE 
            WHEN $2 = 'secret_group' THEN
                (SELECT get_highest_role(ARRAY_AGG(role)) 
                 FROM role_bindings rb6
                 WHERE rb6.user_id = $1 
                   AND rb6.resource_type = 'organization' 
                   AND rb6.resource_id = (SELECT organization_id FROM secret_groups WHERE id = $3))
            WHEN $2 = 'environment' THEN
                (SELECT get_highest_role(ARRAY_AGG(role)) 
                 FROM role_bindings rb7
                 WHERE rb7.user_id = $1 
                   AND rb7.resource_type = 'secret_group' 
                   AND rb7.resource_id = (SELECT secret_group_id FROM environments WHERE id = $3))
            ELSE NULL
        END as inherited_role
),
all_roles AS (
    SELECT role FROM user_direct_roles
    UNION ALL
    SELECT role FROM user_group_roles
    UNION ALL
    SELECT inherited_role FROM parent_inherited_roles WHERE inherited_role IS NOT NULL
)
SELECT get_highest_role(ARRAY_AGG(role)) as effective_role
FROM all_roles;

-- Enhanced ListAccessibleOrganizations with hierarchical RBAC
-- name: ListAccessibleOrganizationsEnhanced :many
WITH user_direct_org_roles AS (
    -- Direct user access to organizations
    SELECT 
        rb.organization_id,
        rb.role,
        'direct' as access_type
    FROM role_bindings rb
    WHERE rb.user_id = $1 
      AND rb.resource_type = 'organization'
      AND rb.group_id IS NULL
),
user_group_org_roles AS (
    -- User access through group membership
    SELECT 
        rb.organization_id,
        rb.role,
        'group' as access_type
    FROM role_bindings rb
    INNER JOIN user_group_members ugm ON rb.group_id = ugm.user_group_id
    WHERE ugm.user_id = $1 
      AND rb.resource_type = 'organization'
      AND rb.user_id IS NULL
),
combined_org_roles AS (
    SELECT organization_id, role, access_type FROM user_direct_org_roles
    UNION ALL
    SELECT organization_id, role, access_type FROM user_group_org_roles
),
effective_org_roles AS (
    SELECT 
        organization_id,
        get_highest_role(ARRAY_AGG(role)) as effective_role
    FROM combined_org_roles
    GROUP BY organization_id
)
SELECT 
    o.id,
    o.name as org_name,
    COALESCE(eor.effective_role, 'viewer') as role
FROM organizations o
LEFT JOIN effective_org_roles eor ON o.id = eor.organization_id
ORDER BY o.name;

-- Enhanced ListAccessibleSecretGroups with hierarchical RBAC
-- name: ListAccessibleSecretGroupsEnhanced :many
WITH user_direct_sg_roles AS (
    -- Direct user access to secret groups
    SELECT 
        rb.secret_group_id,
        rb.role,
        'direct' as access_type
    FROM role_bindings rb
    WHERE rb.user_id = $1 
      AND rb.resource_type = 'secret_group'
      AND rb.organization_id = $2
      AND rb.group_id IS NULL
),
user_group_sg_roles AS (
    -- User access through group membership
    SELECT 
        rb.secret_group_id,
        rb.role,
        'group' as access_type
    FROM role_bindings rb
    INNER JOIN user_group_members ugm ON rb.group_id = ugm.user_group_id
    WHERE ugm.user_id = $1 
      AND rb.resource_type = 'secret_group'
      AND rb.organization_id = $2
      AND rb.user_id IS NULL
),
inherited_sg_roles AS (
    -- Inherited access from organization level
    SELECT 
        sg.id as secret_group_id,
        eor.effective_role as role,
        'inherited' as access_type
    FROM secret_groups sg
    INNER JOIN (
        SELECT 
            organization_id,
            get_highest_role(ARRAY_AGG(role)) as effective_role
        FROM (
            SELECT organization_id, role FROM role_bindings rb4
            WHERE rb4.user_id = $1 AND rb4.resource_type = 'organization' AND rb4.group_id IS NULL
            UNION ALL
            SELECT rb.organization_id, rb.role FROM role_bindings rb
            INNER JOIN user_group_members ugm ON rb.group_id = ugm.user_group_id
            WHERE ugm.user_id = $1 AND rb.resource_type = 'organization' AND rb.user_id IS NULL
        ) org_roles
        GROUP BY organization_id
    ) eor ON sg.organization_id = eor.organization_id
    WHERE sg.organization_id = $2
),
combined_sg_roles AS (
    SELECT secret_group_id, role, access_type FROM user_direct_sg_roles
    UNION ALL
    SELECT secret_group_id, role, access_type FROM user_group_sg_roles
    UNION ALL
    SELECT secret_group_id, role, access_type FROM inherited_sg_roles
),
ranked_sg_roles AS (
    SELECT 
        csr.secret_group_id,
        csr.role,
        csr.access_type,
        ROW_NUMBER() OVER (
            PARTITION BY csr.secret_group_id 
            ORDER BY 
                CASE csr.role
                    WHEN 'owner' THEN 1
                    WHEN 'admin' THEN 2
                    WHEN 'editor' THEN 3
                    WHEN 'viewer' THEN 4
                    ELSE 5
                END,
                CASE csr.access_type
                    WHEN 'direct' THEN 1
                    WHEN 'group' THEN 2
                    WHEN 'inherited' THEN 3
                    ELSE 4
                END
        ) as rn
    FROM combined_sg_roles csr
)
SELECT 
    sg.id,
    sg.name,
    o.name AS organization_name,
    rsr.role as role,
    CASE 
        WHEN rsr.access_type = 'direct' THEN 'secret_group'
        WHEN rsr.access_type = 'group' THEN 'secret_group'
        WHEN rsr.access_type = 'inherited' THEN 'organization'
        ELSE 'none'
    END as inherited_from
FROM secret_groups sg
INNER JOIN organizations o ON sg.organization_id = o.id
INNER JOIN ranked_sg_roles rsr ON sg.id = rsr.secret_group_id AND rsr.rn = 1
WHERE sg.organization_id = $2
ORDER BY sg.name;

-- Enhanced ListAccessibleEnvironments with hierarchical RBAC
-- name: ListAccessibleEnvironmentsEnhanced :many
-- secret_group_id: uuid
WITH user_direct_env_roles AS (
    -- Direct user access to environments
    SELECT 
        rb.environment_id,
        rb.role,
        'direct' as access_type
    FROM role_bindings rb
    WHERE rb.user_id = $1 
      AND rb.resource_type = 'environment'
      AND rb.organization_id = $2
      AND rb.secret_group_id = @secret_group_id
      AND rb.group_id IS NULL
),
user_group_env_roles AS (
    -- User access through group membership
    SELECT 
        rb.environment_id,
        rb.role,
        'group' as access_type
    FROM role_bindings rb
    INNER JOIN user_group_members ugm ON rb.group_id = ugm.user_group_id
    WHERE ugm.user_id = $1 
      AND rb.resource_type = 'environment'
      AND rb.organization_id = $2
      AND rb.secret_group_id = @secret_group_id
      AND rb.user_id IS NULL
),
inherited_env_roles AS (
    -- Inherited access from secret group level
    SELECT 
        e.id as environment_id,
        sg_effective.effective_role as role,
        'inherited' as access_type
    FROM environments e
    INNER JOIN (
        SELECT 
            @secret_group_id as secret_group_id,
            get_highest_role(ARRAY_AGG(role)) as effective_role
        FROM (
            SELECT role FROM role_bindings rb2
            WHERE rb2.user_id = $1 AND rb2.resource_type = 'secret_group' AND rb2.resource_id = @secret_group_id AND rb2.group_id IS NULL
            UNION ALL
            SELECT rb.role FROM role_bindings rb
            INNER JOIN user_group_members ugm ON rb.group_id = ugm.user_group_id
            WHERE ugm.user_id = $1 AND rb.resource_type = 'secret_group' AND rb.resource_id = @secret_group_id AND rb.user_id IS NULL
        ) sg_roles
        HAVING COUNT(*) > 0
    ) sg_effective ON e.secret_group_id = sg_effective.secret_group_id
    WHERE e.secret_group_id = @secret_group_id
),
org_inherited_env_roles AS (
    -- Inherited access from organization level
    SELECT 
        e.id as environment_id,
        org_effective.effective_role as role,
        'org_inherited' as access_type
    FROM environments e
    INNER JOIN secret_groups sg ON e.secret_group_id = sg.id
    INNER JOIN (
        SELECT 
            sg.organization_id,
            get_highest_role(ARRAY_AGG(role)) as effective_role
        FROM secret_groups sg
        INNER JOIN role_bindings rb3 ON rb3.resource_type = 'organization' AND rb3.resource_id = sg.organization_id AND rb3.group_id IS NULL
        WHERE rb3.user_id = $1 AND sg.id = @secret_group_id
        GROUP BY sg.organization_id
        HAVING COUNT(*) > 0
        UNION ALL
        SELECT 
            sg.organization_id,
            get_highest_role(ARRAY_AGG(role)) as effective_role
        FROM secret_groups sg
        INNER JOIN role_bindings rb ON rb.resource_type = 'organization' AND rb.resource_id = sg.organization_id AND rb.user_id IS NULL
        INNER JOIN user_group_members ugm ON rb.group_id = ugm.user_group_id
        WHERE ugm.user_id = $1 AND sg.id = @secret_group_id
        GROUP BY sg.organization_id
        HAVING COUNT(*) > 0

    ) org_effective ON sg.organization_id = org_effective.organization_id
    WHERE e.secret_group_id = @secret_group_id
),
combined_env_roles AS (
    SELECT environment_id, role, access_type FROM user_direct_env_roles
    UNION ALL
    SELECT environment_id, role, access_type FROM user_group_env_roles
    UNION ALL
    SELECT environment_id, role, access_type FROM inherited_env_roles
    UNION ALL
    SELECT environment_id, role, access_type FROM org_inherited_env_roles
),
ranked_env_roles AS (
    SELECT 
        cer.environment_id,
        cer.role,
        cer.access_type,
        ROW_NUMBER() OVER (
            PARTITION BY cer.environment_id 
            ORDER BY 
                CASE cer.role
                    WHEN 'owner' THEN 1
                    WHEN 'admin' THEN 2
                    WHEN 'editor' THEN 3
                    WHEN 'viewer' THEN 4
                    ELSE 5
                END,
                CASE cer.access_type
                    WHEN 'direct' THEN 1
                    WHEN 'group' THEN 2
                    WHEN 'inherited' THEN 3
                    WHEN 'org_inherited' THEN 4
                    ELSE 5
                END
        ) as rn
    FROM combined_env_roles cer
)
SELECT 
    e.id,
    e.name,
    sg.name AS secret_group_name,
    rer.role as role,
    CASE 
        WHEN rer.access_type = 'direct' THEN 'environment'
        WHEN rer.access_type = 'group' THEN 'environment'
        WHEN rer.access_type = 'inherited' THEN 'secret_group'
        WHEN rer.access_type = 'org_inherited' THEN 'organization'
        ELSE 'none'
    END as inherited_from
FROM environments e
INNER JOIN secret_groups sg ON e.secret_group_id = sg.id
INNER JOIN ranked_env_roles rer ON e.id = rer.environment_id AND rer.rn = 1
WHERE e.secret_group_id = @secret_group_id
ORDER BY e.name;

-- Check if user has specific permission on resource
-- name: CheckUserPermission :one
WITH user_permissions AS (
    -- Direct user permissions
    SELECT role FROM role_bindings rb5
    WHERE rb5.user_id = $1 
      AND rb5.resource_type = $2 
      AND rb5.resource_id = $3
      AND rb5.group_id IS NULL
    UNION ALL
    -- Group permissions
    SELECT rb.role FROM role_bindings rb
    INNER JOIN user_group_members ugm ON rb.group_id = ugm.user_group_id
    WHERE ugm.user_id = $1 
      AND rb.resource_type = $2 
      AND rb.resource_id = $3
      AND rb.user_id IS NULL
    UNION ALL
    -- Inherited permissions from parent resources
    SELECT 
        CASE 
            WHEN $2 = 'secret_group' THEN
                (SELECT get_highest_role(ARRAY_AGG(role)) 
                 FROM role_bindings rb8
                 WHERE rb8.user_id = $1 
                   AND rb8.resource_type = 'organization' 
                   AND rb8.resource_id = (SELECT organization_id FROM secret_groups WHERE id = $3))
            WHEN $2 = 'environment' THEN
                (SELECT get_highest_role(ARRAY_AGG(role)) 
                 FROM role_bindings rb9
                 WHERE rb9.user_id = $1 
                   AND rb9.resource_type = 'secret_group' 
                   AND rb9.resource_id = (SELECT secret_group_id FROM environments WHERE id = $3))
            ELSE NULL
        END
)
SELECT 
    CASE 
        WHEN $4 = 'owner' THEN get_highest_role(ARRAY_AGG(role)) = 'owner'
        WHEN $4 = 'admin' THEN get_highest_role(ARRAY_AGG(role)) IN ('owner', 'admin')
        WHEN $4 = 'editor' THEN get_highest_role(ARRAY_AGG(role)) IN ('owner', 'admin', 'editor')
        WHEN $4 = 'viewer' THEN get_highest_role(ARRAY_AGG(role)) IN ('owner', 'admin', 'editor', 'viewer')
        ELSE false
    END as has_permission
FROM user_permissions;

-- List all role bindings for an organization with resolved names
-- name: ListOrganizationRoleBindings :many
WITH direct_org_bindings AS (
    -- Direct user bindings
    SELECT 
        rb.organization_id,
        rb.role,
        'direct' as binding_type,
        'user' as entity_type,
        rb.user_id::text as entity_id,
        COALESCE(NULLIF(u.name, ''), u.email, u.provider_id) as entity_name,
        NULL::uuid as group_id,
        '' as group_name
    FROM role_bindings rb
    INNER JOIN users u ON rb.user_id = u.id
    WHERE rb.resource_type = 'organization' 
      AND rb.organization_id = $1
      AND rb.group_id IS NULL
      AND rb.secret_group_id IS NULL
      AND rb.environment_id IS NULL
),
group_org_bindings AS (
    -- Group bindings
    SELECT 
        rb.organization_id,
        rb.role,
        'direct' as binding_type,
        'group' as entity_type,
        '' as entity_id,
        '' as entity_name,
        rb.group_id,
        ug.name as group_name
    FROM role_bindings rb
    INNER JOIN user_groups ug ON rb.group_id = ug.id
    WHERE rb.resource_type = 'organization' 
      AND rb.organization_id = $1
      AND rb.user_id IS NULL
      AND rb.secret_group_id IS NULL
      AND rb.environment_id IS NULL
)
SELECT 
    organization_id,
    role,
    binding_type,
    entity_type,
    entity_id,
    entity_name,
    group_id,
    group_name
FROM direct_org_bindings
UNION ALL
SELECT 
    organization_id,
    role,
    binding_type,
    entity_type,
    entity_id,
    entity_name,
    group_id,
    group_name
FROM group_org_bindings
ORDER BY role, entity_type, entity_name, group_name;

-- List all role bindings for a secret group with resolved names (including inherited)
-- name: ListSecretGroupRoleBindings :many
WITH direct_sg_bindings AS (
    -- Direct user bindings on secret group
    SELECT 
        rb.secret_group_id,
        rb.role,
        'direct' as binding_type,
        'user' as entity_type,
        rb.user_id::text as entity_id,
        COALESCE(NULLIF(u.name, ''), u.email, u.provider_id) as entity_name,
        NULL::uuid as group_id,
        '' as group_name,
        'secret_group' as source_type
    FROM role_bindings rb
    INNER JOIN users u ON rb.user_id = u.id
    WHERE rb.resource_type = 'secret_group' 
      AND rb.secret_group_id = $1
      AND rb.group_id IS NULL
),
group_sg_bindings AS (
    -- Group bindings on secret group
    SELECT 
        rb.secret_group_id,
        rb.role,
        'direct' as binding_type,
        'group' as entity_type,
        '' as entity_id,
        '' as entity_name,
        rb.group_id,
        ug.name as group_name,
        'secret_group' as source_type
    FROM role_bindings rb
    INNER JOIN user_groups ug ON rb.group_id = ug.id
    WHERE rb.resource_type = 'secret_group' 
      AND rb.secret_group_id = $1
      AND rb.user_id IS NULL
),
inherited_org_bindings AS (
    -- Inherited from organization
    SELECT 
        sg.id as secret_group_id,
        rb.role,
        'inherited' as binding_type,
        'user' as entity_type,
        rb.user_id::text as entity_id,
        COALESCE(NULLIF(u.name, ''), u.email, u.provider_id) as entity_name,
        NULL::uuid as group_id,
        '' as group_name,
        'organization' as source_type
    FROM secret_groups sg
    INNER JOIN role_bindings rb ON rb.resource_type = 'organization' AND rb.resource_id = sg.organization_id
    INNER JOIN users u ON rb.user_id = u.id
    WHERE sg.id = $1
      AND rb.group_id IS NULL
      AND rb.secret_group_id IS NULL
      AND rb.environment_id IS NULL
),
inherited_org_group_bindings AS (
    -- Inherited group bindings from organization
    SELECT 
        sg.id as secret_group_id,
        rb.role,
        'inherited' as binding_type,
        'group' as entity_type,
        '' as entity_id,
        '' as entity_name,
        rb.group_id,
        ug.name as group_name,
        'organization' as source_type
    FROM secret_groups sg
    INNER JOIN role_bindings rb ON rb.resource_type = 'organization' AND rb.resource_id = sg.organization_id
    INNER JOIN user_groups ug ON rb.group_id = ug.id
    WHERE sg.id = $1
      AND rb.user_id IS NULL
      AND rb.secret_group_id IS NULL
      AND rb.environment_id IS NULL
)
SELECT 
    secret_group_id,
    role,
    binding_type,
    entity_type,
    entity_id,
    entity_name,
    group_id,
    group_name,
    source_type
FROM direct_sg_bindings
UNION ALL
SELECT 
    secret_group_id,
    role,
    binding_type,
    entity_type,
    entity_id,
    entity_name,
    group_id,
    group_name,
    source_type
FROM group_sg_bindings
UNION ALL
SELECT 
    secret_group_id,
    role,
    binding_type,
    entity_type,
    entity_id,
    entity_name,
    group_id,
    group_name,
    source_type
FROM inherited_org_bindings
UNION ALL
SELECT 
    secret_group_id,
    role,
    binding_type,
    entity_type,
    entity_id,
    entity_name,
    group_id,
    group_name,
    source_type
FROM inherited_org_group_bindings
ORDER BY source_type, role, entity_type, entity_name, group_name;

-- List all role bindings for an environment with resolved names (including inherited)
-- name: ListEnvironmentRoleBindings :many
WITH direct_env_bindings AS (
    -- Direct user bindings on environment
    SELECT 
        rb.environment_id,
        rb.role,
        'direct' as binding_type,
        'user' as entity_type,
        rb.user_id::text as entity_id,
        COALESCE(NULLIF(u.name, ''), u.email, u.provider_id) as entity_name,
        NULL::uuid as group_id,
        '' as group_name,
        'environment' as source_type
    FROM role_bindings rb
    INNER JOIN users u ON rb.user_id = u.id
    WHERE rb.resource_type = 'environment' 
      AND rb.environment_id = $1
      AND rb.group_id IS NULL
),
group_env_bindings AS (
    -- Group bindings on environment
    SELECT 
        rb.environment_id,
        rb.role,
        'direct' as binding_type,
        'group' as entity_type,
        '' as entity_id,
        '' as entity_name,
        rb.group_id,
        ug.name as group_name,
        'environment' as source_type
    FROM role_bindings rb
    INNER JOIN user_groups ug ON rb.group_id = ug.id
    WHERE rb.resource_type = 'environment' 
      AND rb.environment_id = $1
      AND rb.user_id IS NULL
),
inherited_sg_bindings AS (
    -- Inherited from secret group
    SELECT 
        e.id as environment_id,
        rb.role,
        'inherited' as binding_type,
        'user' as entity_type,
        rb.user_id::text as entity_id,
        COALESCE(NULLIF(u.name, ''), u.email, u.provider_id) as entity_name,
        NULL::uuid as group_id,
        '' as group_name,
        'secret_group' as source_type
    FROM environments e
    INNER JOIN role_bindings rb ON rb.resource_type = 'secret_group' AND rb.resource_id = e.secret_group_id
    INNER JOIN users u ON rb.user_id = u.id
    WHERE e.id = $1
      AND rb.group_id IS NULL
),
inherited_sg_group_bindings AS (
    -- Inherited group bindings from secret group
    SELECT 
        e.id as environment_id,
        rb.role,
        'inherited' as binding_type,
        'group' as entity_type,
        '' as entity_id,
        '' as entity_name,
        rb.group_id,
        ug.name as group_name,
        'secret_group' as source_type
    FROM environments e
    INNER JOIN role_bindings rb ON rb.resource_type = 'secret_group' AND rb.resource_id = e.secret_group_id
    INNER JOIN user_groups ug ON rb.group_id = ug.id
    WHERE e.id = $1
      AND rb.user_id IS NULL
),
inherited_org_bindings AS (
    -- Inherited from organization
    SELECT 
        e.id as environment_id,
        rb.role,
        'inherited' as binding_type,
        'user' as entity_type,
        rb.user_id::text as entity_id,
        COALESCE(NULLIF(u.name, ''), u.email, u.provider_id) as entity_name,
        NULL::uuid as group_id,
        '' as group_name,
        'organization' as source_type
    FROM environments e
    INNER JOIN secret_groups sg ON e.secret_group_id = sg.id
    INNER JOIN role_bindings rb ON rb.resource_type = 'organization' AND rb.resource_id = sg.organization_id
    INNER JOIN users u ON rb.user_id = u.id
    WHERE e.id = $1
      AND rb.group_id IS NULL
      AND rb.secret_group_id IS NULL
      AND rb.environment_id IS NULL
),
inherited_org_group_bindings AS (
    -- Inherited group bindings from organization
    SELECT 
        e.id as environment_id,
        rb.role,
        'inherited' as binding_type,
        'group' as entity_type,
        '' as entity_id,
        '' as entity_name,
        rb.group_id,
        ug.name as group_name,
        'organization' as source_type
    FROM environments e
    INNER JOIN secret_groups sg ON e.secret_group_id = sg.id
    INNER JOIN role_bindings rb ON rb.resource_type = 'organization' AND rb.resource_id = sg.organization_id
    INNER JOIN user_groups ug ON rb.group_id = ug.id
    WHERE e.id = $1
      AND rb.user_id IS NULL
      AND rb.secret_group_id IS NULL
      AND rb.environment_id IS NULL
)
SELECT 
    environment_id,
    role,
    binding_type,
    entity_type,
    entity_id,
    entity_name,
    group_id,
    group_name,
    source_type
FROM direct_env_bindings
UNION ALL
SELECT 
    environment_id,
    role,
    binding_type,
    entity_type,
    entity_id,
    entity_name,
    group_id,
    group_name,
    source_type
FROM group_env_bindings
UNION ALL
SELECT 
    environment_id,
    role,
    binding_type,
    entity_type,
    entity_id,
    entity_name,
    group_id,
    group_name,
    source_type
FROM inherited_sg_bindings
UNION ALL
SELECT 
    environment_id,
    role,
    binding_type,
    entity_type,
    entity_id,
    entity_name,
    group_id,
    group_name,
    source_type
FROM inherited_sg_group_bindings
UNION ALL
SELECT 
    environment_id,
    role,
    binding_type,
    entity_type,
    entity_id,
    entity_name,
    group_id,
    group_name,
    source_type
FROM inherited_org_bindings
UNION ALL
SELECT 
    environment_id,
    role,
    binding_type,
    entity_type,
    entity_id,
    entity_name,
    group_id,
    group_name,
    source_type
FROM inherited_org_group_bindings
ORDER BY source_type, role, entity_type, entity_name, group_name;