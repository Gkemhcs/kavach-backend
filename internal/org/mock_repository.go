package org

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockOrganization represents a mock organization in memory
type MockOrganization struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	OwnerID     uuid.UUID `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// MockRepository is a testify/mock implementation of the organization repository
type MockRepository struct {
	mock.Mock
}

// CreateOrganization creates a new organization in the mock repository
func (m *MockRepository) CreateOrganization(ctx context.Context, name, description string, ownerID uuid.UUID) (*MockOrganization, error) {
	args := m.Called(ctx, name, description, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MockOrganization), args.Error(1)
}

// GetOrganizationByID retrieves an organization by ID
func (m *MockRepository) GetOrganizationByID(ctx context.Context, id uuid.UUID) (*MockOrganization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MockOrganization), args.Error(1)
}

// GetOrganizationByName retrieves an organization by name
func (m *MockRepository) GetOrganizationByName(ctx context.Context, name string) (*MockOrganization, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MockOrganization), args.Error(1)
}

// ListOrganizationsByOwner retrieves all organizations owned by a user
func (m *MockRepository) ListOrganizationsByOwner(ctx context.Context, ownerID uuid.UUID) ([]*MockOrganization, error) {
	args := m.Called(ctx, ownerID)
	return args.Get(0).([]*MockOrganization), args.Error(1)
}

// UpdateOrganization updates an existing organization
func (m *MockRepository) UpdateOrganization(ctx context.Context, id uuid.UUID, name, description string) (*MockOrganization, error) {
	args := m.Called(ctx, id, name, description)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MockOrganization), args.Error(1)
}

// DeleteOrganization deletes an organization
func (m *MockRepository) DeleteOrganization(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
} 