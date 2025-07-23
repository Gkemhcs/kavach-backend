package secretgroup

import (
	"time"
)

// CreateSecretGroupRequest is the request body for creating a secret group.
// Used by the API to validate and parse secret group creation requests.
type CreateSecretGroupRequest struct {
	Name             string `json:"name" binding:"required"` // Name of the secret group
	Description      string `json:"description"`             // Optional description
	UserID           string `json:"user_id"`                 // User ID of the creator
	OrganizationName string `json:"org_name"`                // Organization name (optional, for display)
	OrganizationID   string `json:"org_id"`                  // Organization ID
}

// UpdateSecretGroupRequest is the request body for updating a secret group.
// Used by the API to validate and parse secret group update requests.
type UpdateSecretGroupRequest struct {
	Name        string `json:"name"`        // New name for the secret group
	Description string `json:"description"` // New description for the secret group
}

// SecretGroupResponseData is the response body for secret group data.
// Used to serialize secret group data for API responses.
type SecretGroupResponseData struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    *string   `json:"description"`
	OrganizationID string    `json:"organization_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
