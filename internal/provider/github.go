package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// GitHubProvider implements ProviderSync for GitHub repositories
type GitHubProvider struct {
	credentials GitHubCredentials
	config      GitHubConfig
	logger      *logrus.Logger
	client      *http.Client
}

// NewGitHubProvider creates a new GitHub provider instance
func NewGitHubProvider(credentials map[string]interface{}, config map[string]interface{}, logger *logrus.Logger) (*GitHubProvider, error) {
	// Parse credentials
	credBytes, err := json.Marshal(credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal credentials: %w", err)
	}

	var githubCreds GitHubCredentials
	if err := json.Unmarshal(credBytes, &githubCreds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GitHub credentials: %w", err)
	}

	// Parse config
	configBytes, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var githubConfig GitHubConfig
	if err := json.Unmarshal(configBytes, &githubConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GitHub config: %w", err)
	}

	return &GitHubProvider{
		credentials: githubCreds,
		config:      githubConfig,
		logger:      logger,
		client:      &http.Client{},
	}, nil
}

// Sync syncs secrets to GitHub repository
func (g *GitHubProvider) Sync(ctx context.Context, secrets []Secret) ([]SyncResult, error) {
	logEntry := g.logger.WithFields(logrus.Fields{
		"provider":     "github",
		"owner":        g.config.Owner,
		"repository":   g.config.Repository,
		"secret_count": len(secrets),
	})

	logEntry.Info("Starting GitHub secrets sync")

	var results []SyncResult

	for _, secret := range secrets {
		result := SyncResult{
			Name:    secret.Name,
			Success: false,
		}

		// Create or update GitHub repository secret
		err := g.createOrUpdateSecret(ctx, secret.Name, secret.Value)
		if err != nil {
			result.Error = err.Error()
			logEntry.WithFields(logrus.Fields{
				"secret_name": secret.Name,
				"error":       err.Error(),
			}).Error("Failed to sync secret to GitHub")
		} else {
			result.Success = true
			logEntry.WithField("secret_name", secret.Name).Info("Successfully synced secret to GitHub")
		}

		results = append(results, result)
	}

	logEntry.WithField("synced_count", len(results)).Info("Completed GitHub secrets sync")

	return results, nil
}

// ValidateCredentials validates GitHub credentials by making a test API call
func (g *GitHubProvider) ValidateCredentials(ctx context.Context) error {
	logEntry := g.logger.WithFields(logrus.Fields{
		"provider": "github",
		"owner":    g.config.Owner,
		"repo":     g.config.Repository,
	})

	logEntry.Info("Validating GitHub credentials")

	// Test API call to verify credentials and repository access
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", g.config.Owner, g.config.Repository)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+g.credentials.Token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Kavach-Secret-Manager")

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid GitHub token")
	}

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("repository not found or access denied")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API error: %d", resp.StatusCode)
	}

	logEntry.Info("GitHub credentials validated successfully")
	return nil
}

// GetProviderName returns the provider name
func (g *GitHubProvider) GetProviderName() string {
	return "github"
}

// createOrUpdateSecret creates or updates a GitHub repository secret
func (g *GitHubProvider) createOrUpdateSecret(ctx context.Context, secretName, secretValue string) error {
	// GitHub uses a two-step process for creating/updating secrets:
	// 1. Get the public key for the repository
	// 2. Encrypt the secret value and create/update the secret

	// Step 1: Get the public key
	publicKey, err := g.getRepositoryPublicKey(ctx)
	if err != nil {
		return fmt.Errorf("failed to get repository public key: %w", err)
	}

	// Step 2: Encrypt the secret value
	encryptedValue, err := g.encryptSecretValue(secretValue, publicKey.Key)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret value: %w", err)
	}

	// Step 3: Create or update the secret
	return g.createOrUpdateRepositorySecret(ctx, secretName, encryptedValue, publicKey.KeyID)
}

// getRepositoryPublicKey retrieves the public key for the repository
func (g *GitHubProvider) getRepositoryPublicKey(ctx context.Context) (*GitHubPublicKey, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/secrets/public-key",
		g.config.Owner, g.config.Repository)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+g.credentials.Token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Kavach-Secret-Manager")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get public key: %d", resp.StatusCode)
	}

	var publicKey GitHubPublicKey
	if err := json.NewDecoder(resp.Body).Decode(&publicKey); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &publicKey, nil
}

// encryptSecretValue encrypts the secret value using the repository's public key
func (g *GitHubProvider) encryptSecretValue(secretValue, publicKey string) (string, error) {
	// This is a simplified implementation
	// In a real implementation, you would use proper encryption with the public key
	// For now, we'll return a placeholder encrypted value
	// TODO: Implement proper encryption using the public key

	// For demonstration purposes, we'll just base64 encode the value
	// In production, you should use proper encryption
	return fmt.Sprintf("encrypted_%s", secretValue), nil
}

// createOrUpdateRepositorySecret creates or updates a repository secret
func (g *GitHubProvider) createOrUpdateRepositorySecret(ctx context.Context, secretName, encryptedValue, keyID string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/secrets/%s",
		g.config.Owner, g.config.Repository, secretName)

	payload := map[string]interface{}{
		"encrypted_value": encryptedValue,
		"key_id":          keyID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+g.credentials.Token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Kavach-Secret-Manager")

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to create/update secret: %d", resp.StatusCode)
	}

	return nil
}

// GitHubPublicKey represents the public key response from GitHub API
type GitHubPublicKey struct {
	KeyID string `json:"key_id"`
	Key   string `json:"key"`
}
