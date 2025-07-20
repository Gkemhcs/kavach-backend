package provider

import (
	"context"
)

// OAuthProvider defines the interface for OAuth authentication providers.
type OAuthProvider interface {
	// GetAuthURL returns the provider's authorization URL for the given state.
	GetAuthURL(state string) string
	// ExchangeCode exchanges the authorization code for an access token.
	ExchangeCode(ctx context.Context, code string) (*OAuthToken, error)
	// GetUserInfo retrieves the user's information using the provided OAuth token.
	GetUserInfo(ctx context.Context, token *OAuthToken) (*UserInfo, error)
	// StartDeviceFlow initiates the device authorization flow and returns device/user codes and verification URIs.
	StartDeviceFlow(ctx context.Context) (*DeviceCodeResponse, error)
	// PollDeviceToken polls for the device flow token using the device code.
	PollDeviceToken(ctx context.Context, deviceCode string) (*UserInfo, error)
}
