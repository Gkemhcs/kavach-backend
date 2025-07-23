-- CreateUser inserts a new user into the users table and returns the created user.
-- Used during initial OAuth registration when a user logs in for the first time.
-- name: CreateUser :one
INSERT INTO users (
    provider, provider_id, email, name, avatar_url
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;



-- UpsertUser inserts or updates a user based on provider and provider_id.
-- Ensures user info is always up-to-date after OAuth login.
-- name: UpsertUser :one
INSERT INTO users (provider, provider_id, email, name, avatar_url)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (provider, provider_id) DO UPDATE
SET email = EXCLUDED.email,
    name = EXCLUDED.name,
    avatar_url = EXCLUDED.avatar_url,
    updated_at = now()
RETURNING *;

-- GetUserByProviderID fetches a user by provider and provider_id.
-- Used to look up users during login and token refresh.
-- name: GetUserByProviderID :one
SELECT * FROM users
WHERE provider = $1 AND provider_id = $2
LIMIT 1;

-- GetUserByID fetches a user by their unique ID.
-- Used for user profile lookups and internal references.
-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1
LIMIT 1;


-- DeleteUser removes a user from the users table by ID.
-- Used for account deletion and admin operations.
-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;


