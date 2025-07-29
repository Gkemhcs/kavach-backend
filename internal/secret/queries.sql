-- name: CreateSecretVersion :one
INSERT INTO secret_versions (environment_id, commit_message)
VALUES ($1, $2)
RETURNING *;

-- name: InsertSecret :exec
INSERT INTO secrets (version_id, name, value_encrypted)
VALUES ($1, $2, $3)
ON CONFLICT (version_id, name) DO UPDATE SET value_encrypted = EXCLUDED.value_encrypted;

-- name: ListSecretVersions :many
SELECT * FROM secret_versions WHERE environment_id = $1 ORDER BY created_at DESC;

-- name: GetSecretsForVersion :many
SELECT id, name, value_encrypted FROM secrets WHERE version_id = $1;

-- name: GetSecretVersion :one
SELECT * FROM secret_versions WHERE id = $1;

-- name: RollbackSecretsToVersion :exec
INSERT INTO secrets (version_id, name, value_encrypted)
SELECT $1::VARCHAR(8), s.name, s.value_encrypted
FROM secrets s
WHERE s.version_id = $2
ON CONFLICT (version_id, name) DO UPDATE SET value_encrypted = EXCLUDED.value_encrypted;

-- name: DiffSecretVersions :many
SELECT 
    COALESCE(s1.name, s2.name) as name,
    s1.value_encrypted AS value_v1, 
    s2.value_encrypted AS value_v2
FROM (
    SELECT name, value_encrypted 
    FROM secrets 
    WHERE secrets.version_id = $1
) s1
FULL OUTER JOIN (
    SELECT name, value_encrypted 
    FROM secrets 
    WHERE secrets.version_id = $2
) s2 ON s1.name = s2.name; 