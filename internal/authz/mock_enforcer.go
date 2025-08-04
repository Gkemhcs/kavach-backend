package authz

import (
	"github.com/stretchr/testify/mock"
)

// MockEnforcerBackend is a mock implementation of EnforcerBackend for testing
type MockEnforcerBackend struct {
	mock.Mock
}

// Type returns the type of the enforcer backend
func (m *MockEnforcerBackend) Type() string {
	args := m.Called()
	return args.String(0)
}

// Value returns the actual enforcer value
func (m *MockEnforcerBackend) Value() interface{} {
	args := m.Called()
	return args.Get(0)
}

// MockEnforcer is a mock implementation of the Enforcer interface for testing purposes
// using testify/mock
type MockEnforcer struct {
	mock.Mock
}

// GetEnforcer returns the underlying enforcer
func (m *MockEnforcer) GetEnforcer() EnforcerBackend {
	args := m.Called()
	return args.Get(0).(EnforcerBackend)
}

// AddResourceHierarchy adds a resource hierarchy mapping (parent -> child)
func (m *MockEnforcer) AddResourceHierarchy(parentResource, childResource string) error {
	args := m.Called(parentResource, childResource)
	return args.Error(0)
}

// RemoveResourceHierarchy removes a resource hierarchy mapping
func (m *MockEnforcer) RemoveResourceHierarchy(parentResource, childResource string) error {
	args := m.Called(parentResource, childResource)
	return args.Error(0)
}

// AddResourceOwner adds owner permission for a resource
func (m *MockEnforcer) AddResourceOwner(userID, resource string) error {
	args := m.Called(userID, resource)
	return args.Error(0)
}

// RemoveResource removes a resource and all its policies
func (m *MockEnforcer) RemoveResource(resource string) error {
	args := m.Called(resource)
	return args.Error(0)
}

// AddUserToGroup adds a user to a group
func (m *MockEnforcer) AddUserToGroup(userID, groupID string) error {
	args := m.Called(userID, groupID)
	return args.Error(0)
}

// RemoveUserFromGroup removes a user from a group
func (m *MockEnforcer) RemoveUserFromGroup(userID, groupID string) error {
	args := m.Called(userID, groupID)
	return args.Error(0)
}

// DeleteUserGroup deletes a user group and removes all members
func (m *MockEnforcer) DeleteUserGroup(groupID string) error {
	args := m.Called(groupID)
	return args.Error(0)
}

// GrantRole grants a role to a subject on a specific resource
func (m *MockEnforcer) GrantRole(subject, role, resource string) error {
	args := m.Called(subject, role, resource)
	return args.Error(0)
}

// GrantRoleWithPermissions grants a role with specific permissions
func (m *MockEnforcer) GrantRoleWithPermissions(userID, role, resource string) error {
	args := m.Called(userID, role, resource)
	return args.Error(0)
}

// RevokeRole revokes a role from a subject on a specific resource
func (m *MockEnforcer) RevokeRole(subject, role, resource string) error {
	args := m.Called(subject, role, resource)
	return args.Error(0)
}

// RevokeRoleCascade revokes a role from a subject on a specific resource and all child resources
func (m *MockEnforcer) RevokeRoleCascade(subject, role, resource string) error {
	args := m.Called(subject, role, resource)
	return args.Error(0)
}

// CheckPermission checks if a user has permission to perform an action on a resource
func (m *MockEnforcer) CheckPermission(userID, action, resource string) (bool, error) {
	args := m.Called(userID, action, resource)
	return args.Bool(0), args.Error(1)
}

// CheckPermissionEx checks permission with extended information
func (m *MockEnforcer) CheckPermissionEx(userID, action, resource string) (bool, []string, error) {
	args := m.Called(userID, action, resource)
	return args.Bool(0), args.Get(1).([]string), args.Error(2)
}

// GetUserRoles gets all roles for a user
func (m *MockEnforcer) GetUserRoles(userID string) ([]string, error) {
	args := m.Called(userID)
	return args.Get(0).([]string), args.Error(1)
}

// GetUserGroups gets all groups for a user
func (m *MockEnforcer) GetUserGroups(userID string) ([]string, error) {
	args := m.Called(userID)
	return args.Get(0).([]string), args.Error(1)
}

// GetUsersForGroup gets all users in a group
func (m *MockEnforcer) GetUsersForGroup(groupID string) ([]string, error) {
	args := m.Called(groupID)
	return args.Get(0).([]string), args.Error(1)
}

// GetResourcePermissions gets all permissions for a resource
func (m *MockEnforcer) GetResourcePermissions(resource string) ([][]string, error) {
	args := m.Called(resource)
	return args.Get(0).([][]string), args.Error(1)
}

// SavePolicy saves the current policy to storage
func (m *MockEnforcer) SavePolicy() error {
	args := m.Called()
	return args.Error(0)
}

// LoadPolicy loads the policy from storage
func (m *MockEnforcer) LoadPolicy() error {
	args := m.Called()
	return args.Error(0)
}

// DebugPolicies prints all policies for debugging
func (m *MockEnforcer) DebugPolicies() {
	m.Called()
}

// DebugUserPermissions prints all permissions for a user
func (m *MockEnforcer) DebugUserPermissions(userID string) {
	m.Called(userID)
}

// DebugResourcePermissions prints all permissions for a resource
func (m *MockEnforcer) DebugResourcePermissions(resource string) {
	m.Called(resource)
}
