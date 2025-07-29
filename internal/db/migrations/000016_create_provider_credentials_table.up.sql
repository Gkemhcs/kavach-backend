-- +goose Up
-- Migration to create provider_credentials table for storing provider authentication and configuration

CREATE TABLE provider_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    provider TEXT NOT NULL CHECK (provider IN ('github', 'gcp', 'azure')),
    credentials BYTEA NOT NULL, -- Encrypted credentials for the provider
    config JSONB NOT NULL, -- Provider-specific configuration
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (environment_id, provider)
);

-- Index for efficient lookups by environment and provider
CREATE INDEX idx_provider_credentials_environment_provider ON provider_credentials(environment_id, provider);

-- Index for provider type lookups
CREATE INDEX idx_provider_credentials_provider ON provider_credentials(provider);