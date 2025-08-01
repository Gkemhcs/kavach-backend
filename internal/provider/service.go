package provider

import (
	"context"
	"database/sql"
	"encoding/json"

	appErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	providerdb "github.com/Gkemhcs/kavach-backend/internal/provider/gen"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ProviderService handles business logic for provider credentials
type ProviderService struct {
	providerRepo providerdb.Querier
	factory      ProviderFactory
	logger       *logrus.Logger
	encryptor    *utils.Encryptor
}

// NewProviderService creates a new ProviderService instance
func NewProviderService(providerRepo providerdb.Querier, factory ProviderFactory, logger *logrus.Logger, encryptor *utils.Encryptor) *ProviderService {

	return &ProviderService{
		providerRepo: providerRepo,
		factory:      factory,
		logger:       logger,
		encryptor:    encryptor,
	}
}

// CreateProviderCredential creates a new provider credential for an environment
func (s *ProviderService) CreateProviderCredential(ctx context.Context, environmentID, userID string, req CreateProviderCredentialRequest) (*ProviderCredentialResponse, error) {
	logEntry := s.logger.WithFields(logrus.Fields{
		"method":         "CreateProviderCredential",
		"environment_id": environmentID,
		"provider":       req.Provider,
	})

	logEntry.Info("Creating provider credential")

	// Validate provider type
	if !s.isValidProvider(req.Provider) {
		logEntry.Error("Invalid provider type")
		return nil, appErrors.ErrInvalidProviderType
	}

	// Parse environment ID
	envUUID, err := uuid.Parse(environmentID)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Invalid environment ID")
		return nil, appErrors.ErrInternalServer
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Invalid user ID")
		return nil, appErrors.ErrInternalServer
	}

	// Validate credentials and config based on provider type
	if err := s.validateProviderData(req.Provider, req.Credentials, req.Config); err != nil {
		logEntry.WithField("error", err.Error()).Error("Invalid provider data")
		return nil, appErrors.ErrInvalidProviderData
	}

	// Check if provider credential already exists
	existing, err := s.providerRepo.GetProviderCredential(ctx, providerdb.GetProviderCredentialParams{
		EnvironmentID: envUUID,
		Provider:      string(req.Provider),
	})
	if err == nil && existing.ID != uuid.Nil {
		logEntry.Error("Provider credential already exists")
		return nil, appErrors.ErrProviderCredentialExists
	}

	// Convert map[string]interface{} to json.RawMessage
	credentialsJSON, err := json.Marshal(req.Credentials)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to marshal credentials")
		return nil, appErrors.ErrProviderCredentialCreateFailed
	}

	// Encrypt credentials before storing
	encryptedCredentials, err := s.encryptor.Encrypt(credentialsJSON)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to encrypt credentials")
		return nil, appErrors.ErrProviderEncryptionFailed
	}

	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to marshal config")
		return nil, appErrors.ErrProviderCredentialCreateFailed
	}

	// Create provider credential
	credential, err := s.providerRepo.CreateProviderCredential(ctx, providerdb.CreateProviderCredentialParams{
		EnvironmentID: envUUID,
		Provider:      string(req.Provider),
		Credentials:   json.RawMessage(encryptedCredentials),
		Config:        configJSON,
		CreatedBy:     userUUID,
	})
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to create provider credential")
		return nil, appErrors.ErrProviderCredentialCreateFailed
	}

	// Convert json.RawMessage back to map[string]interface{} for response
	var configMap map[string]interface{}
	if err := json.Unmarshal(credential.Config, &configMap); err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to unmarshal config")
		return nil, appErrors.ErrProviderCredentialCreateFailed
	}

	logEntry.WithField("credential_id", credential.ID).Info("Successfully created provider credential")

	return &ProviderCredentialResponse{
		ID:            credential.ID.String(),
		EnvironmentID: credential.EnvironmentID,
		Provider:      ProviderType(credential.Provider),
		Config:        configMap,
		CreatedAt:     credential.CreatedAt,
		UpdatedAt:     credential.UpdatedAt,
	}, nil
}

// GetProviderCredential retrieves a provider credential by environment ID and provider
func (s *ProviderService) GetProviderCredential(ctx context.Context, environmentID, provider string) (*ProviderCredentialResponse, error) {
	logEntry := s.logger.WithFields(logrus.Fields{
		"method":         "GetProviderCredential",
		"environment_id": environmentID,
		"provider":       provider,
	})

	logEntry.Info("Retrieving provider credential")

	envUUID, err := uuid.Parse(environmentID)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Invalid environment ID")
		return nil, appErrors.ErrInternalServer
	}

	credential, err := s.providerRepo.GetProviderCredential(ctx, providerdb.GetProviderCredentialParams{
		EnvironmentID: envUUID,
		Provider:      provider,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			logEntry.Error("Provider credential not found")
			return nil, appErrors.ErrProviderCredentialNotFound
		}
		logEntry.WithField("error", err.Error()).Error("Failed to retrieve provider credential")
		return nil, appErrors.ErrProviderCredentialGetFailed
	}

	// Convert json.RawMessage to map[string]interface{}
	var configMap map[string]interface{}
	if err := json.Unmarshal(credential.Config, &configMap); err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to unmarshal config")
		return nil, appErrors.ErrProviderCredentialGetFailed
	}

	logEntry.Info("Successfully retrieved provider credential")

	return &ProviderCredentialResponse{
		ID:            credential.ID.String(),
		EnvironmentID: credential.EnvironmentID,
		Provider:      ProviderType(credential.Provider),
		Config:        configMap,
		CreatedAt:     credential.CreatedAt,
		UpdatedAt:     credential.UpdatedAt,
	}, nil
}

// ListProviderCredentials lists all provider credentials for an environment
func (s *ProviderService) ListProviderCredentials(ctx context.Context, environmentID string) ([]ProviderCredentialResponse, error) {
	logEntry := s.logger.WithFields(logrus.Fields{
		"method":         "ListProviderCredentials",
		"environment_id": environmentID,
	})

	logEntry.Info("Listing provider credentials")

	envUUID, err := uuid.Parse(environmentID)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Invalid environment ID")
		return nil, appErrors.ErrInternalServer
	}

	credentials, err := s.providerRepo.ListProviderCredentials(ctx, envUUID)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to list provider credentials")
		return nil, appErrors.ErrProviderCredentialListFailed
	}

	var response []ProviderCredentialResponse
	for _, cred := range credentials {
		// Convert json.RawMessage to map[string]interface{}
		var configMap map[string]interface{}
		if err := json.Unmarshal(cred.Config, &configMap); err != nil {
			logEntry.WithField("error", err.Error()).Error("Failed to unmarshal config")
			return nil, appErrors.ErrProviderCredentialListFailed
		}

		response = append(response, ProviderCredentialResponse{
			ID:            cred.ID.String(),
			EnvironmentID: cred.EnvironmentID,
			Provider:      ProviderType(cred.Provider),
			Config:        configMap,
			CreatedAt:     cred.CreatedAt,
			UpdatedAt:     cred.UpdatedAt,
		})
	}

	logEntry.WithField("count", len(response)).Info("Successfully listed provider credentials")

	return response, nil
}

// UpdateProviderCredential updates an existing provider credential
func (s *ProviderService) UpdateProviderCredential(ctx context.Context, environmentID, provider string, req UpdateProviderCredentialRequest) (*ProviderCredentialResponse, error) {
	logEntry := s.logger.WithFields(logrus.Fields{
		"method":         "UpdateProviderCredential",
		"environment_id": environmentID,
		"provider":       provider,
	})

	logEntry.Info("Updating provider credential")

	envUUID, err := uuid.Parse(environmentID)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Invalid environment ID")
		return nil, appErrors.ErrInternalServer
	}

	// Validate provider type
	providerType := ProviderType(provider)
	if !s.isValidProvider(providerType) {
		logEntry.Error("Invalid provider type")
		return nil, appErrors.ErrInvalidProviderType
	}

	// Validate credentials and config
	if err := s.validateProviderData(providerType, req.Credentials, req.Config); err != nil {
		logEntry.WithField("error", err.Error()).Error("Invalid provider data")
		return nil, appErrors.ErrInvalidProviderData
	}

	// Convert map[string]interface{} to json.RawMessage
	credentialsJSON, err := json.Marshal(req.Credentials)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to marshal credentials")
		return nil, appErrors.ErrProviderCredentialUpdateFailed
	}

	// Encrypt credentials before storing
	encryptedCredentials, err := s.encryptor.Encrypt(credentialsJSON)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to encrypt credentials")
		return nil, appErrors.ErrProviderEncryptionFailed
	}

	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to marshal config")
		return nil, appErrors.ErrProviderCredentialUpdateFailed
	}

	credential, err := s.providerRepo.UpdateProviderCredential(ctx, providerdb.UpdateProviderCredentialParams{
		EnvironmentID: envUUID,
		Provider:      provider,
		Credentials:   json.RawMessage(encryptedCredentials),
		Config:        configJSON,
	})
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to update provider credential")
		return nil, appErrors.ErrProviderCredentialUpdateFailed
	}

	// Convert json.RawMessage back to map[string]interface{} for response
	var configMap map[string]interface{}
	if err := json.Unmarshal(credential.Config, &configMap); err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to unmarshal config")
		return nil, appErrors.ErrProviderCredentialUpdateFailed
	}

	logEntry.Info("Successfully updated provider credential")

	return &ProviderCredentialResponse{
		ID:            credential.ID.String(),
		EnvironmentID: credential.EnvironmentID,
		Provider:      ProviderType(credential.Provider),
		Config:        configMap,
		CreatedAt:     credential.CreatedAt,
		UpdatedAt:     credential.UpdatedAt,
	}, nil
}

// DeleteProviderCredential deletes a provider credential
func (s *ProviderService) DeleteProviderCredential(ctx context.Context, environmentID, provider string) error {
	logEntry := s.logger.WithFields(logrus.Fields{
		"method":         "DeleteProviderCredential",
		"environment_id": environmentID,
		"provider":       provider,
	})

	logEntry.Info("Deleting provider credential")

	envUUID, err := uuid.Parse(environmentID)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Invalid environment ID")
		return appErrors.ErrInternalServer
	}

	err = s.providerRepo.DeleteProviderCredential(ctx, providerdb.DeleteProviderCredentialParams{
		EnvironmentID: envUUID,
		Provider:      provider,
	})
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to delete provider credential")
		return appErrors.ErrProviderCredentialDeleteFailed
	}

	logEntry.Info("Successfully deleted provider credential")
	return nil
}

// GetProviderSyncer creates a ProviderSyncer instance for the given environment and provider
func (s *ProviderService) GetProviderSyncer(ctx context.Context, environmentID, provider string) (ProviderSyncer, error) {
	logEntry := s.logger.WithFields(logrus.Fields{
		"method":         "GetProviderSyncer",
		"environment_id": environmentID,
		"provider":       provider,
	})

	logEntry.Info("Creating provider syncer instance")

	// Get provider credential from database
	envUUID, err := uuid.Parse(environmentID)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Invalid environment ID")
		return nil, appErrors.ErrInternalServer
	}

	credential, err := s.providerRepo.GetProviderCredential(ctx, providerdb.GetProviderCredentialParams{
		EnvironmentID: envUUID,
		Provider:      provider,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			logEntry.Error("Provider credential not found")
			return nil, appErrors.ErrProviderCredentialNotFound
		}
		logEntry.WithField("error", err.Error()).Error("Failed to get provider credential")
		return nil, appErrors.ErrProviderCredentialGetFailed
	}

	// Decrypt credentials
	decryptedCredentialsBytes, err := s.encryptor.Decrypt(string(credential.Credentials))
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to decrypt credentials")
		return nil, appErrors.ErrProviderDecryptionFailed
	}

	// Convert json.RawMessage to map[string]interface{}
	var credentialsMap map[string]interface{}
	if err := json.Unmarshal(decryptedCredentialsBytes, &credentialsMap); err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to unmarshal credentials")
		return nil, appErrors.ErrProviderCredentialGetFailed
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal(credential.Config, &configMap); err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to unmarshal config")
		return nil, appErrors.ErrProviderCredentialGetFailed
	}

	// Create provider syncer instance using factory
	providerSyncer, err := s.factory.CreateProvider(ProviderType(credential.Provider), credentialsMap, configMap)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to create provider syncer")
		return nil, appErrors.ErrProviderSyncFailed
	}

	logEntry.Info("Successfully created provider syncer instance")
	return providerSyncer, nil
}

// Helper methods

func (s *ProviderService) isValidProvider(provider ProviderType) bool {
	supported := s.factory.GetSupportedProviders()
	for _, p := range supported {
		if p == provider {
			return true
		}
	}
	return false
}

func (s *ProviderService) validateProviderData(provider ProviderType, credentials, config map[string]interface{}) error {
	switch provider {
	case ProviderGitHub:
		return s.validateGitHubData(credentials, config)
	case ProviderGCP:
		return s.validateGCPData(credentials, config)
	case ProviderAzure:
		return s.validateAzureData(credentials, config)
	default:
		return appErrors.ErrInvalidProviderType
	}
}

func (s *ProviderService) validateGitHubData(credentials, config map[string]interface{}) error {
	// Validate credentials
	if token, ok := credentials["token"].(string); !ok || token == "" {
		return appErrors.ErrProviderCredentialValidationFailed
	}

	// Validate config
	if owner, ok := config["owner"].(string); !ok || owner == "" {
		return appErrors.ErrProviderCredentialValidationFailed
	}
	if repo, ok := config["repository"].(string); !ok || repo == "" {
		return appErrors.ErrProviderCredentialValidationFailed
	}

	return nil
}

func (s *ProviderService) validateGCPData(credentials, config map[string]interface{}) error {
	// Validate credentials
	requiredCredFields := []string{"type", "project_id", "private_key_id", "private_key", "client_email"}
	for _, field := range requiredCredFields {
		if val, ok := credentials[field].(string); !ok || val == "" {
			return appErrors.ErrProviderCredentialValidationFailed
		}
	}

	// Validate config
	if projectID, ok := config["project_id"].(string); !ok || projectID == "" {
		return appErrors.ErrProviderCredentialValidationFailed
	}

	return nil
}

func (s *ProviderService) validateAzureData(credentials, config map[string]interface{}) error {
	// Validate credentials
	requiredCredFields := []string{"tenant_id", "client_id", "client_secret"}
	for _, field := range requiredCredFields {
		if val, ok := credentials[field].(string); !ok || val == "" {
			return appErrors.ErrInvalidProviderData
		}
	}

	// Validate config
	requiredConfigFields := []string{"subscription_id", "resource_group", "key_vault_name"}
	for _, field := range requiredConfigFields {
		if val, ok := config[field].(string); !ok || val == "" {
			return appErrors.ErrInvalidProviderData
		}
	}

	return nil
}
