-- +goose Down
-- Rollback migration for secret_versions and secrets tables

DROP TABLE IF EXISTS secrets;
DROP TABLE IF EXISTS secret_versions; 