package environment

// CreateEnvironmentRequest is the request body for creating an environment.
type CreateEnvironmentRequest struct {
	Name         string `json:"name" binding:"required"`
	SecretGroup  string `json:"secret_group"`
	Organization string `json:"organization"`
	UserId       string `json:"user_id"`
}

// UpdateEnvironmentRequest is the request body for updating an environment.
type UpdateEnvironmentRequest struct {
	Name string `json:"name"`
}
