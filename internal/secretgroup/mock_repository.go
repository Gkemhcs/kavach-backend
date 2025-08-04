package secretgroup

import (
	"context"

	secretgroupdb "github.com/Gkemhcs/kavach-backend/internal/secretgroup/gen"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockSecretGroupRepository mocks the secretgroupdb.Querier interface
type MockSecretGroupRepository struct {
	mock.Mock
}

// AddSecretGroupMember mocks adding a member to a secret group
func (m *MockSecretGroupRepository) AddSecretGroupMember(ctx context.Context, arg secretgroupdb.AddSecretGroupMemberParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

// CreateSecretGroup mocks creating a secret group
func (m *MockSecretGroupRepository) CreateSecretGroup(ctx context.Context, arg secretgroupdb.CreateSecretGroupParams) (secretgroupdb.SecretGroup, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(secretgroupdb.SecretGroup), args.Error(1)
}

// DeleteSecretGroup mocks deleting a secret group
func (m *MockSecretGroupRepository) DeleteSecretGroup(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// GetSecretGroupByID mocks getting a secret group by ID
func (m *MockSecretGroupRepository) GetSecretGroupByID(ctx context.Context, id uuid.UUID) (secretgroupdb.SecretGroup, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(secretgroupdb.SecretGroup), args.Error(1)
}

// GetSecretGroupByName mocks getting a secret group by name
func (m *MockSecretGroupRepository) GetSecretGroupByName(ctx context.Context, arg secretgroupdb.GetSecretGroupByNameParams) (secretgroupdb.SecretGroup, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(secretgroupdb.SecretGroup), args.Error(1)
}

// GetSecretGroupMember mocks getting a secret group member
func (m *MockSecretGroupRepository) GetSecretGroupMember(ctx context.Context, arg secretgroupdb.GetSecretGroupMemberParams) (secretgroupdb.SecretGroupMember, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(secretgroupdb.SecretGroupMember), args.Error(1)
}

// ListMembersOfSecretGroup mocks listing members of a secret group
func (m *MockSecretGroupRepository) ListMembersOfSecretGroup(ctx context.Context, secretGroupID uuid.UUID) ([]secretgroupdb.ListMembersOfSecretGroupRow, error) {
	args := m.Called(ctx, secretGroupID)
	return args.Get(0).([]secretgroupdb.ListMembersOfSecretGroupRow), args.Error(1)
}

// ListSecretGroupMembers mocks listing secret group members
func (m *MockSecretGroupRepository) ListSecretGroupMembers(ctx context.Context, secretGroupID uuid.UUID) ([]secretgroupdb.SecretGroupMember, error) {
	args := m.Called(ctx, secretGroupID)
	return args.Get(0).([]secretgroupdb.SecretGroupMember), args.Error(1)
}

// ListSecretGroupsByOrg mocks listing secret groups by organization
func (m *MockSecretGroupRepository) ListSecretGroupsByOrg(ctx context.Context, organizationID uuid.UUID) ([]secretgroupdb.SecretGroup, error) {
	args := m.Called(ctx, organizationID)
	return args.Get(0).([]secretgroupdb.SecretGroup), args.Error(1)
}

// ListSecretGroupsWithMember mocks listing secret groups with a member
func (m *MockSecretGroupRepository) ListSecretGroupsWithMember(ctx context.Context, arg secretgroupdb.ListSecretGroupsWithMemberParams) ([]secretgroupdb.ListSecretGroupsWithMemberRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]secretgroupdb.ListSecretGroupsWithMemberRow), args.Error(1)
}

// RemoveSecretGroupMember mocks removing a member from a secret group
func (m *MockSecretGroupRepository) RemoveSecretGroupMember(ctx context.Context, arg secretgroupdb.RemoveSecretGroupMemberParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

// UpdateSecretGroup mocks updating a secret group
func (m *MockSecretGroupRepository) UpdateSecretGroup(ctx context.Context, arg secretgroupdb.UpdateSecretGroupParams) (secretgroupdb.SecretGroup, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(secretgroupdb.SecretGroup), args.Error(1)
}
