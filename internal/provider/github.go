package provider

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	appErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	"github.com/google/go-github/v74/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/nacl/box"
)

// GitHubProvider implements ProviderSync for GitHub repositories
type GitHubProvider struct {
	credentials GitHubCredentials
	config      GitHubConfig
	logger      *logrus.Logger
	client      *github.Client
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

	// Set default retry configuration if not provided
	if githubConfig.RetryConfig.MaxRetries == 0 {
		githubConfig.RetryConfig = RetryConfig{
			MaxRetries:  3,
			RetryDelay:  2 * time.Second,
			MaxDelay:    30 * time.Second,
			BackoffType: "exponential",
		}
	}

	// Create GitHub client
	client := github.NewClient(nil).WithAuthToken(githubCreds.Token)

	return &GitHubProvider{
		credentials: githubCreds,
		config:      githubConfig,
		logger:      logger,
		client:      client,
	}, nil
}

// Sync syncs secrets to GitHub repository with retry logic
func (g *GitHubProvider) Sync(ctx context.Context, secrets []Secret) ([]SyncResult, error) {
	logEntry := g.logger.WithFields(logrus.Fields{
		"provider":     "github",
		"owner":        g.config.Owner,
		"repository":   g.config.Repository,
		"environment":  g.config.Environment,
		"secret_count": len(secrets),
	})

	logEntry.Info("Starting GitHub secrets sync with retry logic")

	var results []SyncResult

	for _, secret := range secrets {
		result := SyncResult{
			Name:    secret.Name,
			Success: false,
		}

		// Use retry wrapper for creating/updating secret
		err := g.retryCreateSecret(ctx, secret.Name, secret.Value)
		if err != nil {
			result.Error = err.Error()
			logEntry.WithFields(logrus.Fields{
				"secret_name": secret.Name,
				"error":       err.Error(),
			}).Error("Failed to sync secret to GitHub after all retry attempts")
		} else {
			result.Success = true
			logEntry.WithField("secret_name", secret.Name).Info("Successfully synced secret to GitHub")
		}

		results = append(results, result)
	}

	logEntry.WithFields(logrus.Fields{
		"synced_count": len(results),
		"total_count":  len(secrets),
	}).Info("Completed GitHub secrets sync")

	return results, nil
}

// retryCreateSecret wraps the createSecret function with retry logic
func (g *GitHubProvider) retryCreateSecret(ctx context.Context, secretName, secretValue string) error {
	logEntry := g.logger.WithFields(logrus.Fields{
		"secret_name": secretName,
		"max_retries": g.config.RetryConfig.MaxRetries,
	})

	var lastErr error
	for attempt := 1; attempt <= g.config.RetryConfig.MaxRetries; attempt++ {
		logEntry.WithField("attempt", attempt).Info("Attempting to create/update secret")

		err := g.createSecret(ctx, secretName, secretValue)
		if err != nil {
			lastErr = err

			// Don't retry for environment not found errors - they will always fail
			if err == appErrors.ErrGitHubEnvironmentNotFound {
				logEntry.WithFields(logrus.Fields{
					"attempt": attempt,
					"error":   err.Error(),
				}).Error("GitHub environment not found, skipping retries")
				return err
			}

			logEntry.WithFields(logrus.Fields{
				"attempt": attempt,
				"error":   err.Error(),
			}).Warn("Failed to create/update secret, will retry")

			if attempt < g.config.RetryConfig.MaxRetries {
				delay := g.calculateRetryDelay(attempt)
				logEntry.WithField("delay", delay).Info("Waiting before retry")
				time.Sleep(delay)
				continue
			}
		} else {
			logEntry.WithField("attempt", attempt).Info("Successfully created/updated secret")
			return nil
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", g.config.RetryConfig.MaxRetries, lastErr)
}

// createSecret creates or updates a GitHub repository or environment secret
func (g *GitHubProvider) createSecret(ctx context.Context, secretName, secretValue string) error {
	logEntry := g.logger.WithFields(logrus.Fields{
		"secret_name": secretName,
		"environment": g.config.Environment,
	})

	logEntry.Info("Creating or updating GitHub secret")

	// Determine if we're using repository secrets or environment secrets
	if g.config.Environment == "" || g.config.Environment == "default" {
		return g.createRepositorySecret(ctx, secretName, secretValue)
	} else {
		return g.createEnvironmentSecret(ctx, secretName, secretValue)
	}
}

// createRepositorySecret creates or updates a repository secret
func (g *GitHubProvider) createRepositorySecret(ctx context.Context, secretName, secretValue string) error {
	logEntry := g.logger.WithFields(logrus.Fields{
		"secret_name": secretName,
		"secret_type": "repository",
	})

	logEntry.Info("Creating or updating repository secret")

	// Get repository public key
	publicKey, err := g.getRepositoryPublicKey(ctx)
	if err != nil {
		return fmt.Errorf("failed to get repository public key: %w", err)
	}

	// Encrypt secret value using libsodium (NaCl) box encryption
	logEntry.WithFields(logrus.Fields{
		"public_key_id":     publicKey.KeyID,
		"public_key_length": len(publicKey.Key),
	}).Debug("Encrypting secret value for repository")

	encryptedValue, err := g.encryptSecretValueWithLibsodium(secretValue, publicKey.Key)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to encrypt secret value")
		return appErrors.ErrGitHubEncryptionFailed
	}

	logEntry.WithFields(logrus.Fields{
		"encrypted_value_length": len(encryptedValue),
	}).Debug("Secret value encrypted successfully for repository")

	// Create or update the secret using GitHub API
	secret := &github.EncryptedSecret{
		Name:           secretName,
		KeyID:          publicKey.KeyID,
		EncryptedValue: encryptedValue,
	}

	_, err = g.client.Actions.CreateOrUpdateRepoSecret(ctx, g.config.Owner, g.config.Repository, secret)
	if err != nil {
		return fmt.Errorf("failed to create/update repository secret: %w", err)
	}

	logEntry.Info("Successfully created/updated repository secret")
	return nil
}

// createEnvironmentSecret creates or updates an environment secret
func (g *GitHubProvider) createEnvironmentSecret(ctx context.Context, secretName, secretValue string) error {
	logEntry := g.logger.WithFields(logrus.Fields{
		"secret_name": secretName,
		"environment": g.config.Environment,
		"secret_type": "environment",
	})

	logEntry.Info("Creating or updating environment secret")

	// Get repository ID first
	repo, _, err := g.client.Repositories.Get(ctx, g.config.Owner, g.config.Repository)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}

	// Get environment public key
	publicKey, err := g.getEnvironmentPublicKey(ctx, repo.GetID())
	if err != nil {
		// Check if it's a 404 error (environment not found)
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Not Found") {
			logEntry.WithField("environment", g.config.Environment).Error("GitHub environment not found")
			return appErrors.ErrGitHubEnvironmentNotFound
		}
		return fmt.Errorf("failed to get environment public key: %w", err)
	}

	// Encrypt secret value using libsodium (NaCl) box encryption
	logEntry.WithFields(logrus.Fields{
		"public_key_id":     publicKey.KeyID,
		"public_key_length": len(publicKey.Key),
		"environment":       g.config.Environment,
	}).Debug("Encrypting secret value for environment")

	encryptedValue, err := g.encryptSecretValueWithLibsodium(secretValue, publicKey.Key)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to encrypt secret value")
		return appErrors.ErrGitHubEncryptionFailed
	}

	logEntry.WithFields(logrus.Fields{
		"encrypted_value_length": len(encryptedValue),
		"environment":            g.config.Environment,
	}).Debug("Secret value encrypted successfully for environment")

	// Create or update the environment secret using GitHub API
	secret := &github.EncryptedSecret{
		Name:           secretName,
		KeyID:          publicKey.KeyID,
		EncryptedValue: encryptedValue,
	}

	_, err = g.client.Actions.CreateOrUpdateEnvSecret(ctx, int(repo.GetID()), g.config.Environment, secret)
	if err != nil {
		// Check if it's a 404 error (environment not found)
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Not Found") {
			logEntry.WithField("environment", g.config.Environment).Error("GitHub environment not found")
			return appErrors.ErrGitHubEnvironmentNotFound
		}
		return fmt.Errorf("failed to create/update environment secret: %w", err)
	}

	logEntry.Info("Successfully created/updated environment secret")
	return nil
}

// getRepositoryPublicKey gets the repository's public key for encryption
func (g *GitHubProvider) getRepositoryPublicKey(ctx context.Context) (*GitHubPublicKey, error) {
	logEntry := g.logger.WithFields(logrus.Fields{
		"owner":      g.config.Owner,
		"repository": g.config.Repository,
	})

	logEntry.Info("Getting repository public key")

	publicKey, _, err := g.client.Actions.GetRepoPublicKey(ctx, g.config.Owner, g.config.Repository)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository public key: %w", err)
	}

	return &GitHubPublicKey{
		KeyID: publicKey.GetKeyID(),
		Key:   publicKey.GetKey(),
	}, nil
}

// getEnvironmentPublicKey gets the environment's public key for encryption
func (g *GitHubProvider) getEnvironmentPublicKey(ctx context.Context, repoID int64) (*GitHubPublicKey, error) {
	logEntry := g.logger.WithFields(logrus.Fields{
		"owner":       g.config.Owner,
		"repository":  g.config.Repository,
		"environment": g.config.Environment,
		"repo_id":     repoID,
	})

	logEntry.Info("Getting environment public key")

	publicKey, _, err := g.client.Actions.GetEnvPublicKey(ctx, int(repoID), g.config.Environment)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment public key: %w", err)
	}

	return &GitHubPublicKey{
		KeyID: publicKey.GetKeyID(),
		Key:   publicKey.GetKey(),
	}, nil
}

// encryptSecretValueWithLibsodium encrypts a secret value using libsodium sealed box encryption
// GitHub expects sealed box format: [ephemeral_public_key (32 bytes)] + [encrypted_data]
// This implements crypto_box_seal which is what GitHub expects
func (g *GitHubProvider) encryptSecretValueWithLibsodium(secretValue, publicKeyBase64 string) (string, error) {
	logEntry := g.logger.WithFields(logrus.Fields{
		"secret_length":     len(secretValue),
		"public_key_length": len(publicKeyBase64),
	})

	logEntry.Debug("Starting libsodium sealed box encryption")

	// Decode the base64 public key
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode public key: %w", err)
	}

	// Parse the public key (GitHub uses NaCl box format)
	if len(publicKeyBytes) != 32 {
		return "", fmt.Errorf("invalid public key length: expected 32 bytes, got %d", len(publicKeyBytes))
	}

	var publicKey [32]byte
	copy(publicKey[:], publicKeyBytes)

	// Use box.SealAnonymous for sealed box encryption (crypto_box_seal equivalent)
	// This automatically handles ephemeral key generation and combines everything
	secretBytes := []byte(secretValue)
	encrypted, err := box.SealAnonymous(nil, secretBytes, &publicKey, rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Return base64 encoded result
	// box.SealAnonymous already includes the ephemeral public key in the encrypted data
	result := base64.StdEncoding.EncodeToString(encrypted)

	logEntry.WithFields(logrus.Fields{
		"encrypted_length": len(encrypted),
		"result_length":    len(result),
	}).Debug("Libsodium sealed box encryption completed")

	return result, nil
}

// calculateRetryDelay calculates the delay for the next retry attempt
func (g *GitHubProvider) calculateRetryDelay(attempt int) time.Duration {
	switch g.config.RetryConfig.BackoffType {
	case "exponential":
		delay := g.config.RetryConfig.RetryDelay * time.Duration(1<<(attempt-1))
		if delay > g.config.RetryConfig.MaxDelay {
			delay = g.config.RetryConfig.MaxDelay
		}
		return delay
	case "linear":
		delay := g.config.RetryConfig.RetryDelay * time.Duration(attempt)
		if delay > g.config.RetryConfig.MaxDelay {
			delay = g.config.RetryConfig.MaxDelay
		}
		return delay
	case "constant":
		return g.config.RetryConfig.RetryDelay
	default:
		// Default to exponential backoff
		delay := g.config.RetryConfig.RetryDelay * time.Duration(1<<(attempt-1))
		if delay > g.config.RetryConfig.MaxDelay {
			delay = g.config.RetryConfig.MaxDelay
		}
		return delay
	}
}

// ValidateCredentials validates GitHub credentials by making a test API call
func (g *GitHubProvider) ValidateCredentials(ctx context.Context) error {
	logEntry := g.logger.WithFields(logrus.Fields{
		"provider":   "github",
		"owner":      g.config.Owner,
		"repository": g.config.Repository,
	})

	logEntry.Info("Validating GitHub credentials")

	// Test API call to verify credentials
	_, _, err := g.client.Repositories.Get(ctx, g.config.Owner, g.config.Repository)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("GitHub credential validation failed")
		return appErrors.ErrProviderCredentialValidationFailed
	}

	logEntry.Info("GitHub credentials validated successfully")
	return nil
}

// GetProviderName returns the name of the provider
func (g *GitHubProvider) GetProviderName() string {
	return "github"
}

// GitHubPublicKey represents GitHub's public key for encryption
type GitHubPublicKey struct {
	KeyID string `json:"key_id"`
	Key   string `json:"key"`
}
