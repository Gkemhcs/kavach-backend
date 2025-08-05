package environment

import (
	"time"
)

// CreateEnvironmentRequest is the request body for creating an environment.
// Used by the API to validate and parse environment creation requests.
type CreateEnvironmentRequest struct {
	Name         string `json:"name" binding:"required"` // Name of the environment
	Description  string `json:"description"`             // Optional description
	SecretGroup  string `json:"secret_group"`            // Secret group ID
	Organization string `json:"organization"`            // Organization ID
	UserId       string `json:"user_id"`                 // User ID of the creator
}

type ListAccessibleEnvironmentsRow struct {
	ID              string `json:"environment_id"`
	Name            string `json:"name"`
	SecretGroupName string `json:"secret_group_name"`
	Role            string `json:"role"`
	InheritedFrom   string `json:"inherited_from"`
}

// EnvironmentResponseData is the response body for environment data.
// Used to serialize environment data for API responses.
type EnvironmentResponseData struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	SecretGroupID  string    `json:"secret_group_id"`
	OrganizationID string    `json:"organization_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Description    *string   `json:"description"`
}

// UpdateEnvironmentRequest is the request body for updating an environment.
// Used by the API to validate and parse environment update requests.
type UpdateEnvironmentRequest struct {
	Name string `json:"name"` // New name for the environment
}
