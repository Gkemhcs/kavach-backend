package provider

import (
	"context"
)

// Secret represents a secret to be synced to a provider
type Secret struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SyncResult represents the result of syncing a single secret
type SyncResult struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ProviderConfig represents the configuration and credentials for a provider
type ProviderConfig struct {
	Provider    ProviderType           `json:"provider"`
	Credentials map[string]interface{} `json:"credentials"`
	Config      map[string]interface{} `json:"config"`
}

// ProviderGetter defines the interface for retrieving provider configurations
// This interface will be implemented by the provider service and injected into the secret service
type ProviderGetter interface {
	// GetProviderConfig retrieves and decrypts provider configuration for a given environment and provider
	GetProviderConfig(ctx context.Context, environmentID string, provider ProviderType) (*ProviderConfig, error)
}

// ProviderSyncer defines the interface for syncing secrets to external providers
// This is the Strategy interface that different providers implement
type ProviderSyncer interface {
	// Sync syncs a list of secrets to the provider
	// Returns a list of sync results for each secret
	Sync(ctx context.Context, secrets []Secret) ([]SyncResult, error)

	// ValidateCredentials validates the provider credentials
	ValidateCredentials(ctx context.Context) error

	// GetProviderName returns the name of the provider
	GetProviderName() string
}

// ProviderFactory creates provider sync instances
type ProviderFactory interface {
	// CreateProvider creates a new provider sync instance for the given provider type
	CreateProvider(providerType ProviderType, credentials map[string]interface{}, config map[string]interface{}) (ProviderSyncer, error)

	// GetSupportedProviders returns a list of supported provider types
	GetSupportedProviders() []ProviderType
}
