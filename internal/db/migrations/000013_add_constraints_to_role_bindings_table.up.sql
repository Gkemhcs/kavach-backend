-- Enforce mutual exclusivity between user_id and group_id
ALTER TABLE role_bindings
ADD CONSTRAINT chk_subject_exclusive
  CHECK (
    (user_id IS NOT NULL AND group_id IS NULL) OR
    (user_id IS NULL AND group_id IS NOT NULL)
  );

-- Unique role per user per resource
CREATE UNIQUE INDEX unique_user_role_per_resource
ON role_bindings(user_id, resource_type, resource_id)
WHERE user_id IS NOT NULL;

-- Unique role per group per resource
CREATE UNIQUE INDEX unique_group_role_per_resource
ON role_bindings(group_id, resource_type, resource_id)
WHERE group_id IS NOT NULL;
