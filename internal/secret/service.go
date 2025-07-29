package secret

import (
	"context"
	"fmt"

	apiErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	secretdb "github.com/Gkemhcs/kavach-backend/internal/secret/gen"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// SecretService handles business logic for secret management
type SecretService struct {
	repo    secretdb.Querier
	encrypt *EncryptionService
	logger  *logrus.Logger
}

// NewSecretService creates a new secret service
func NewSecretService(repo secretdb.Querier, encrypt *EncryptionService, logger *logrus.Logger) *SecretService {
	return &SecretService{
		repo:    repo,
		encrypt: encrypt,
		logger:  logger,
	}
}

// CreateVersion creates a new version of secrets for an environment
func (s *SecretService) CreateVersion(ctx context.Context, environmentID string, req CreateSecretVersionRequest) (*SecretVersionResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"environment_id": environmentID,
		"secret_count":   len(req.Secrets),
		"commit_message": req.CommitMessage,
	}).Info("Creating new secret version")

	// Validate input
	if err := s.validateCreateVersionRequest(req); err != nil {
		s.logger.WithField("error", err.Error()).Error("Invalid create version request")
		return nil, err
	}

	environmentUUID, err := uuid.Parse(environmentID)
	if err != nil {
		return nil, err
	}

	// Create the version
	version, err := s.repo.CreateSecretVersion(ctx, secretdb.CreateSecretVersionParams{
		EnvironmentID: environmentUUID,
		CommitMessage: req.CommitMessage,
	})
	if err != nil {
		s.logger.WithField("error", err.Error()).Error("Failed to create secret version")
		return nil, fmt.Errorf("failed to create secret version: %w", err)
	}

	// Encrypt and store secrets
	for _, secret := range req.Secrets {
		encryptedValue, err := s.encrypt.Encrypt(secret.Value)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"error": err.Error(),
				"name":  secret.Name,
			}).Error("Failed to encrypt secret value")
			return nil, apiErrors.ErrEncryptionFailed
		}

		err = s.repo.InsertSecret(ctx, secretdb.InsertSecretParams{
			VersionID:      version.ID,
			Name:           secret.Name,
			ValueEncrypted: encryptedValue,
		})
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"error": err.Error(),
				"name":  secret.Name,
			}).Error("Failed to insert secret")
			return nil, fmt.Errorf("failed to insert secret %s: %w", secret.Name, err)
		}
	}

	s.logger.WithFields(logrus.Fields{
		"version_id":   version.ID,
		"secret_count": len(req.Secrets),
	}).Info("Successfully created secret version")

	return &SecretVersionResponse{
		ID:            version.ID,
		EnvironmentID: version.EnvironmentID,
		CommitMessage: version.CommitMessage,
		CreatedAt:     version.CreatedAt,
		SecretCount:   len(req.Secrets),
	}, nil
}

// ListVersions lists all versions for an environment
func (s *SecretService) ListVersions(ctx context.Context, environmentID string) ([]SecretVersionResponse, error) {
	s.logger.WithField("environment_id", environmentID).Info("Listing secret versions")

	environmentUUID, err := uuid.Parse(environmentID)
	if err != nil {
		return nil, err
	}

	versions, err := s.repo.ListSecretVersions(ctx, environmentUUID)
	if err != nil {
		s.logger.WithField("error", err.Error()).Error("Failed to list secret versions")
		return nil, fmt.Errorf("failed to list secret versions: %w", err)
	}

	// Convert to response format
	responses := make([]SecretVersionResponse, len(versions))
	for i, version := range versions {
		responses[i] = SecretVersionResponse{
			ID:            version.ID,
			EnvironmentID: version.EnvironmentID,
			CommitMessage: version.CommitMessage,
			CreatedAt:     version.CreatedAt,
			// Note: SecretCount would need a separate query to get actual count
		}
	}

	s.logger.WithField("version_count", len(versions)).Info("Successfully listed secret versions")
	return responses, nil
}

// GetVersionDetails gets detailed information about a specific version including secrets
func (s *SecretService) GetVersionDetails(ctx context.Context, versionID string) (*SecretVersionDetailResponse, error) {
	s.logger.WithField("version_id", versionID).Info("Getting version details")

	// Get version info
	version, err := s.repo.GetSecretVersion(ctx, versionID)
	if err != nil {
		s.logger.WithField("error", err.Error()).Error("Failed to get secret version")
		return nil, apiErrors.ErrSecretVersionNotFound
	}

	// Get secrets for this version
	secrets, err := s.repo.GetSecretsForVersion(ctx, versionID)
	if err != nil {
		s.logger.WithField("error", err.Error()).Error("Failed to get secrets for version")
		return nil, apiErrors.ErrSecretNotFound
	}

	// Decrypt secrets
	decryptedSecrets := make([]SecretWithValue, len(secrets))
	for i, secret := range secrets {
		decryptedValue, err := s.encrypt.Decrypt(secret.ValueEncrypted)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"error": err.Error(),
				"name":  secret.Name,
			}).Error("Failed to decrypt secret value")
			return nil, err
		}

		decryptedSecrets[i] = SecretWithValue{
			Name:  secret.Name,
			Value: decryptedValue,
		}
	}

	s.logger.WithFields(logrus.Fields{
		"version_id":   versionID,
		"secret_count": len(decryptedSecrets),
	}).Info("Successfully retrieved version details")

	return &SecretVersionDetailResponse{
		ID:            version.ID,
		EnvironmentID: version.EnvironmentID,
		CommitMessage: version.CommitMessage,
		CreatedAt:     version.CreatedAt,
		Secrets:       decryptedSecrets,
	}, nil
}

// RollbackToVersion creates a new version by copying secrets from a previous version
func (s *SecretService) RollbackToVersion(ctx context.Context, environmentID string, req RollbackRequest) (*SecretVersionResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"environment_id": environmentID,
		"target_version": req.VersionID,
		"commit_message": req.CommitMessage,
	}).Info("Rolling back to previous version")

	environmentUUID, err := uuid.Parse(environmentID)
	if err != nil {
		return nil, err
	}

	// Verify the target version exists
	targetVersion, err := s.repo.GetSecretVersion(ctx, req.VersionID)
	if err != nil {
		s.logger.WithField("error", err.Error()).Error("Target version not found")
		return nil, apiErrors.ErrTargetSecretVersionNotFound
	}

	// Verify the target version belongs to the same environment
	if targetVersion.EnvironmentID != environmentUUID {
		s.logger.Error("Target version does not belong to the specified environment")
		return nil, apiErrors.ErrEnvironmentsMisMatch
	}

	// Create new version
	newVersion, err := s.repo.CreateSecretVersion(ctx, secretdb.CreateSecretVersionParams{
		EnvironmentID: environmentUUID,
		CommitMessage: req.CommitMessage,
	})
	if err != nil {
		s.logger.WithField("error", err.Error()).Error("Failed to create rollback version")
		return nil, apiErrors.ErrRollbackFailed
	}

	// Copy secrets from target version to new version
	err = s.repo.RollbackSecretsToVersion(ctx, secretdb.RollbackSecretsToVersionParams{
		Column1:   newVersion.ID,
		VersionID: req.VersionID,
	})
	if err != nil {
		s.logger.WithField("error", err.Error()).Error("Failed to copy secrets for rollback")
		return nil, apiErrors.ErrCopySecretsFailed
	}

	// Get secret count for response
	secrets, err := s.repo.GetSecretsForVersion(ctx, newVersion.ID)
	if err != nil {
		s.logger.WithField("error", err.Error()).Error("Failed to get secret count for rollback")
		return nil, fmt.Errorf("failed to get secret count for rollback: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"new_version_id": newVersion.ID,
		"target_version": req.VersionID,
		"secret_count":   len(secrets),
	}).Info("Successfully rolled back to previous version")

	return &SecretVersionResponse{
		ID:            newVersion.ID,
		EnvironmentID: newVersion.EnvironmentID,
		CommitMessage: newVersion.CommitMessage,
		CreatedAt:     newVersion.CreatedAt,
		SecretCount:   len(secrets),
	}, nil
}

// GetVersionDiff gets the differences between two versions
func (s *SecretService) GetVersionDiff(ctx context.Context, fromVersionID, toVersionID string) (*SecretDiffResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"from_version": fromVersionID,
		"to_version":   toVersionID,
	}).Info("Getting version diff")

	// Verify both versions exist
	fromVersion, err := s.repo.GetSecretVersion(ctx, fromVersionID)
	if err != nil {
		s.logger.WithField("error", err.Error()).Error("From version not found")
		return nil, fmt.Errorf("from version not found: %w", err)
	}

	toVersion, err := s.repo.GetSecretVersion(ctx, toVersionID)
	if err != nil {
		s.logger.WithField("error", err.Error()).Error("To version not found")
		return nil, fmt.Errorf("to version not found: %w", err)
	}

	// Verify both versions belong to the same environment
	if fromVersion.EnvironmentID != toVersion.EnvironmentID {
		s.logger.Error("Versions do not belong to the same environment")
		return nil, fmt.Errorf("versions do not belong to the same environment")
	}

	// Get diff data
	diffData, err := s.repo.DiffSecretVersions(ctx, secretdb.DiffSecretVersionsParams{
		VersionID:   fromVersionID,
		VersionID_2: toVersionID,
	})
	if err != nil {
		s.logger.WithField("error", err.Error()).Error("Failed to get version diff")
		return nil, fmt.Errorf("failed to get version diff: %w", err)
	}

	// Process diff data
	changes := make([]SecretDiffChange, 0, len(diffData))
	for _, diff := range diffData {
		change := SecretDiffChange{Name: diff.Name}

		// Decrypt values if they exist
		if diff.ValueV1 != nil {
			decryptedV1, err := s.encrypt.Decrypt(diff.ValueV1)
			if err != nil {
				s.logger.WithField("error", err.Error()).Error("Failed to decrypt v1 value")
				return nil, fmt.Errorf("failed to decrypt v1 value for %s: %w", diff.Name, err)
			}
			change.OldValue = decryptedV1
		}

		if diff.ValueV2 != nil {
			decryptedV2, err := s.encrypt.Decrypt(diff.ValueV2)
			if err != nil {
				s.logger.WithField("error", err.Error()).Error("Failed to decrypt v2 value")
				return nil, err
			}
			change.NewValue = decryptedV2
		}

		// Determine change type
		if diff.ValueV1 == nil && diff.ValueV2 != nil {
			change.Type = "added"
		} else if diff.ValueV1 != nil && diff.ValueV2 == nil {
			change.Type = "removed"
		} else if diff.ValueV1 != nil && diff.ValueV2 != nil {
			// Both values exist, check if they're actually different
			if change.OldValue == change.NewValue {
				change.Type = "no_change"
			} else {
				change.Type = "modified"
			}
		}

		changes = append(changes, change)
	}

	s.logger.WithField("change_count", len(changes)).Info("Successfully generated version diff")

	return &SecretDiffResponse{
		FromVersion: fromVersionID,
		ToVersion:   toVersionID,
		Changes:     changes,
	}, nil
}

// validateCreateVersionRequest validates the create version request
func (s *SecretService) validateCreateVersionRequest(req CreateSecretVersionRequest) error {
	// Validate commit message
	if req.CommitMessage == "" {
		return fmt.Errorf("commit message cannot be empty")
	}

	// Validate secrets
	if len(req.Secrets) == 0 {
		return apiErrors.ErrEmptySecrets
	}

	const maxSecrets = 1000
	if len(req.Secrets) > maxSecrets {
		return apiErrors.ErrTooManySecrets
	}

	// Validate each secret
	seenNames := make(map[string]bool)
	for _, secret := range req.Secrets {
		// Check for duplicate names
		if seenNames[secret.Name] {
			return fmt.Errorf("duplicate secret name: %s", secret.Name)
		}
		seenNames[secret.Name] = true

		// Validate secret name
		if err := s.encrypt.ValidateSecretName(secret.Name); err != nil {
			return err
		}

		// Validate secret value
		if err := s.encrypt.ValidateSecretValue(secret.Value); err != nil {
			return err
		}
	}

	return nil
}
