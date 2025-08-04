package authz

// EnforcerBackend represents the underlying enforcer implementation
// This allows different types of enforcers (Casbin, Mock, etc.) to be returned
type EnforcerBackend interface {
	// Type returns the type of the enforcer backend
	Type() string
	// Value returns the actual enforcer value
	Value() interface{}
}

// Enforcer defines the interface for authorization enforcement operations
type Enforcer interface {
	// GetEnforcer returns the underlying enforcer
	GetEnforcer() EnforcerBackend

	// AddResourceHierarchy adds a resource hierarchy mapping (parent -> child)
	AddResourceHierarchy(parentResource, childResource string) error

	// RemoveResourceHierarchy removes a resource hierarchy mapping
	RemoveResourceHierarchy(parentResource, childResource string) error

	// AddResourceOwner adds owner permission for a resource
	AddResourceOwner(userID, resource string) error

	// RemoveResource removes a resource and all its policies
	RemoveResource(resource string) error

	// AddUserToGroup adds a user to a group
	AddUserToGroup(userID, groupID string) error

	// RemoveUserFromGroup removes a user from a group
	RemoveUserFromGroup(userID, groupID string) error

	// DeleteUserGroup deletes a user group and removes all members
	DeleteUserGroup(groupID string) error

	// GrantRole grants a role to a subject on a specific resource
	GrantRole(subject, role, resource string) error

	// GrantRoleWithPermissions grants a role with specific permissions
	GrantRoleWithPermissions(userID, role, resource string) error

	// RevokeRole revokes a role from a subject on a specific resource
	RevokeRole(subject, role, resource string) error

	// RevokeRoleCascade revokes a role from a subject on a specific resource and all child resources
	RevokeRoleCascade(subject, role, resource string) error

	// CheckPermission checks if a user has permission to perform an action on a resource
	CheckPermission(userID, action, resource string) (bool, error)

	// CheckPermissionEx checks permission with extended information
	CheckPermissionEx(userID, action, resource string) (bool, []string, error)

	// GetUserRoles gets all roles for a user
	GetUserRoles(userID string) ([]string, error)

	// GetUserGroups gets all groups for a user
	GetUserGroups(userID string) ([]string, error)

	// GetUsersForGroup gets all users in a group
	GetUsersForGroup(groupID string) ([]string, error)

	// GetResourcePermissions gets all permissions for a resource
	GetResourcePermissions(resource string) ([][]string, error)

	// SavePolicy saves the current policy to storage
	SavePolicy() error

	// LoadPolicy loads the policy from storage
	LoadPolicy() error

	// DebugPolicies prints all policies for debugging
	DebugPolicies()

	// DebugUserPermissions prints all permissions for a user
	DebugUserPermissions(userID string)

	// DebugResourcePermissions prints all permissions for a resource
	DebugResourcePermissions(resource string)
}
