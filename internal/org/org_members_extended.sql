-- ListOrganizationsWithMember returns all organizations a user is a member of.
-- Used to show organizations accessible to a user for a given context.
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



-- ListMembersOfOrganization returns all members and their roles for a given organization.
-- Used to manage and display organization membership and permissions.
-- name: ListMembersOfOrganization :many
SELECT 
  om.user_id,
  om.role,
  om.org_id,
  o.name AS org_name
FROM org_members om
JOIN organizations o ON om.org_id = o.id
WHERE om.org_id = $1;
