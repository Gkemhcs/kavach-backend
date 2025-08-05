package org

import (
	"context"

	orgdb "github.com/Gkemhcs/kavach-backend/internal/org/gen"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockOrgRepository is a mock implementation of orgdb.Querier
type MockOrgRepository struct {
	mock.Mock
}

// AddOrgMember mocks the AddOrgMember method
func (m *MockOrgRepository) AddOrgMember(ctx context.Context, arg orgdb.AddOrgMemberParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

// CreateOrganization mocks the CreateOrganization method
func (m *MockOrgRepository) CreateOrganization(ctx context.Context, arg orgdb.CreateOrganizationParams) (orgdb.Organization, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(orgdb.Organization), args.Error(1)
}

// DeleteOrganization mocks the DeleteOrganization method
func (m *MockOrgRepository) DeleteOrganization(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// GetOrgMember mocks the GetOrgMember method
func (m *MockOrgRepository) GetOrgMember(ctx context.Context, arg orgdb.GetOrgMemberParams) (orgdb.OrgMember, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(orgdb.OrgMember), args.Error(1)
}

// GetOrganizationByID mocks the GetOrganizationByID method
func (m *MockOrgRepository) GetOrganizationByID(ctx context.Context, id uuid.UUID) (orgdb.Organization, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(orgdb.Organization), args.Error(1)
}

// GetOrganizationByName mocks the GetOrganizationByName method
func (m *MockOrgRepository) GetOrganizationByName(ctx context.Context, name string) (orgdb.Organization, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(orgdb.Organization), args.Error(1)
}

// ListMembersOfOrganization mocks the ListMembersOfOrganization method
func (m *MockOrgRepository) ListMembersOfOrganization(ctx context.Context, orgID uuid.UUID) ([]orgdb.ListMembersOfOrganizationRow, error) {
	args := m.Called(ctx, orgID)
	return args.Get(0).([]orgdb.ListMembersOfOrganizationRow), args.Error(1)
}

// ListOrgMembers mocks the ListOrgMembers method
func (m *MockOrgRepository) ListOrgMembers(ctx context.Context, orgID uuid.UUID) ([]orgdb.OrgMember, error) {
	args := m.Called(ctx, orgID)
	return args.Get(0).([]orgdb.OrgMember), args.Error(1)
}



// ListOrganizationsByOwner mocks the ListOrganizationsByOwner method
func (m *MockOrgRepository) ListOrganizationsByOwner(ctx context.Context, ownerID uuid.UUID) ([]orgdb.Organization, error) {
	args := m.Called(ctx, ownerID)
	return args.Get(0).([]orgdb.Organization), args.Error(1)
}

// ListOrganizationsWithMember mocks the ListOrganizationsWithMember method
func (m *MockOrgRepository) ListOrganizationsWithMember(ctx context.Context, userID uuid.UUID) ([]orgdb.ListOrganizationsWithMemberRow, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]orgdb.ListOrganizationsWithMemberRow), args.Error(1)
}

// RemoveOrgMember mocks the RemoveOrgMember method
func (m *MockOrgRepository) RemoveOrgMember(ctx context.Context, arg orgdb.RemoveOrgMemberParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

// UpdateOrganization mocks the UpdateOrganization method
func (m *MockOrgRepository) UpdateOrganization(ctx context.Context, arg orgdb.UpdateOrganizationParams) (orgdb.Organization, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(orgdb.Organization), args.Error(1)
}
