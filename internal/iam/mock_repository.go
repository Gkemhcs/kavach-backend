package iam

import (
	"context"
	"database/sql"

	iam_db "github.com/Gkemhcs/kavach-backend/internal/iam/gen"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockIamRepository is a mock implementation of the IAM repository interface
// for testing purposes using testify/mock
type MockIamRepository struct {
	mock.Mock
}

// CreateRoleBinding mocks the CreateRoleBinding method
func (m *MockIamRepository) CreateRoleBinding(ctx context.Context, arg iam_db.CreateRoleBindingParams) (iam_db.RoleBinding, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(iam_db.RoleBinding), args.Error(1)
}

// DeleteRoleBinding mocks the DeleteRoleBinding method
func (m *MockIamRepository) DeleteRoleBinding(ctx context.Context, arg iam_db.DeleteRoleBindingParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

// GetRoleBinding mocks the GetRoleBinding method
func (m *MockIamRepository) GetRoleBinding(ctx context.Context, arg iam_db.GetRoleBindingParams) (iam_db.RoleBinding, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(iam_db.RoleBinding), args.Error(1)
}

// GrantRoleBinding mocks the GrantRoleBinding method
func (m *MockIamRepository) GrantRoleBinding(ctx context.Context, arg iam_db.GrantRoleBindingParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

// ListAccessibleEnvironments mocks the ListAccessibleEnvironments method
func (m *MockIamRepository) ListAccessibleEnvironments(ctx context.Context, arg iam_db.ListAccessibleEnvironmentsParams) ([]iam_db.ListAccessibleEnvironmentsRow, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]iam_db.ListAccessibleEnvironmentsRow), args.Error(1)
}

// ListAccessibleOrganizations mocks the ListAccessibleOrganizations method
func (m *MockIamRepository) ListAccessibleOrganizations(ctx context.Context, userID uuid.NullUUID) ([]iam_db.ListAccessibleOrganizationsRow, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]iam_db.ListAccessibleOrganizationsRow), args.Error(1)
}

// ListAccessibleSecretGroups mocks the ListAccessibleSecretGroups method
func (m *MockIamRepository) ListAccessibleSecretGroups(ctx context.Context, arg iam_db.ListAccessibleSecretGroupsParams) ([]iam_db.ListAccessibleSecretGroupsRow, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]iam_db.ListAccessibleSecretGroupsRow), args.Error(1)
}

// RevokeRoleBinding mocks the RevokeRoleBinding method
func (m *MockIamRepository) RevokeRoleBinding(ctx context.Context, arg iam_db.RevokeRoleBindingParams) (sql.Result, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(sql.Result), args.Error(1)
}

// UpdateUserRole mocks the UpdateUserRole method
func (m *MockIamRepository) UpdateUserRole(ctx context.Context, arg iam_db.UpdateUserRoleParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

// CheckUserPermission mocks the CheckUserPermission method
func (m *MockIamRepository) CheckUserPermission(ctx context.Context, arg iam_db.CheckUserPermissionParams) (bool, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(bool), args.Error(1)
}

// GetUserEffectiveRole mocks the GetUserEffectiveRole method
func (m *MockIamRepository) GetUserEffectiveRole(ctx context.Context, arg iam_db.GetUserEffectiveRoleParams) (iam_db.UserRole, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(iam_db.UserRole), args.Error(1)
}

// ListAccessibleEnvironmentsEnhanced mocks the ListAccessibleEnvironmentsEnhanced method
func (m *MockIamRepository) ListAccessibleEnvironmentsEnhanced(ctx context.Context, arg iam_db.ListAccessibleEnvironmentsEnhancedParams) ([]iam_db.ListAccessibleEnvironmentsEnhancedRow, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]iam_db.ListAccessibleEnvironmentsEnhancedRow), args.Error(1)
}

// ListAccessibleOrganizationsEnhanced mocks the ListAccessibleOrganizationsEnhanced method
func (m *MockIamRepository) ListAccessibleOrganizationsEnhanced(ctx context.Context, userID uuid.NullUUID) ([]iam_db.ListAccessibleOrganizationsEnhancedRow, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]iam_db.ListAccessibleOrganizationsEnhancedRow), args.Error(1)
}

// ListAccessibleSecretGroupsEnhanced mocks the ListAccessibleSecretGroupsEnhanced method
func (m *MockIamRepository) ListAccessibleSecretGroupsEnhanced(ctx context.Context, arg iam_db.ListAccessibleSecretGroupsEnhancedParams) ([]iam_db.ListAccessibleSecretGroupsEnhancedRow, error) {
	args := m.Called(ctx, arg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]iam_db.ListAccessibleSecretGroupsEnhancedRow), args.Error(1)
}

// MockSqlResult is a mock implementation of sql.Result for testing
type MockSqlResult struct {
	mock.Mock
}

// LastInsertId mocks the LastInsertId method
func (m *MockSqlResult) LastInsertId() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// RowsAffected mocks the RowsAffected method
func (m *MockSqlResult) RowsAffected() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}
