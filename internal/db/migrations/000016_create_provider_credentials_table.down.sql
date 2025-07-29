-- +goose Down
-- Migration to drop provider_credentials table

DROP TABLE IF EXISTS provider_credentials;
DROP INDEX IF EXISTS idx_provider_credentials_environment_provider;
DROP INDEX IF EXISTS idx_provider_credentials_provider;