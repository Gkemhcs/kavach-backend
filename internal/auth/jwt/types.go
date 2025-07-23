package jwt

import (
	jwtx "github.com/golang-jwt/jwt/v4"
)

// Claims represents the JWT claims for a user, including standard and custom fields.
type Claims struct {
	UserID                string `json:"user_id"`     // Unique user identifier
	Provider              string `json:"provider"`    // OAuth provider name
	ProviderID            string `json:"provider_id"` // Provider-specific user ID
	Email                 string `json:"email"`       // User email address
	Username              string `json:"username"`    // Username or display name
	jwtx.RegisteredClaims        // Embedded standard JWT claims
}

// CreateJwtParams contains the parameters required to generate a JWT for a user.
type CreateJwtParams struct {
	UserID     string // Unique user identifier
	Provider   string // OAuth provider name
	ProviderID string // Provider-specific user ID
	Email      string // User email address
	Username   string // Username or display name
}
