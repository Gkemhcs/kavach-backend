package provider

// OAuthToken represents an OAuth access token and related information.
type OAuthToken struct {
	AccessToken  string // The access token string
	RefreshToken string // The refresh token string (if provided)
	Expiry       int64  // Expiry time as a Unix timestamp
	TokenType    string // The type of the token (e.g., Bearer)
}

// UserInfo contains user profile information returned by the OAuth provider.
type UserInfo struct {
	ProviderID int    // Provider-specific user ID
	Provider   string // OAuth provider name
	Email      string // User email address
	Username   string // Username or display name
	AvatarURL  string // URL to the user's avatar image
}

// DeviceCodeResponse represents the response from the device code endpoint.
type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`               // Code for device authentication
	UserCode                string `json:"user_code"`                 // Code for user authentication
	VerificationURI         string `json:"verification_uri"`          // URI for user verification
	VerificationURIComplete string `json:"verification_uri_complete"` // Complete URI for user verification
	ExpiresIn               int    `json:"expires_in"`                // Expiry time in seconds
	Interval                int    `json:"interval"`                  // Polling interval in seconds
}

type DeviceTokenResponse struct {
	Token        string    `json:"token"`         // Access token
	RefreshToken string    `json:"refresh_token"` // Refresh token
	User         *UserInfo `json:"user"`          // Associated user info
}
