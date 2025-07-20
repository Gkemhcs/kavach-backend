package auth

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	userdb "github.com/Gkemhcs/kavach-backend/internal/auth/gen"
	"github.com/Gkemhcs/kavach-backend/internal/auth/jwt"
	"github.com/Gkemhcs/kavach-backend/internal/auth/provider"
)

// AuthService provides authentication logic using an OAuth provider, user repository, and JWT manager.
type AuthService struct {
	provider provider.OAuthProvider
	userRepo userdb.Querier
	jwter    *jwt.Manager
}

// NewAuthService creates a new AuthService with the given provider, repository, and JWT manager.
func NewAuthService(provider provider.OAuthProvider, repository userdb.Querier, jwter *jwt.Manager) *AuthService {
	return &AuthService{
		provider: provider,
		userRepo: repository,
		jwter:    jwter,
	}
}




func (s *AuthService) StartDeviceFlow(ctx context.Context) (*provider.DeviceCodeResponse, error) {
	// Call GitHub's device code endpoint
	resp, err := s.provider.StartDeviceFlow(ctx)
	if err != nil {
		return nil, err
	}
	return &provider.DeviceCodeResponse{
		DeviceCode:              resp.DeviceCode,
		UserCode:                resp.UserCode,
		VerificationURI:         resp.VerificationURI,
		VerificationURIComplete: resp.VerificationURIComplete,
		ExpiresIn:               resp.ExpiresIn,
		Interval:                resp.Interval,
	}, nil
}

func (s *AuthService) PollDeviceToken(ctx context.Context, deviceCode string) (*provider.DeviceTokenResponse, error) {
	// Poll GitHub for access token
	userInfo, err := s.provider.PollDeviceToken(ctx, deviceCode)
	if err != nil {
		return nil, err
	}
	// Upsert user and generate JWT as before
	params := userdb.UpsertUserParams{
		Provider:   userInfo.Provider,
		ProviderID: strconv.Itoa(userInfo.ProviderID),
		AvatarUrl: sql.NullString{
			String: userInfo.AvatarURL,
			Valid:  true,
		},
		Email: sql.NullString{
			String: userInfo.Email,
			Valid:  true,
		},
		Name: sql.NullString{
			String: userInfo.Username,
			Valid:  true,
		},
	}
	user, err := s.userRepo.UpsertUser(ctx, params)
	if err != nil {
		return nil, err
	}
	createJwtParams := jwt.CreateJwtParams{
		UserID:     user.ID.String(),
		Provider:   user.Provider,
		ProviderID: user.ProviderID,
		Email:      user.Email.String,
		Username:   user.Name.String,
	}
	tokenStr, err := s.jwter.Generate(createJwtParams)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}
	return &provider.DeviceTokenResponse{
		Token: tokenStr,
		User:  userInfo,
	}, nil
}

// GetLoginURL returns the OAuth provider's login URL for the given state.
func (s *AuthService) GetLoginURL(state string) string {
	return s.provider.GetAuthURL(state)
}

// HandleCallback processes the OAuth callback, upserts the user, and returns user info and a JWT token.
func (s *AuthService) HandleCallback(ctx context.Context, code string) (*provider.UserInfo, string, error) {
	token, err := s.provider.ExchangeCode(ctx, code)
	if err != nil {
		return nil, "", fmt.Errorf("exchange code failed: %w", err)
	}

	userInfo, err := s.provider.GetUserInfo(ctx, token)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user info: %w", err)
	}
	// Prepare parameters for upserting the user in the database
	params := userdb.UpsertUserParams{
		Provider:   userInfo.Provider,
		ProviderID: strconv.Itoa(userInfo.ProviderID),
		AvatarUrl: sql.NullString{
			String: userInfo.AvatarURL,
			Valid:  true,
		},
		Email: sql.NullString{
			String: userInfo.Email,
			Valid:  true,
		},
		Name: sql.NullString{
			String: userInfo.Username,
			Valid:  true,
		},
	}
	user, err := s.userRepo.UpsertUser(ctx, params)
	if err != nil {
		return nil, "", err
	}

	// Generate JWT token for the authenticated user
	createJwtParams := jwt.CreateJwtParams{
		UserID:     user.ID.String(),
		Provider:   user.Provider,
		ProviderID: user.ProviderID,
		Email:      user.Email.String,
		Username:   user.Name.String,
	}

	tokenStr, err := s.jwter.Generate(createJwtParams)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	return userInfo, tokenStr, nil
}
