-- +goose Up
-- Migration to create secret_versions and secrets tables for versioned secret storage

CREATE TABLE secret_versions (
    id VARCHAR(8) PRIMARY KEY DEFAULT substr(md5(random()::text), 1, 8),
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    commit_message TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE secrets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version_id VARCHAR(8) NOT NULL REFERENCES secret_versions(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    value_encrypted BYTEA NOT NULL,
    UNIQUE (version_id, name)
); 