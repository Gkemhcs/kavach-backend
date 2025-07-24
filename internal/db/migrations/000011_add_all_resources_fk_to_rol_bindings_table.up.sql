-- Add columns
ALTER TABLE role_bindings
  ADD COLUMN organization_id UUID NOT NULL,
  ADD COLUMN secret_group_id UUID,
  ADD COLUMN environment_id UUID;

-- Add foreign keys
ALTER TABLE role_bindings
  ADD CONSTRAINT fk_organization_id
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE;

ALTER TABLE role_bindings
  ADD CONSTRAINT fk_secret_group_id
    FOREIGN KEY (secret_group_id) REFERENCES secret_groups(id) ON DELETE CASCADE;

ALTER TABLE role_bindings
  ADD CONSTRAINT fk_environment_id
    FOREIGN KEY (environment_id) REFERENCES environments(id) ON DELETE CASCADE;
