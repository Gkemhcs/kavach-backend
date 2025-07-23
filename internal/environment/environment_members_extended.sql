-- ListEnvironmentsWithMember returns all environments a user is a member of within a secret group and organization.
-- Used to show environments accessible to a user for a given group/org context.
-- name: ListEnvironmentsWithMember :many
SELECT 
  e.id AS environment_id,
  e.name,
  e.secret_group_id,
  sg.name AS secret_group_name,
  e.created_at,
  em.user_id,
  em.role
FROM environment_members em
JOIN environments e ON em.environment_id = e.id
JOIN secret_groups sg ON e.secret_group_id = sg.id
WHERE em.user_id = $1 AND e.secret_group_id = $2 and sg.organization_id = $3;


-- ListMembersOfEnvironment returns all members and their roles for a given environment.
-- Used to manage and display environment membership and permissions.
-- name: ListMembersOfEnvironment :many
SELECT 
  em.user_id,
  em.role,
  em.environment_id,
  e.name AS environment_name
FROM environment_members em
JOIN environments e ON em.environment_id = e.id
WHERE em.environment_id = $1;
