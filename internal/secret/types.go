package secret

import (
	"time"

	"github.com/google/uuid"
)

// Secret represents a single secret with its encrypted value
type Secret struct {
	ID             uuid.UUID `json:"id"`
	VersionID      string    `json:"version_id"`
	Name           string    `json:"name"`
	ValueEncrypted []byte    `json:"-"` // Never expose encrypted values in JSON
}

// SecretVersion represents a version of secrets with commit message and metadata
type SecretVersion struct {
	ID            string    `json:"id"`
	EnvironmentID uuid.UUID `json:"environment_id"`
	CommitMessage string    `json:"commit_message"`
	CreatedAt     time.Time `json:"created_at"`
}

// SecretWithValue represents a secret with its decrypted value (for internal use only)
type SecretWithValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// CreateSecretVersionRequest represents the request to create a new version of secrets
type CreateSecretVersionRequest struct {
	Secrets       []SecretInput `json:"secrets" binding:"required"`
	CommitMessage string        `json:"commit_message" binding:"required"`
}

// SecretInput represents a secret input from the client
type SecretInput struct {
	Name  string `json:"name" binding:"required"`
	Value string `json:"value" binding:"required"`
}

// RollbackRequest represents the request to rollback to a specific version
type RollbackRequest struct {
	VersionID     string `json:"version_id" binding:"required"`
	CommitMessage string `json:"commit_message" binding:"required"`
}

// SecretVersionResponse represents the response for a secret version
type SecretVersionResponse struct {
	ID            string    `json:"id"`
	EnvironmentID uuid.UUID `json:"environment_id"`
	CommitMessage string    `json:"commit_message"`
	CreatedAt     time.Time `json:"created_at"`
	SecretCount   int       `json:"secret_count"`
}

// SecretVersionDetailResponse represents the detailed response for a secret version with secrets
type SecretVersionDetailResponse struct {
	ID            string            `json:"id"`
	EnvironmentID uuid.UUID         `json:"environment_id"`
	CommitMessage string            `json:"commit_message"`
	CreatedAt     time.Time         `json:"created_at"`
	Secrets       []SecretWithValue `json:"secrets"`
}

// SecretDiffResponse represents the diff between two versions
type SecretDiffResponse struct {
	FromVersion string             `json:"from_version"`
	ToVersion   string             `json:"to_version"`
	Changes     []SecretDiffChange `json:"changes"`
}

// SecretDiffChange represents a change in a secret between versions
type SecretDiffChange struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // "added", "removed", "modified"
	OldValue string `json:"old_value,omitempty"`
	NewValue string `json:"new_value,omitempty"`
}

// PushToProviderRequest represents the request to push secrets to an external provider
type PushToProviderRequest struct {
	Provider string            `json:"provider" binding:"required"` // "github", "gcp", etc.
	Config   map[string]string `json:"config" binding:"required"`   // Provider-specific configuration
}
