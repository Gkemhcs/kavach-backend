package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GitHubProvider implements OAuthProvider for GitHub authentication.
type GitHubProvider struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// NewGitHubProvider creates a new GitHubProvider with the given client ID, secret, and redirect URL.
func NewGitHubProvider(clientID, clientSecret, redirectURL string) *GitHubProvider {
	return &GitHubProvider{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
	}
}

// GetAuthURL returns the GitHub OAuth authorization URL for the given state.
func (g *GitHubProvider) GetAuthURL(state string) string {
	baseURL := "https://github.com/login/oauth/authorize"
	params := url.Values{}
	params.Add("client_id", g.ClientID)
	params.Add("redirect_uri", g.RedirectURL)
	params.Add("scope", "read:user user:email")
	params.Add("state", state)

	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

// ExchangeCode exchanges the authorization code for an access token from GitHub.
func (g *GitHubProvider) ExchangeCode(ctx context.Context, code string) (*OAuthToken, error) {
	data := url.Values{}
	data.Set("client_id", g.ClientID)
	data.Set("client_secret", g.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", g.RedirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return &OAuthToken{
		AccessToken: res.AccessToken,
		TokenType:   res.TokenType,
		Expiry:      time.Now().Add(1 * time.Hour).Unix(), // GitHub tokens don’t expire by default
	}, nil
}

// GetUserInfo retrieves the user's information from GitHub using the provided OAuth token.
func (g *GitHubProvider) GetUserInfo(ctx context.Context, token *OAuthToken) (*UserInfo, error) {
	// Make request to GitHub /user endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var githubUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
		Email     string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		return nil, err
	}

	// If primary email not available from /user, fallback to /emails endpoint
	if githubUser.Email == "" {
		emailReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
		if err != nil {
			return nil, err
		}
		emailReq.Header.Set("Authorization", "Bearer "+token.AccessToken)

		emailResp, err := http.DefaultClient.Do(emailReq)
		if err != nil {
			return nil, err
		}
		defer emailResp.Body.Close()

		var emails []struct {
			Email   string `json:"email"`
			Primary bool   `json:"primary"`
		}
		if err := json.NewDecoder(emailResp.Body).Decode(&emails); err != nil {
			return nil, err
		}
		for _, e := range emails {
			if e.Primary {
				githubUser.Email = e.Email
				break
			}
		}
	}

	return &UserInfo{
		Email:      githubUser.Email,
		Username:   githubUser.Login,
		AvatarURL:  githubUser.AvatarURL,
		ProviderID: int(githubUser.ID),
		Provider:   "github",
	}, nil
}

// StartDeviceFlow initiates the device authorization flow and returns device/user codes and verification URIs.
func (g *GitHubProvider) StartDeviceFlow(ctx context.Context) (*DeviceCodeResponse, error) {
	data := url.Values{}
	data.Set("client_id", g.ClientID)
	data.Set("scope", "read:user user:email")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/device/code", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body for debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check if response is successful
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub device flow failed with status %d: %s", resp.StatusCode, string(body))
	}

	var res DeviceCodeResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %v, body: %s", err, string(body))
	}

	// Validate that we got the required fields
	if res.DeviceCode == "" || res.UserCode == "" {
		return nil, fmt.Errorf("GitHub returned empty device code or user code: %+v", res)
	}

	return &res, nil
}

// PollDeviceToken polls for the device flow token using the device code.
func (g *GitHubProvider) PollDeviceToken(ctx context.Context, deviceCode string) (*UserInfo, error) {
	data := url.Values{}
	data.Set("client_id", g.ClientID)
	data.Set("device_code", deviceCode)
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	start := time.Now()
	pollInterval := 5 * time.Second // Start with 5 seconds

	for {
		// Timeout after 2 minutes
		if time.Since(start) > 2*time.Minute {
			return nil, fmt.Errorf("device_authorization_timeout")
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var res struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
			Scope       string `json:"scope"`
			Error       string `json:"error"`
			ErrorDesc   string `json:"error_description"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return nil, err
		}

		if res.Error != "" {
			if res.Error == "authorization_pending" {
				time.Sleep(pollInterval)
				continue
			}
			if res.Error == "slow_down" {
				// GitHub is asking us to slow down, increase polling interval
				pollInterval = time.Duration(float64(pollInterval) * 1.5)
				if pollInterval > 30*time.Second {
					pollInterval = 30 * time.Second // Cap at 30 seconds
				}
				time.Sleep(pollInterval)
				continue
			}
			return nil, fmt.Errorf("device flow error: %s", res.ErrorDesc)
		}

		token := &OAuthToken{
			AccessToken: res.AccessToken,
			TokenType:   res.TokenType,
			Expiry:      time.Now().Add(1 * time.Hour).Unix(),
		}
		return g.GetUserInfo(ctx, token)
	}
}
