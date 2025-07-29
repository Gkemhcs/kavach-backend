-- name: CreateProviderCredential :one
INSERT INTO provider_credentials (environment_id, provider, credentials, config,created_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetProviderCredential :one
SELECT * FROM provider_credentials 
WHERE environment_id = $1 AND provider = $2;

-- name: ListProviderCredentials :many
SELECT * FROM provider_credentials 
WHERE environment_id = $1 
ORDER BY created_at DESC;

-- name: UpdateProviderCredential :one
UPDATE provider_credentials 
SET credentials = $3, config = $4, updated_at = now()
WHERE environment_id = $1 AND provider = $2
RETURNING *;

-- name: DeleteProviderCredential :exec
DELETE FROM provider_credentials 
WHERE environment_id = $1 AND provider = $2;

-- name: GetProviderCredentialByID :one
SELECT * FROM provider_credentials 
WHERE id = $1;