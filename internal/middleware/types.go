package middleware

import (
	"github.com/google/uuid"
)

// GrantRoleBindingRequest represents the request payload for granting/revoking roles
type GrantRoleBindingRequest struct {
	UserName       string        `json:"user_name"`
	GroupName      string        `json:"group_name"`
	Role           string        `json:"role"`
	ResourceType   string        `json:"resource_type"`
	ResourceID     uuid.UUID     `json:"resource_id"`
	OrganizationID uuid.UUID     `json:"organization_id"`
	SecretGroupID  uuid.NullUUID `json:"secret_group_id"`
	EnvironmentID  uuid.NullUUID `json:"environment_id"`
}
