package org

// CreateOrganizationRequest is the request body for creating an organization.
type CreateOrganizationRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	UserID      string `json:"user_id"`
} 

// UpdateOrganizationRequest is the request body for updating an organization.
type UpdateOrganizationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	UserID      string `json:"user_id"`

}
