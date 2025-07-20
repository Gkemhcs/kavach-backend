package jwt

import (
	"time"

	jwtx "github.com/golang-jwt/jwt/v4"
)

// Manager handles JWT creation and verification using a secret key and token duration.
type Manager struct {
	secretKey     string
	tokenDuration time.Duration
}

// NewManager creates a new JWT Manager with the given secret key and token duration.
func NewManager(secretKey string, tokenDuration time.Duration) *Manager {
	return &Manager{
		secretKey:     secretKey,
		tokenDuration: tokenDuration,
	}
}

// Generate creates a signed JWT token string using the provided parameters.
func (m *Manager) Generate(params CreateJwtParams) (string, error) {
	claims := &Claims{
		UserID:     params.UserID,
		Provider:   params.Provider,
		Email:      params.Email,
		Username:   params.Username,
		ProviderID: params.ProviderID,
		RegisteredClaims: jwtx.RegisteredClaims{
			ExpiresAt: jwtx.NewNumericDate(time.Now().Add(m.tokenDuration)),
			IssuedAt:  jwtx.NewNumericDate(time.Now()),
		},
	}
	token := jwtx.NewWithClaims(jwtx.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.secretKey))
}

// Verify parses and validates a JWT token string, returning the claims if valid.
func (m *Manager) Verify(tokenStr string) (*Claims, error) {
	token, err := jwtx.ParseWithClaims(tokenStr, &Claims{}, func(token *jwtx.Token) (interface{}, error) {
		return []byte(m.secretKey), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwtx.ErrTokenInvalidClaims
	}
	return claims, nil
}
