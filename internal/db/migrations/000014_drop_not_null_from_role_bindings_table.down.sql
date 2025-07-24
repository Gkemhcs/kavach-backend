-- Ensure there are no nulls first (safe check)
UPDATE role_bindings
SET user_id = '00000000-0000-0000-0000-000000000000'
WHERE user_id IS NULL; -- optional fallback if you need to run down forcefully

ALTER TABLE role_bindings
ALTER COLUMN user_id SET NOT NULL;
