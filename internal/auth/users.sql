-- name: CreateUser :one
INSERT INTO users (
    provider, provider_id, email, name, avatar_url
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;



-- name: UpsertUser :one
INSERT INTO users (provider, provider_id, email, name, avatar_url)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (provider, provider_id) DO UPDATE
SET email = EXCLUDED.email,
    name = EXCLUDED.name,
    avatar_url = EXCLUDED.avatar_url,
    updated_at = now()
RETURNING *;

-- name: GetUserByProviderID :one
SELECT * FROM users
WHERE provider = $1 AND provider_id = $2
LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1
LIMIT 1;


-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;


