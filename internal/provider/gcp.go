package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	appErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GCPProvider implements ProviderSync for Google Cloud Secret Manager
type GCPProvider struct {
	credentials GCPCredentials
	config      GCPConfig
	logger      *logrus.Logger
	client      *secretmanager.Client
}

// NewGCPProvider creates a new GCP provider instance
func NewGCPProvider(credentials map[string]interface{}, config map[string]interface{}, logger *logrus.Logger) (*GCPProvider, error) {
	// Parse credentials
	credBytes, err := json.Marshal(credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal credentials: %w", err)
	}

	var gcpCreds GCPCredentials
	if err := json.Unmarshal(credBytes, &gcpCreds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GCP credentials: %w", err)
	}

	// Parse config
	configBytes, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var gcpConfig GCPConfig
	if err := json.Unmarshal(configBytes, &gcpConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GCP config: %w", err)
	}

	// Create service account credentials JSON
	serviceAccountJSON := map[string]interface{}{
		"type":                        gcpCreds.Type,
		"project_id":                  gcpCreds.ProjectID,
		"private_key_id":              gcpCreds.PrivateKeyID,
		"private_key":                 gcpCreds.PrivateKey,
		"client_email":                gcpCreds.ClientEmail,
		"client_id":                   gcpCreds.ClientID,
		"auth_uri":                    gcpCreds.AuthURI,
		"token_uri":                   gcpCreds.TokenURI,
		"auth_provider_x509_cert_url": gcpCreds.AuthProviderX509CertURL,
		"client_x509_cert_url":        gcpCreds.ClientX509CertURL,
	}

	credentialsJSON, err := json.Marshal(serviceAccountJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal service account credentials: %w", err)
	}

	// Create Secret Manager client based on location configuration
	ctx := context.Background()
	var client *secretmanager.Client

	if gcpConfig.SecretManagerLocation != "" {
		// Create regional client for specific location
		endpoint := fmt.Sprintf("secretmanager.%s.rep.googleapis.com:443", gcpConfig.SecretManagerLocation)
		client, err = secretmanager.NewClient(ctx,
			option.WithCredentialsJSON(credentialsJSON),
			option.WithEndpoint(endpoint))
		if err != nil {
			return nil, fmt.Errorf("failed to create regional Secret Manager client: %w", err)
		}
	} else {
		// Create global client (default)
		client, err = secretmanager.NewClient(ctx, option.WithCredentialsJSON(credentialsJSON))
		if err != nil {
			return nil, fmt.Errorf("failed to create global Secret Manager client: %w", err)
		}
	}

	return &GCPProvider{
		credentials: gcpCreds,
		config:      gcpConfig,
		logger:      logger,
		client:      client,
	}, nil
}

// Sync syncs secrets to Google Cloud Secret Manager
func (g *GCPProvider) Sync(ctx context.Context, secrets []Secret) ([]SyncResult, error) {
	logEntry := g.logger.WithFields(logrus.Fields{
		"provider":     "gcp",
		"project_id":   g.config.ProjectID,
		"location":     g.config.SecretManagerLocation,
		"secret_count": len(secrets),
	})

	logEntry.Info("Starting GCP secrets sync")

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
			}).Error("Failed to sync secret to GCP after all retry attempts")
		} else {
			result.Success = true
			logEntry.WithField("secret_name", secret.Name).Info("Successfully synced secret to GCP")
		}

		results = append(results, result)
	}

	logEntry.WithFields(logrus.Fields{
		"synced_count": len(results),
		"total_count":  len(secrets),
	}).Info("Completed GCP secrets sync")

	return results, nil
}

// retryCreateSecret wraps the createSecret function with retry logic
func (g *GCPProvider) retryCreateSecret(ctx context.Context, secretName, secretValue string) error {
	logEntry := g.logger.WithFields(logrus.Fields{
		"secret_name": secretName,
		"max_retries": g.config.RetryConfig.MaxRetries,
	})

	// Set default retry configuration if not provided
	if g.config.RetryConfig.MaxRetries == 0 {
		g.config.RetryConfig = RetryConfig{
			MaxRetries:  3,
			RetryDelay:  2 * time.Second,
			MaxDelay:    30 * time.Second,
			BackoffType: "exponential",
		}
	}

	var lastErr error
	for attempt := 1; attempt <= g.config.RetryConfig.MaxRetries; attempt++ {
		logEntry.WithField("attempt", attempt).Info("Attempting to create/update secret")

		err := g.createOrUpdateSecret(ctx, secretName, secretValue)
		if err != nil {
			lastErr = err
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

// buildSecretReplication builds the appropriate replication configuration for the secret
func (g *GCPProvider) buildSecretReplication() *secretmanagerpb.Secret {
	secret := &secretmanagerpb.Secret{
		Labels: map[string]string{
			"managed-by": "kavach-backend",
		},
	}

	if g.config.SecretManagerLocation != "" {
		// Regional secret - replication is automatically handled by the regional endpoint
		// Do not set replication field for regional secrets
	} else {
		// Global secret - use automatic replication
		secret.Replication = &secretmanagerpb.Replication{
			Replication: &secretmanagerpb.Replication_Automatic_{
				Automatic: &secretmanagerpb.Replication_Automatic{},
			},
		}
	}

	return secret
}

// calculateRetryDelay calculates the delay for the next retry attempt
func (g *GCPProvider) calculateRetryDelay(attempt int) time.Duration {
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

// ValidateCredentials validates GCP credentials by making a test API call
func (g *GCPProvider) ValidateCredentials(ctx context.Context) error {
	logEntry := g.logger.WithFields(logrus.Fields{
		"provider":   "gcp",
		"project_id": g.config.ProjectID,
	})

	logEntry.Info("Validating GCP credentials")

	// Test API call to verify credentials work
	parent := fmt.Sprintf("projects/%s", g.config.ProjectID)
	req := &secretmanagerpb.ListSecretsRequest{
		Parent:   parent,
		PageSize: 1, // Just get one secret to test access
	}

	_, err := g.client.ListSecrets(ctx, req).Next()
	if err != nil {
		if status.Code(err) == codes.PermissionDenied {
			return fmt.Errorf("GCP credentials do not have permission to access Secret Manager")
		}
		if status.Code(err) == codes.Unauthenticated {
			return fmt.Errorf("GCP credentials are invalid")
		}
		return fmt.Errorf("failed to validate GCP credentials: %w", err)
	}

	logEntry.Info("GCP credentials validated successfully")
	return nil
}

// GetProviderName returns the provider name
func (g *GCPProvider) GetProviderName() string {
	return "gcp"
}

// createOrUpdateSecret creates or updates a GCP secret
func (g *GCPProvider) createOrUpdateSecret(ctx context.Context, secretName, secretValue string) error {
	logEntry := g.logger.WithFields(logrus.Fields{
		"secret_name": secretName,
		"location":    g.config.SecretManagerLocation,
		"replication": g.config.Replication,
	})

	logEntry.Info("Creating or updating GCP secret")

	// Apply prefix if configured
	fullSecretName := secretName
	if g.config.Prefix != "" {
		fullSecretName = fmt.Sprintf("%s_%s", g.config.Prefix, secretName)
	}

	// Sanitize secret name for GCP (only alphanumeric, hyphens, underscores)
	fullSecretName = sanitizeSecretName(fullSecretName)

	// Build parent path based on location configuration
	var parent, secretPath string
	if g.config.SecretManagerLocation != "" {
		// Regional secret
		parent = fmt.Sprintf("projects/%s/locations/%s", g.config.ProjectID, g.config.SecretManagerLocation)
		secretPath = fmt.Sprintf("%s/secrets/%s", parent, fullSecretName)
	} else {
		// Global secret
		parent = fmt.Sprintf("projects/%s", g.config.ProjectID)
		secretPath = fmt.Sprintf("%s/secrets/%s", parent, fullSecretName)
	}

	secretID := fullSecretName
	_, err := g.client.GetSecret(ctx, &secretmanagerpb.GetSecretRequest{
		Name: secretPath,
	})

	if err != nil {
		if status.Code(err) == codes.NotFound {
			// Secret doesn't exist, create it
			return g.createSecret(ctx, parent, secretID, secretValue)
		}
		return fmt.Errorf("failed to check if secret exists: %w", err)
	}

	// Secret exists, add a new version
	return g.addSecretVersion(ctx, secretPath, secretValue)
}

// createSecret creates a new secret in GCP Secret Manager
func (g *GCPProvider) createSecret(ctx context.Context, parent, secretID, secretValue string) error {
	logEntry := g.logger.WithFields(logrus.Fields{
		"secret_id":   secretID,
		"location":    g.config.SecretManagerLocation,
		"replication": g.config.Replication,
	})

	logEntry.Info("Creating new GCP secret")

	// Note: Location validation will be handled by GCP client
	// If location is invalid, GCP will return an appropriate error

	// Create the secret with appropriate replication
	secret := g.buildSecretReplication()

	req := &secretmanagerpb.CreateSecretRequest{
		Parent:   parent,
		SecretId: secretID,
		Secret:   secret,
	}

	createdSecret, err := g.client.CreateSecret(ctx, req)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to create secret")

		// Check if it's a location-related error
		if status.Code(err) == codes.InvalidArgument {
			// Check if the error message indicates location issues
			if strings.Contains(err.Error(), "location") || strings.Contains(err.Error(), "region") {
				return appErrors.ErrGCPInvalidLocation
			}
		}

		return fmt.Errorf("failed to create secret: %w", err)
	}

	logEntry.WithField("secret_path", createdSecret.Name).Info("Successfully created GCP secret")

	// Add the secret version
	return g.addSecretVersion(ctx, createdSecret.Name, secretValue)
}

// addSecretVersion adds a new version to an existing secret
func (g *GCPProvider) addSecretVersion(ctx context.Context, secretPath, secretValue string) error {
	logEntry := g.logger.WithFields(logrus.Fields{
		"secret_path":            secretPath,
		"disable_older_versions": g.config.DisableOlderVersions,
	})

	logEntry.Info("Adding new secret version")

	// If disable_older_versions is enabled, disable older versions first
	if g.config.DisableOlderVersions {
		if err := g.disableOlderVersions(ctx, secretPath); err != nil {
			logEntry.WithField("error", err.Error()).Warn("Failed to disable older versions, continuing with new version")
			// Continue with adding new version even if disabling fails
		}
	}

	req := &secretmanagerpb.AddSecretVersionRequest{
		Parent: secretPath,
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(secretValue),
		},
	}

	_, err := g.client.AddSecretVersion(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to add secret version: %w", err)
	}

	logEntry.Info("Successfully added new secret version")
	return nil
}

// sanitizeSecretName sanitizes the secret name for GCP Secret Manager
func sanitizeSecretName(name string) string {
	// Replace invalid characters with underscores
	invalidChars := []string{" ", ".", "/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range invalidChars {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Ensure it starts with a letter
	if len(result) > 0 && (result[0] < 'a' || result[0] > 'z') && (result[0] < 'A' || result[0] > 'Z') {
		result = "secret_" + result
	}

	// Limit length to 255 characters
	if len(result) > 255 {
		result = result[:255]
	}

	return result
}

// disableOlderVersions disables the current version before creating a new one
func (g *GCPProvider) disableOlderVersions(ctx context.Context, secretPath string) error {
	logEntry := g.logger.WithField("secret_path", secretPath)
	logEntry.Info("Disabling current version before creating new one")

	// List all versions of the secret
	req := &secretmanagerpb.ListSecretVersionsRequest{
		Parent: secretPath,
	}

	it := g.client.ListSecretVersions(ctx, req)
	var versions []*secretmanagerpb.SecretVersion

	for {
		version, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list secret versions: %w", err)
		}
		versions = append(versions, version)
	}

	logEntry.WithField("total_versions", len(versions)).Info("Found secret versions")

	if len(versions) == 0 {
		// No versions exist, nothing to disable
		logEntry.Info("No current version to disable")
		return nil
	}

	// Find the current (latest) enabled version
	var currentVersion *secretmanagerpb.SecretVersion
	for _, version := range versions {
		if version.State == secretmanagerpb.SecretVersion_ENABLED {
			if currentVersion == nil || version.CreateTime.AsTime().After(currentVersion.CreateTime.AsTime()) {
				currentVersion = version
			}
		}
	}

	if currentVersion == nil {
		logEntry.Info("No enabled current version to disable")
		return nil
	}

	logEntry.WithFields(logrus.Fields{
		"current_version_name": currentVersion.Name,
		"current_create_time":  currentVersion.CreateTime.AsTime(),
	}).Info("Found current version to disable")

	// Disable the current version before creating a new one
	disableReq := &secretmanagerpb.DisableSecretVersionRequest{
		Name: currentVersion.Name,
	}

	_, err := g.client.DisableSecretVersion(ctx, disableReq)
	if err != nil {
		logEntry.WithFields(logrus.Fields{
			"version_name": currentVersion.Name,
			"error":        err.Error(),
		}).Warn("Failed to disable current version")
		return fmt.Errorf("failed to disable current version: %w", err)
	}

	logEntry.WithField("version_name", currentVersion.Name).Info("Disabled current version")
	logEntry.Info("Completed disabling current version")
	return nil
}
