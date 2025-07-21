package secretgroup

// CreateSecretGroupRequest is the request body for creating a secret group.
type CreateSecretGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	UserID string `json:"user_id"`
	OrganizationID string `json:"org_id"`
}

// UpdateSecretGroupRequest is the request body for updating a secret group.
type UpdateSecretGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
