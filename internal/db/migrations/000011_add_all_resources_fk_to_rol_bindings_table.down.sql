-- Drop foreign keys
ALTER TABLE role_bindings DROP CONSTRAINT IF EXISTS fk_environment_id;
ALTER TABLE role_bindings DROP CONSTRAINT IF EXISTS fk_secret_group_id;
ALTER TABLE role_bindings DROP CONSTRAINT IF EXISTS fk_organization_id;

-- Drop columns
ALTER TABLE role_bindings DROP COLUMN IF EXISTS environment_id;
ALTER TABLE role_bindings DROP COLUMN IF EXISTS secret_group_id;
ALTER TABLE role_bindings DROP COLUMN IF EXISTS organization_id;
