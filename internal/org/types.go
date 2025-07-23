package org

import (
	"github.com/google/uuid"
)

// CreateOrganizationRequest is the request body for creating an organization.
// Used by the API to validate and parse organization creation requests.
type CreateOrganizationRequest struct {
	Name        string `json:"name" binding:"required"` // Name of the organization
	Description string `json:"description"`             // Optional description
	UserID      string `json:"user_id"`                 // User ID of the creator
}

// OrganizationResponseData is the response body for organization data.
// Used to serialize organization data for API responses.
type OrganizationResponseData struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	CreatedAt   string  `json:"created_at"`
}

// ListOrganizationsWithMemberRow represents a user's membership in an organization.
// Used to serialize organization membership data for API responses.
type ListOrganizationsWithMemberRow struct {
	OrgID uuid.UUID `json:"org_id"`
	Name  string    `json:"name"`
	Role  string    `json:"role"`
}

// UpdateOrganizationRequest is the request body for updating an organization.
// Used by the API to validate and parse organization update requests.
type UpdateOrganizationRequest struct {
	Name        string `json:"name"`        // New name for the organization
	Description string `json:"description"` // New description for the organization
	UserID      string `json:"user_id"`     // User ID of the updater
}
