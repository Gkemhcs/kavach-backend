-- Drop unique index for group role
DROP INDEX IF EXISTS unique_group_role_per_resource;

-- Drop unique index for user role
DROP INDEX IF EXISTS unique_user_role_per_resource;

-- Drop check constraint for mutual exclusivity
ALTER TABLE role_bindings
DROP CONSTRAINT IF EXISTS chk_subject_exclusive;
