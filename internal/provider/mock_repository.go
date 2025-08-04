package provider

import (
	"context"

	providerdb "github.com/Gkemhcs/kavach-backend/internal/provider/gen"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockProviderRepository is a mock implementation of the providerdb.Querier interface
type MockProviderRepository struct {
	mock.Mock
}

// CreateProviderCredential mocks the CreateProviderCredential method
func (m *MockProviderRepository) CreateProviderCredential(ctx context.Context, arg providerdb.CreateProviderCredentialParams) (providerdb.ProviderCredential, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return providerdb.ProviderCredential{}, args.Error(1)
	}
	return args.Get(0).(providerdb.ProviderCredential), args.Error(1)
}

// GetProviderCredential mocks the GetProviderCredential method
func (m *MockProviderRepository) GetProviderCredential(ctx context.Context, arg providerdb.GetProviderCredentialParams) (providerdb.ProviderCredential, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return providerdb.ProviderCredential{}, args.Error(1)
	}
	return args.Get(0).(providerdb.ProviderCredential), args.Error(1)
}

// GetProviderCredentialByID mocks the GetProviderCredentialByID method
func (m *MockProviderRepository) GetProviderCredentialByID(ctx context.Context, id uuid.UUID) (providerdb.ProviderCredential, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return providerdb.ProviderCredential{}, args.Error(1)
	}
	return args.Get(0).(providerdb.ProviderCredential), args.Error(1)
}

// ListProviderCredentials mocks the ListProviderCredentials method
func (m *MockProviderRepository) ListProviderCredentials(ctx context.Context, environmentID uuid.UUID) ([]providerdb.ProviderCredential, error) {
	args := m.Called(ctx, environmentID)
	if args.Get(0) == nil {
		return []providerdb.ProviderCredential{}, args.Error(1)
	}
	return args.Get(0).([]providerdb.ProviderCredential), args.Error(1)
}

// UpdateProviderCredential mocks the UpdateProviderCredential method
func (m *MockProviderRepository) UpdateProviderCredential(ctx context.Context, arg providerdb.UpdateProviderCredentialParams) (providerdb.ProviderCredential, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return providerdb.ProviderCredential{}, args.Error(1)
	}
	return args.Get(0).(providerdb.ProviderCredential), args.Error(1)
}

// DeleteProviderCredential mocks the DeleteProviderCredential method
func (m *MockProviderRepository) DeleteProviderCredential(ctx context.Context, arg providerdb.DeleteProviderCredentialParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}
