package provider

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// ProviderType represents supported secret sync providers
type ProviderType string

const (
	ProviderGitHub ProviderType = "github"
	ProviderGCP    ProviderType = "gcp"
	ProviderAzure  ProviderType = "azure"
)

// ProviderFactoryImpl implements ProviderFactory interface
type ProviderFactoryImpl struct {
	logger *logrus.Logger
}

// NewProviderFactory creates a new provider factory instance
func NewProviderFactory(logger *logrus.Logger) ProviderFactory {
	return &ProviderFactoryImpl{
		logger: logger,
	}
}

// CreateProvider creates a new provider sync instance for the given provider type
func (f *ProviderFactoryImpl) CreateProvider(providerType ProviderType, credentials map[string]interface{}, config map[string]interface{}) (ProviderSyncer, error) {
	switch providerType {
	case ProviderGitHub:
		return NewGitHubProvider(credentials, config, f.logger)
	case ProviderGCP:
		return NewGCPProvider(credentials, config, f.logger)
	case ProviderAzure:
		return NewAzureProvider(credentials, config, f.logger)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// GetSupportedProviders returns a list of supported provider types
func (f *ProviderFactoryImpl) GetSupportedProviders() []ProviderType {
	return []ProviderType{
		ProviderGitHub,
		ProviderGCP,
		ProviderAzure,
	}
}
