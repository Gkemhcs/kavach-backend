-- ListSecretGroupsWithMember returns all secret groups a user is a member of within an organization.
-- Used to show secret groups accessible to a user for a given org context.
-- name: ListSecretGroupsWithMember :many
SELECT 
  sg.id AS secret_group_id,
  sg.name,
  sg.organization_id,
  o.name AS organization_name,
  sg.created_at,
  sgm.user_id,
  sgm.role
FROM secret_group_members sgm
JOIN secret_groups sg ON sgm.secret_group_id = sg.id
JOIN organizations o ON sg.organization_id = o.id
WHERE sgm.user_id = $1 AND  sg.organization_id = $2 ;


-- ListMembersOfSecretGroup returns all members and their roles for a given secret group.
-- Used to manage and display secret group membership and permissions.
-- name: ListMembersOfSecretGroup :many
SELECT 
  sgm.user_id,
  sgm.role,
  sgm.secret_group_id,
  sg.name AS secret_group_name
FROM secret_group_members sgm
JOIN secret_groups sg ON sgm.secret_group_id = sg.id
WHERE sgm.secret_group_id = $1;
