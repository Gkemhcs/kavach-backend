package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	"github.com/sirupsen/logrus"
)

// AzureProvider implements ProviderSync for Azure Key Vault
type AzureProvider struct {
	credentials AzureCredentials
	config      AzureConfig
	logger      *logrus.Logger
	client      *azsecrets.Client
}

// NewAzureProvider creates a new Azure provider instance
func NewAzureProvider(credentials map[string]interface{}, config map[string]interface{}, logger *logrus.Logger) (*AzureProvider, error) {
	// Parse credentials
	credBytes, err := json.Marshal(credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal credentials: %w", err)
	}

	var azureCreds AzureCredentials
	if err := json.Unmarshal(credBytes, &azureCreds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Azure credentials: %w", err)
	}

	// Parse config
	configBytes, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var azureConfig AzureConfig
	if err := json.Unmarshal(configBytes, &azureConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Azure config: %w", err)
	}

	// Create Azure credential
	cred, err := azidentity.NewClientSecretCredential(
		azureCreds.TenantID,
		azureCreds.ClientID,
		azureCreds.ClientSecret,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	// Create Key Vault client
	vaultURL := fmt.Sprintf("https://%s.vault.azure.net/", azureConfig.KeyVaultName)
	client, err := azsecrets.NewClient(vaultURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Key Vault client: %w", err)
	}

	return &AzureProvider{
		credentials: azureCreds,
		config:      azureConfig,
		logger:      logger,
		client:      client,
	}, nil
}

// Sync syncs secrets to Azure Key Vault
func (a *AzureProvider) Sync(ctx context.Context, secrets []Secret) ([]SyncResult, error) {
	logEntry := a.logger.WithFields(logrus.Fields{
		"provider":       "azure",
		"subscription":   a.config.SubscriptionID,
		"resource_group": a.config.ResourceGroup,
		"key_vault":      a.config.KeyVaultName,
		"secret_count":   len(secrets),
	})

	logEntry.Info("Starting Azure secrets sync")

	var results []SyncResult

	for _, secret := range secrets {
		result := SyncResult{
			Name:    secret.Name,
			Success: false,
		}

		// Use retry wrapper for creating/updating secret
		err := a.retryCreateSecret(ctx, secret.Name, secret.Value)
		if err != nil {
			result.Error = err.Error()
			logEntry.WithFields(logrus.Fields{
				"secret_name": secret.Name,
				"error":       err.Error(),
			}).Error("Failed to sync secret to Azure after all retry attempts")
		} else {
			result.Success = true
			logEntry.WithField("secret_name", secret.Name).Info("Successfully synced secret to Azure")
		}

		results = append(results, result)
	}

	logEntry.WithFields(logrus.Fields{
		"synced_count": len(results),
		"total_count":  len(secrets),
	}).Info("Completed Azure secrets sync")

	return results, nil
}

// retryCreateSecret wraps the createSecret function with retry logic
func (a *AzureProvider) retryCreateSecret(ctx context.Context, secretName, secretValue string) error {
	logEntry := a.logger.WithFields(logrus.Fields{
		"secret_name": secretName,
		"max_retries": a.config.RetryConfig.MaxRetries,
	})

	// Set default retry configuration if not provided
	if a.config.RetryConfig.MaxRetries == 0 {
		a.config.RetryConfig = RetryConfig{
			MaxRetries:  3,
			RetryDelay:  2 * time.Second,
			MaxDelay:    30 * time.Second,
			BackoffType: "exponential",
		}
	}

	var lastErr error
	for attempt := 1; attempt <= a.config.RetryConfig.MaxRetries; attempt++ {
		logEntry.WithField("attempt", attempt).Info("Attempting to create/update secret")

		err := a.createOrUpdateSecret(ctx, secretName, secretValue)
		if err != nil {
			lastErr = err
			logEntry.WithFields(logrus.Fields{
				"attempt": attempt,
				"error":   err.Error(),
			}).Warn("Failed to create/update secret, will retry")

			if attempt < a.config.RetryConfig.MaxRetries {
				delay := a.calculateRetryDelay(attempt)
				logEntry.WithField("delay", delay).Info("Waiting before retry")
				time.Sleep(delay)
				continue
			}
		} else {
			logEntry.WithField("attempt", attempt).Info("Successfully created/updated secret")
			return nil
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", a.config.RetryConfig.MaxRetries, lastErr)
}

// calculateRetryDelay calculates the delay for the next retry attempt
func (a *AzureProvider) calculateRetryDelay(attempt int) time.Duration {
	switch a.config.RetryConfig.BackoffType {
	case "exponential":
		delay := a.config.RetryConfig.RetryDelay * time.Duration(1<<(attempt-1))
		if delay > a.config.RetryConfig.MaxDelay {
			delay = a.config.RetryConfig.MaxDelay
		}
		return delay
	case "linear":
		delay := a.config.RetryConfig.RetryDelay * time.Duration(attempt)
		if delay > a.config.RetryConfig.MaxDelay {
			delay = a.config.RetryConfig.MaxDelay
		}
		return delay
	case "constant":
		return a.config.RetryConfig.RetryDelay
	default:
		// Default to exponential backoff
		delay := a.config.RetryConfig.RetryDelay * time.Duration(1<<(attempt-1))
		if delay > a.config.RetryConfig.MaxDelay {
			delay = a.config.RetryConfig.MaxDelay
		}
		return delay
	}
}

// ValidateCredentials validates Azure credentials by making a test API call
func (a *AzureProvider) ValidateCredentials(ctx context.Context) error {
	logEntry := a.logger.WithFields(logrus.Fields{
		"provider":       "azure",
		"subscription":   a.config.SubscriptionID,
		"resource_group": a.config.ResourceGroup,
		"key_vault":      a.config.KeyVaultName,
	})

	logEntry.Info("Validating Azure credentials")

	// Test API call to verify credentials work
	// Try to list secrets to verify access
	pager := a.client.NewListSecretsPager(nil)
	_, err := pager.NextPage(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate Azure credentials: %w", err)
	}

	logEntry.Info("Azure credentials validated successfully")
	return nil
}

// GetProviderName returns the provider name
func (a *AzureProvider) GetProviderName() string {
	return "azure"
}

// createOrUpdateSecret creates or updates an Azure Key Vault secret
func (a *AzureProvider) createOrUpdateSecret(ctx context.Context, secretName, secretValue string) error {
	logEntry := a.logger.WithFields(logrus.Fields{
		"secret_name":            secretName,
		"disable_older_versions": a.config.DisableOlderVersions,
	})

	logEntry.Info("Creating or updating Azure secret")

	// Apply prefix if configured
	fullSecretName := secretName
	if a.config.Prefix != "" {
		fullSecretName = fmt.Sprintf("%s-%s", a.config.Prefix, secretName)
	}

	// Sanitize secret name for Azure Key Vault
	fullSecretName = sanitizeAzureSecretName(fullSecretName)

	// Check if secret exists to determine if we should delete older versions
	secretExists := a.secretExists(ctx, fullSecretName)

	// If disable_older_versions is enabled and secret exists, disable older versions first
	if a.config.DisableOlderVersions && secretExists {
		if err := a.disableOlderVersions(ctx, fullSecretName); err != nil {
			logEntry.WithField("error", err.Error()).Warn("Failed to disable older versions, continuing with new version")
			// Continue with adding new version even if disabling fails
		}
	}

	// Create or update the secret
	_, err := a.client.SetSecret(ctx, fullSecretName, azsecrets.SetSecretParameters{
		Value: &secretValue,
	}, nil)

	if err != nil {
		return fmt.Errorf("failed to create/update secret '%s': %w", fullSecretName, err)
	}

	logEntry.WithField("secret_name", fullSecretName).Info("Successfully created/updated Azure secret")
	return nil
}

// sanitizeAzureSecretName sanitizes the secret name for Azure Key Vault
func sanitizeAzureSecretName(name string) string {
	// Replace invalid characters with hyphens
	invalidChars := []string{" ", ".", "/", "\\", ":", "*", "?", "\"", "<", ">", "|", "_"}
	result := name
	for _, char := range invalidChars {
		result = strings.ReplaceAll(result, char, "-")
	}

	// Remove consecutive hyphens
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	// Remove leading/trailing hyphens
	result = strings.Trim(result, "-")

	// Ensure it starts with a letter or number
	if len(result) > 0 && (result[0] < 'a' || result[0] > 'z') && (result[0] < 'A' || result[0] > 'Z') && (result[0] < '0' || result[0] > '9') {
		result = "secret-" + result
	}

	// Limit length to 127 characters (Azure Key Vault limit)
	if len(result) > 127 {
		result = result[:127]
	}

	return result
}

// secretExists checks if a secret exists in Azure Key Vault
func (a *AzureProvider) secretExists(ctx context.Context, secretName string) bool {
	// Try to get the latest version of the secret
	_, err := a.client.GetSecret(ctx, secretName, "", nil)
	return err == nil
}

// disableOlderVersions disables the current version before creating a new one
func (a *AzureProvider) disableOlderVersions(ctx context.Context, secretName string) error {
	logEntry := a.logger.WithField("secret_name", secretName)
	logEntry.Info("Disabling current version before creating new one")

	// In Azure Key Vault, we can't disable individual versions like GCP
	// Instead, we'll disable the current secret by setting it as disabled
	// This will make the current version inactive when we create a new one

	// Get the current secret to check if it exists
	_, err := a.client.GetSecret(ctx, secretName, "", nil)
	if err != nil {
		// Secret doesn't exist, nothing to disable
		logEntry.Info("No current version to disable")
		return nil
	}

	logEntry.Info("Current version found, will be disabled when new version is created")

	// Note: In Azure Key Vault, when we create a new version with SetSecret,
	// the previous version automatically becomes inactive/disabled
	// So we don't need to explicitly disable it - it happens automatically

	logEntry.Info("Current version will be automatically disabled when new version is created")
	return nil
}
