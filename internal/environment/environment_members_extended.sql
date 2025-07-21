-- name: ListEnvironmentsWithMember :many
SELECT 
  e.id AS environment_id,
  e.name,
  e.secret_group_id,
  e.created_at,
  em.user_id,
  em.role
FROM environment_members em
JOIN environments e ON em.environment_id = e.id
WHERE em.user_id = $1;



-- name: ListMembersOfEnvironment :many
SELECT 
  em.user_id,
  em.role,
  em.environment_id,
  e.name AS environment_name
FROM environment_members em
JOIN environments e ON em.environment_id = e.id
WHERE em.environment_id = $1;
