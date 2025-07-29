package provider

import (
	"time"

	"github.com/google/uuid"
)


// CreateProviderCredentialRequest represents the request to create a new provider credential
type CreateProviderCredentialRequest struct {
	Provider    ProviderType           `json:"provider" binding:"required"`
	Credentials map[string]interface{} `json:"credentials" binding:"required"`
	Config      map[string]interface{} `json:"config" binding:"required"`
}

// UpdateProviderCredentialRequest represents the request to update an existing provider credential
type UpdateProviderCredentialRequest struct {
	Credentials map[string]interface{} `json:"credentials" binding:"required"`
	Config      map[string]interface{} `json:"config" binding:"required"`
}

// ProviderCredentialResponse represents the response for provider credential data
type ProviderCredentialResponse struct {
	ID            string                 `json:"id"`
	EnvironmentID uuid.UUID              `json:"environment_id"`
	Provider      ProviderType           `json:"provider"`
	Credentials   map[string]interface{} `json:"credentials,omitempty"` // Decrypted credentials
	Config        map[string]interface{} `json:"config"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// GitHubCredentials represents GitHub Personal Access Token credentials
type GitHubCredentials struct {
	Token string `json:"token" binding:"required"`
}

// GitHubConfig represents GitHub-specific configuration
type GitHubConfig struct {
	Owner            string `json:"owner" binding:"required"`
	Repository       string `json:"repository" binding:"required"`
	Environment      string `json:"environment,omitempty"`
	SecretVisibility string `json:"secret_visibility,omitempty"` // all, selected, private
}

// GCPCredentials represents GCP Service Account key credentials
type GCPCredentials struct {
	Type                    string `json:"type" binding:"required"`
	ProjectID               string `json:"project_id" binding:"required"`
	PrivateKeyID            string `json:"private_key_id" binding:"required"`
	PrivateKey              string `json:"private_key" binding:"required"`
	ClientEmail             string `json:"client_email" binding:"required"`
	ClientID                string `json:"client_id,omitempty"`
	AuthURI                 string `json:"auth_uri,omitempty"`
	TokenURI                string `json:"token_uri,omitempty"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url,omitempty"`
	ClientX509CertURL       string `json:"client_x509_cert_url,omitempty"`
}

// GCPConfig represents GCP-specific configuration
type GCPConfig struct {
	ProjectID             string `json:"project_id" binding:"required"`
	SecretManagerLocation string `json:"secret_manager_location,omitempty"`
	Prefix                string `json:"prefix,omitempty"`
	Replication           string `json:"replication,omitempty"`
}

// AzureCredentials represents Azure Service Principal credentials
type AzureCredentials struct {
	TenantID     string `json:"tenant_id" binding:"required"`
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret" binding:"required"`
}

// AzureConfig represents Azure-specific configuration
type AzureConfig struct {
	SubscriptionID string `json:"subscription_id" binding:"required"`
	ResourceGroup  string `json:"resource_group" binding:"required"`
	KeyVaultName   string `json:"key_vault_name" binding:"required"`
	Prefix         string `json:"prefix,omitempty"`
}

// SyncRequest represents the request to sync secrets to a provider
type SyncRequest struct {
	Provider  ProviderType `json:"provider" binding:"required"`
	VersionID string       `json:"version_id,omitempty"` // Optional: sync specific version, otherwise latest
}

// SyncResponse represents the response for a sync operation
type SyncResponse struct {
	Provider    ProviderType `json:"provider"`
	Status      string       `json:"status"` // success, failed, partial
	Message     string       `json:"message"`
	SyncedCount int          `json:"synced_count"`
	FailedCount int          `json:"failed_count,omitempty"`
	Errors      []string     `json:"errors,omitempty"`
}
