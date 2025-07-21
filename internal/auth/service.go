package auth

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	userdb "github.com/Gkemhcs/kavach-backend/internal/auth/gen"
	"github.com/Gkemhcs/kavach-backend/internal/auth/jwt"
	"github.com/Gkemhcs/kavach-backend/internal/auth/provider"
	apperrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	"github.com/sirupsen/logrus"
)

// AuthService provides authentication logic using an OAuth provider, user repository, and JWT manager.
type AuthService struct {
	provider provider.OAuthProvider
	userRepo userdb.Querier
	jwter    *jwt.Manager
	logger   *logrus.Logger
}

// NewAuthService creates a new AuthService with the given provider, repository, and JWT manager.
func NewAuthService(provider provider.OAuthProvider, repository userdb.Querier, jwter *jwt.Manager, logger *logrus.Logger) *AuthService {
	return &AuthService{
		provider: provider,
		userRepo: repository,
		jwter:    jwter,
		logger:   logger,
	}
}

func (s *AuthService) StartDeviceFlow(ctx context.Context) (*provider.DeviceCodeResponse, error) {
	s.logger.Info("Starting device flow with OAuth provider")
	resp, err := s.provider.StartDeviceFlow(ctx)
	if err != nil {
		s.logger.Errorf("Device flow start error: %v", err)
		return nil, err
	}
	s.logger.Infof("Device flow started: device_code=%s user_code=%s", resp.DeviceCode, resp.UserCode)
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
	s.logger.Infof("Polling device token for device_code=%s", deviceCode)
	userInfo, err := s.provider.PollDeviceToken(ctx, deviceCode)
	if err != nil {
		s.logger.Errorf("Device token polling error: %v", err)
		return nil, err
	}
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
	s.logger.Infof("Upserting user: provider=%s provider_id=%s", params.Provider, params.ProviderID)
	user, err := s.userRepo.UpsertUser(ctx, params)
	if err != nil {
		s.logger.Errorf("User upsert error: %v", err)
		return nil, err
	}
	claims := &jwt.Claims{
		UserID:     user.ID.String(),
		Provider:   user.Provider,
		ProviderID: user.ProviderID,
		Email:      user.Email.String,
		Username:   user.Name.String,
	}
	createJwtParams := s.createJwtParamsFromClaims(claims)
	tokenStr, err := s.jwter.Generate(createJwtParams)
	if err != nil {
		s.logger.Errorf("JWT generation error: %v", err)
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}
	refreshToken, err := s.jwter.GenerateRefresh(createJwtParams)
	if err != nil {
		s.logger.Errorf("Refresh token generation error: %v", err)
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	s.logger.Infof("Tokens generated for user_id=%s", user.ID.String())
	return &provider.DeviceTokenResponse{
		Token:        tokenStr,
		RefreshToken: refreshToken,
		User:         userInfo,
	}, nil
}

// GetLoginURL returns the OAuth provider's login URL for the given state.
func (s *AuthService) GetLoginURL(state string) string {
	return s.provider.GetAuthURL(state)
}

// HandleCallback processes the OAuth callback, upserts the user, and returns user info and a JWT token.
func (s *AuthService) HandleCallback(ctx context.Context, code string) (*provider.UserInfo, string, string, error) {
	s.logger.Infof("Handling OAuth callback with code=%s", code)
	token, err := s.provider.ExchangeCode(ctx, code)
	if err != nil {
		s.logger.Errorf("Exchange code error: %v", err)
		return nil, "", "", fmt.Errorf("exchange code failed: %w", err)
	}

	userInfo, err := s.provider.GetUserInfo(ctx, token)
	if err != nil {
		s.logger.Errorf("Get user info error: %v", err)
		return nil, "", "", fmt.Errorf("failed to get user info: %w", err)
	}
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
	s.logger.Infof("Upserting user: provider=%s provider_id=%s", params.Provider, params.ProviderID)
	user, err := s.userRepo.UpsertUser(ctx, params)
	if err != nil {
		s.logger.Errorf("User upsert error: %v", err)
		return nil, "", "", err
	}
	claims := &jwt.Claims{
		UserID:     user.ID.String(),
		Provider:   user.Provider,
		ProviderID: user.ProviderID,
		Email:      user.Email.String,
		Username:   user.Name.String,
	}
	createJwtParams := s.createJwtParamsFromClaims(claims)
	tokenStr, err := s.jwter.Generate(createJwtParams)
	if err != nil {
		s.logger.Errorf("JWT generation error: %v", err)
		return nil, "", "", fmt.Errorf("failed to generate JWT: %w", err)
	}
	refreshToken, err := s.jwter.GenerateRefresh(createJwtParams)
	if err != nil {
		s.logger.Errorf("Refresh token generation error: %v", err)
		return nil, "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	s.logger.Infof("Tokens generated for user_id=%s", user.ID.String())
	return userInfo, tokenStr, refreshToken, nil
}

// RefreshTokens validates the refresh token and issues new access and refresh tokens.
func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (string, string, error) {
	s.logger.Info("Refreshing tokens using refresh token")
	claims, err := s.jwter.Verify(refreshToken)
	if err != nil {
		if err == apperrors.ErrExpiredToken {
			s.logger.Warn("Refresh token expired")
			return "", "", apperrors.ErrExpiredToken
		}
		s.logger.Warnf("Invalid refresh token: %v", err)
		return "", "", apperrors.ErrInvalidToken
	}
	params := s.createJwtParamsFromClaims(claims)
	token, err := s.jwter.Generate(params)
	if err != nil {
		s.logger.Errorf("JWT generation error: %v", err)
		return "", "", err
	}
	newRefreshToken, err := s.jwter.GenerateRefresh(params)
	if err != nil {
		s.logger.Errorf("Refresh token generation error: %v", err)
		return "", "", err
	}
	s.logger.Infof("Refreshed tokens for user_id=%s", claims.UserID)
	return token, newRefreshToken, nil
}

// createJwtParamsFromClaims creates CreateJwtParams from JWT claims (for refresh token flow)
func (s *AuthService) createJwtParamsFromClaims(claims *jwt.Claims) jwt.CreateJwtParams {
	return jwt.CreateJwtParams{
		UserID:     claims.UserID,
		Provider:   claims.Provider,
		ProviderID: claims.ProviderID,
		Email:      claims.Email,
		Username:   claims.Username,
	}
}
