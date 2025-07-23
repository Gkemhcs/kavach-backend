package jwt

import (
	"time"

	apperrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	jwtx "github.com/golang-jwt/jwt/v4"
)

// Manager handles JWT creation and verification using a secret key and token duration.
type Manager struct {
	secretKey            string
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
}

// NewManager creates a new JWT Manager with the given secret key and token durations.
func NewManager(secretKey string, accessTokenDuration, refreshTokenDuration time.Duration) *Manager {
	return &Manager{
		secretKey:            secretKey,
		accessTokenDuration:  accessTokenDuration,
		refreshTokenDuration: refreshTokenDuration,
	}
}

// Generate creates a signed JWT token string using the provided parameters.
func (m *Manager) Generate(params CreateJwtParams) (string, error) {
	// Set up claims for access token
	claims := &Claims{
		UserID:     params.UserID,
		Provider:   params.Provider,
		Email:      params.Email,
		Username:   params.Username,
		ProviderID: params.ProviderID,
		RegisteredClaims: jwtx.RegisteredClaims{
			ExpiresAt: jwtx.NewNumericDate(time.Now().Add(m.accessTokenDuration)),
			IssuedAt:  jwtx.NewNumericDate(time.Now()),
		},
	}
	// Create and sign the token
	token := jwtx.NewWithClaims(jwtx.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.secretKey))
}

// GenerateRefresh creates a signed refresh JWT token string using the provided parameters, with a longer expiry.
func (m *Manager) GenerateRefresh(params CreateJwtParams) (string, error) {
	// Set up claims for refresh token
	claims := &Claims{
		UserID:     params.UserID,
		Provider:   params.Provider,
		Email:      params.Email,
		Username:   params.Username,
		ProviderID: params.ProviderID,
		RegisteredClaims: jwtx.RegisteredClaims{
			ExpiresAt: jwtx.NewNumericDate(time.Now().Add(m.refreshTokenDuration)),
			IssuedAt:  jwtx.NewNumericDate(time.Now()),
		},
	}
	// Create and sign the refresh token
	token := jwtx.NewWithClaims(jwtx.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.secretKey))
}

// Verify parses and validates a JWT token string, returning the claims if valid.
func (m *Manager) Verify(tokenStr string) (*Claims, error) {
	// Parse the token with claims
	token, err := jwtx.ParseWithClaims(tokenStr, &Claims{}, func(token *jwtx.Token) (interface{}, error) {
		return []byte(m.secretKey), nil
	})
	if err != nil {
		// Check for expired token error
		if ve, ok := err.(*jwtx.ValidationError); ok {
			if ve.Errors&jwtx.ValidationErrorExpired != 0 {
				return nil, apperrors.ErrExpiredToken
			}
		}
		return nil, apperrors.ErrInvalidToken
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, apperrors.ErrInvalidToken
	}
	// Check expiry explicitly (defensive)
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, apperrors.ErrExpiredToken
	}
	return claims, nil
}
