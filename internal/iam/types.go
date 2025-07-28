package iam

import "github.com/google/uuid"

// CreateRoleBindingRequest represents the request payload for creating a role binding.
// Used internally for creating role bindings with explicit user IDs and resource references.
type CreateRoleBindingRequest struct {
	UserID         uuid.UUID     // ID of the user to grant the role to
	Role           string        // Role to grant (owner, admin, editor, viewer)
	ResourceType   string        // Type of resource (organization, secret_group, environment)
	ResourceID     uuid.UUID     // ID of the resource to grant access to
	OrganizationID uuid.UUID     // Organization context for the role binding
	SecretGroupID  uuid.NullUUID // Optional secret group context (for environment-level permissions)
	EnvironmentID  uuid.NullUUID // Optional environment context (for environment-level permissions)
}

type DeleteRoleBindingRequest struct {
	ResourceType string    // Type of resource (organization, secret_group, environment)
	ResourceID   uuid.UUID // ID of the resource to grant access to

}

// GrantRoleBindingRequest represents the request payload for granting a role to a user or user group.
// Supports both user-based and group-based role assignments on various resource types.
type GrantRoleBindingRequest struct {
	UserName       string        `json:"user_name"`       // GitHub username of the user (mutually exclusive with group_name)
	GroupName      string        `json:"group_name"`      // Name of the user group (mutually exclusive with user_name)
	Role           string        `json:"role"`            // Role to grant (owner, admin, editor, viewer)
	ResourceType   string        `json:"resource_type"`   // Type of resource (organization, secret_group, environment)
	ResourceID     uuid.UUID     `json:"resource_id"`     // ID of the resource to grant access to
	OrganizationID uuid.UUID     `json:"organization_id"` // Organization context for the role binding
	SecretGroupID  uuid.NullUUID `json:"secret_group_id"` // Optional secret group context (for environment-level permissions)
	EnvironmentID  uuid.NullUUID `json:"environment_id"`  // Optional environment context (for environment-level permissions)
}

// RevokeRoleBindingRequest represents the request payload for revoking a role from a user or user group.
// Supports both user-based and group-based role revocation on various resource types.
type RevokeRoleBindingRequest struct {
	UserName       string        `json:"user_name"`       // GitHub username of the user (mutually exclusive with group_name)
	GroupName      string        `json:"group_name"`      // Name of the user group (mutually exclusive with user_name)
	Role           string        `json:"role"`            // Role to revoke (owner, admin, editor, viewer)
	ResourceType   string        `json:"resource_type"`   // Type of resource (organization, secret_group, environment)
	ResourceID     uuid.UUID     `json:"resource_id"`     // ID of the resource to revoke access from
	OrganizationID uuid.UUID     `json:"organization_id"` // Organization context for the role binding
	SecretGroupID  uuid.NullUUID `json:"secret_group_id"` // Optional secret group context (for environment-level permissions)
	EnvironmentID  uuid.NullUUID `json:"environment_id"`  // Optional environment context (for environment-level permissions)

}
