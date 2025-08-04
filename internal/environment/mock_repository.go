package environment

import (
	"context"

	environmentdb "github.com/Gkemhcs/kavach-backend/internal/environment/gen"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockEnvironmentRepository mocks the environment repository interface
type MockEnvironmentRepository struct {
	mock.Mock
}

// AddEnvironmentMember mocks the AddEnvironmentMember method
func (m *MockEnvironmentRepository) AddEnvironmentMember(ctx context.Context, arg environmentdb.AddEnvironmentMemberParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

// CreateEnvironment mocks the CreateEnvironment method
func (m *MockEnvironmentRepository) CreateEnvironment(ctx context.Context, arg environmentdb.CreateEnvironmentParams) (environmentdb.Environment, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(environmentdb.Environment), args.Error(1)
}

// DeleteEnvironment mocks the DeleteEnvironment method
func (m *MockEnvironmentRepository) DeleteEnvironment(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// GetEnvironmentByID mocks the GetEnvironmentByID method
func (m *MockEnvironmentRepository) GetEnvironmentByID(ctx context.Context, id uuid.UUID) (environmentdb.Environment, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(environmentdb.Environment), args.Error(1)
}

// GetEnvironmentByName mocks the GetEnvironmentByName method
func (m *MockEnvironmentRepository) GetEnvironmentByName(ctx context.Context, arg environmentdb.GetEnvironmentByNameParams) (environmentdb.GetEnvironmentByNameRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(environmentdb.GetEnvironmentByNameRow), args.Error(1)
}

// GetEnvironmentMember mocks the GetEnvironmentMember method
func (m *MockEnvironmentRepository) GetEnvironmentMember(ctx context.Context, arg environmentdb.GetEnvironmentMemberParams) (environmentdb.EnvironmentMember, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(environmentdb.EnvironmentMember), args.Error(1)
}

// ListEnvironmentMembers mocks the ListEnvironmentMembers method
func (m *MockEnvironmentRepository) ListEnvironmentMembers(ctx context.Context, environmentID uuid.UUID) ([]environmentdb.EnvironmentMember, error) {
	args := m.Called(ctx, environmentID)
	return args.Get(0).([]environmentdb.EnvironmentMember), args.Error(1)
}

// ListEnvironmentsBySecretGroup mocks the ListEnvironmentsBySecretGroup method
func (m *MockEnvironmentRepository) ListEnvironmentsBySecretGroup(ctx context.Context, secretGroupID uuid.UUID) ([]environmentdb.Environment, error) {
	args := m.Called(ctx, secretGroupID)
	return args.Get(0).([]environmentdb.Environment), args.Error(1)
}

// ListEnvironmentsWithMember mocks the ListEnvironmentsWithMember method
func (m *MockEnvironmentRepository) ListEnvironmentsWithMember(ctx context.Context, arg environmentdb.ListEnvironmentsWithMemberParams) ([]environmentdb.ListEnvironmentsWithMemberRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]environmentdb.ListEnvironmentsWithMemberRow), args.Error(1)
}

// ListMembersOfEnvironment mocks the ListMembersOfEnvironment method
func (m *MockEnvironmentRepository) ListMembersOfEnvironment(ctx context.Context, environmentID uuid.UUID) ([]environmentdb.ListMembersOfEnvironmentRow, error) {
	args := m.Called(ctx, environmentID)
	return args.Get(0).([]environmentdb.ListMembersOfEnvironmentRow), args.Error(1)
}

// RemoveEnvironmentMember mocks the RemoveEnvironmentMember method
func (m *MockEnvironmentRepository) RemoveEnvironmentMember(ctx context.Context, arg environmentdb.RemoveEnvironmentMemberParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

// UpdateEnvironment mocks the UpdateEnvironment method
func (m *MockEnvironmentRepository) UpdateEnvironment(ctx context.Context, arg environmentdb.UpdateEnvironmentParams) (environmentdb.Environment, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(environmentdb.Environment), args.Error(1)
}
