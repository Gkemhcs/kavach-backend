-- name: ListOrganizationsWithMember :many
SELECT 
  o.id AS org_id,
  o.name,
  o.created_at,
  om.user_id,
  om.role
FROM org_members om
JOIN organizations o ON om.org_id = o.id
WHERE om.user_id = $1;




-- name: ListMembersOfOrganization :many
SELECT 
  om.user_id,
  om.role,
  om.org_id,
  o.name AS org_name
FROM org_members om
JOIN organizations o ON om.org_id = o.id
WHERE om.org_id = $1;
