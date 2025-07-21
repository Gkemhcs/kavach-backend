-- name: ListSecretGroupsWithMember :many
SELECT 
  sg.id AS secret_group_id,
  sg.name,
  sg.organization_id,
  sg.created_at,
  sgm.user_id,
  sgm.role
FROM secret_group_members sgm
JOIN secret_groups sg ON sgm.secret_group_id = sg.id
WHERE sgm.user_id = $1;


-- name: ListMembersOfSecretGroup :many
SELECT 
  sgm.user_id,
  sgm.role,
  sgm.secret_group_id,
  sg.name AS secret_group_name
FROM secret_group_members sgm
JOIN secret_groups sg ON sgm.secret_group_id = sg.id
WHERE sgm.secret_group_id = $1;
