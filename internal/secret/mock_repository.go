package secret

import (
	"context"

	secretdb "github.com/Gkemhcs/kavach-backend/internal/secret/gen"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockSecretRepository is a mock implementation of the secretdb.Querier interface
type MockSecretRepository struct {
	mock.Mock
}

// CreateSecretVersion mocks the CreateSecretVersion method
func (m *MockSecretRepository) CreateSecretVersion(ctx context.Context, arg secretdb.CreateSecretVersionParams) (secretdb.SecretVersion, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return secretdb.SecretVersion{}, args.Error(1)
	}
	return args.Get(0).(secretdb.SecretVersion), args.Error(1)
}

// DiffSecretVersions mocks the DiffSecretVersions method
func (m *MockSecretRepository) DiffSecretVersions(ctx context.Context, arg secretdb.DiffSecretVersionsParams) ([]secretdb.DiffSecretVersionsRow, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]secretdb.DiffSecretVersionsRow), args.Error(1)
}

// GetSecretVersion mocks the GetSecretVersion method
func (m *MockSecretRepository) GetSecretVersion(ctx context.Context, id string) (secretdb.SecretVersion, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return secretdb.SecretVersion{}, args.Error(1)
	}
	return args.Get(0).(secretdb.SecretVersion), args.Error(1)
}

// GetSecretsForVersion mocks the GetSecretsForVersion method
func (m *MockSecretRepository) GetSecretsForVersion(ctx context.Context, versionID string) ([]secretdb.GetSecretsForVersionRow, error) {
	args := m.Called(ctx, versionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]secretdb.GetSecretsForVersionRow), args.Error(1)
}

// InsertSecret mocks the InsertSecret method
func (m *MockSecretRepository) InsertSecret(ctx context.Context, arg secretdb.InsertSecretParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

// ListSecretVersions mocks the ListSecretVersions method
func (m *MockSecretRepository) ListSecretVersions(ctx context.Context, environmentID uuid.UUID) ([]secretdb.SecretVersion, error) {
	args := m.Called(ctx, environmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]secretdb.SecretVersion), args.Error(1)
}

// RollbackSecretsToVersion mocks the RollbackSecretsToVersion method
func (m *MockSecretRepository) RollbackSecretsToVersion(ctx context.Context, arg secretdb.RollbackSecretsToVersionParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}
