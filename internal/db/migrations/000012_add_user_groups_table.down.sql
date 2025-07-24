DROP TABLE user_group_members;
DROP TABLE user_groups;

ALTER TABLE role_bindings
DROP COLUMN IF EXISTS group_id;