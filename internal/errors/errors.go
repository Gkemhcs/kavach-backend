package errors

import (
	"errors"
	"net/http"
	"strings"

	"github.com/lib/pq"
)

// APIError represents a structured error for API responses.
// Includes a code, message, and HTTP status for consistent error handling.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

// Error implements the error interface for APIError.
func (e *APIError) Error() string {
	return e.Message
}

// NewAPIError creates a new APIError with the given code, message, and status.
func NewAPIError(code, message string, status int) *APIError {
	return &APIError{Code: code, Message: message, Status: status}
}

// Predefined API errors for common scenarios.
var (
	ErrInvalidBody                        = NewAPIError("invalid_body_format", "unable to parse the request body", http.StatusUnprocessableEntity)
	ErrInvalidToken                       = NewAPIError("invalid_token", "Invalid token", http.StatusUnauthorized)
	ErrExpiredToken                       = NewAPIError("expired_token", "Expired token", http.StatusUnauthorized)
	ErrUserNotFound                       = NewAPIError("user_not_exist", "user not found or created", http.StatusBadRequest)
	ErrOrganizationNotFound               = NewAPIError("organisation_not_exist", "the organisation you are trying to operate not exist", http.StatusBadRequest)
	ErrSecretGroupNotFound                = NewAPIError("secretgroup_not_exist", "the secret group you are trying to operate not exist", http.StatusBadRequest)
	ErrEnvironmentNotFound                = NewAPIError("environment_not_exist", "the environment you are trying to operate not exist", http.StatusBadRequest)
	ErrUserGroupNotFound                  = NewAPIError("user_group_not_exist", "the user group you are trying to operate not exist", http.StatusBadRequest)
	ErrUserMembershipNotFound             = NewAPIError("user_membership_not_exist", "the user groups doesnt contain the user", http.StatusBadRequest)
	ErrRoleBindingNotFound                = NewAPIError("role_binding_not_found", "the role binding not found", http.StatusBadRequest)
	ErrDuplicateOrganization              = NewAPIError("duplicate_organization", "Organization already exists", http.StatusConflict)
	ErrDuplicateSecretGroup               = NewAPIError("duplicate_secret_group", "Secret group already exists", http.StatusConflict)
	ErrDuplicateEnvironment               = NewAPIError("duplicate_environment", "Environment already exists", http.StatusConflict)
	ErrDuplicateUserGroup                 = NewAPIError("duplicate_user_group", "User Group Already Exist in Organization", http.StatusBadRequest)
	ErrDuplicateMemberOfUserGroup         = NewAPIError("duplicate_user_group_membership", "The User is already added to the group", http.StatusBadRequest)
	ErrDuplicateRoleBinding               = NewAPIError("duplicate_role_binding", "the role binding was already present", http.StatusBadRequest)
	ErrNotFound                           = NewAPIError("not_found", "Resource not found", http.StatusNotFound)
	ErrInternalServer                     = NewAPIError("internal_error", "Internal server error", http.StatusInternalServerError)
	ErrEnvironmentNameNotAllowed          = NewAPIError("environment_name_not_allowed", "environment name  you entered is not allowed allowed names are:-prod,dev,staging", http.StatusConflict)
	ErrForeignKeyViolation                = NewAPIError("foreign_key_constraint_violation", "violating foreign key constraint", http.StatusConflict)
	ErrSecretVersionNotFound              = NewAPIError("secret_version_not_found", "the secret version you are trying to operate not exist", http.StatusBadRequest)
	ErrSecretNotFound                     = NewAPIError("secret_not_found", "the secret you are trying to operate not exist", http.StatusBadRequest)
	ErrTargetSecretVersionNotFound        = NewAPIError("target_secret_version_not_found", "the target secret version you are trying to operate not exist", http.StatusBadRequest)
	ErrEnvironmentsMisMatch               = NewAPIError("environment_mismatch", "the target secret version environment is different ", http.StatusBadRequest)
	ErrEncryptionFailed                   = NewAPIError("encryption_failed", "failed to encrypt secret value", http.StatusInternalServerError)
	ErrDecryptionFailed                   = NewAPIError("decryption_failed", "failed to decrypt secret value", http.StatusInternalServerError)
	ErrRollbackFailed                     = NewAPIError("rollback_failed", "failed to rollback to specified version", http.StatusInternalServerError)
	ErrCopySecretsFailed                  = NewAPIError("secret_copy_failed", "failed to copy the secrets from previous version to rollback", http.StatusBadRequest)
	ErrInvalidSecretName                  = NewAPIError("invalid_secret_name", "secret name contains invalid characters or is empty", http.StatusBadRequest)
	ErrSecretValueTooLong                 = NewAPIError("secret_value_too_long", "secret value exceeds maximum allowed length", http.StatusBadRequest)
	ErrEmptySecrets                       = NewAPIError("empty_secrets", "the secrets are empty", http.StatusBadRequest)
	ErrTooManySecrets                     = NewAPIError("too_many_secrets", "number of secrets exceeds maximum allowed limit", http.StatusBadRequest)
	ErrProviderSyncFailed                 = NewAPIError("provider_sync_failed", "failed to sync secrets to external provider", http.StatusInternalServerError)
	ErrProviderCredentialNotFound         = NewAPIError("provider_credential_not_found", "provider credential not found", http.StatusNotFound)
	ErrProviderCredentialExists           = NewAPIError("provider_credential_exists", "provider credential already exists", http.StatusConflict)
	ErrInvalidProviderType                = NewAPIError("invalid_provider_type", "unsupported provider type", http.StatusBadRequest)
	ErrInvalidProviderData                = NewAPIError("invalid_provider_data", "invalid provider credentials or configuration", http.StatusBadRequest)
	ErrProviderCredentialCreateFailed     = NewAPIError("provider_credential_create_failed", "failed to create provider credential", http.StatusInternalServerError)
	ErrProviderCredentialUpdateFailed     = NewAPIError("provider_credential_update_failed", "failed to update provider credential", http.StatusInternalServerError)
	ErrProviderCredentialDeleteFailed     = NewAPIError("provider_credential_delete_failed", "failed to delete provider credential", http.StatusInternalServerError)
	ErrProviderCredentialListFailed       = NewAPIError("provider_credential_list_failed", "failed to list provider credentials", http.StatusInternalServerError)
	ErrProviderCredentialGetFailed        = NewAPIError("provider_credential_get_failed", "failed to retrieve provider credential", http.StatusInternalServerError)
	ErrProviderEncryptionFailed           = NewAPIError("provider_encryption_failed", "failed to encrypt provider credentials", http.StatusInternalServerError)
	ErrProviderDecryptionFailed           = NewAPIError("provider_decryption_failed", "failed to decrypt provider credentials", http.StatusInternalServerError)
	ErrNoSecretsToSync                    = NewAPIError("no_secrets_to_sync", "❌ No secrets found to sync. Please ensure secrets exist in the environment", http.StatusBadRequest)
	ErrGitHubEnvironmentNotFound          = NewAPIError("github_environment_not_found", "❌ GitHub environment specified in config was not found in the repository", http.StatusBadRequest)
	ErrGCPInvalidLocation                 = NewAPIError("gcp_invalid_location", "❌ GCP Secret Manager location specified in config is invalid or not supported", http.StatusBadRequest)
	ErrProviderCredentialValidationFailed = NewAPIError("provider_credential_validation_failed", "❌ Provider credential validation failed. Please check your credentials", http.StatusBadRequest)
	ErrGitHubEncryptionFailed             = NewAPIError("github_encryption_failed", "❌ Failed to encrypt secret for GitHub. Please try again", http.StatusInternalServerError)
)

// IsUniqueViolation checks for unique constraint violation (Postgres).
// Used to detect duplicate resource errors from the database.
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	// Try to cast to pq.Error and check the code
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23505" // unique_violation
	}

	// Fallback to message-based detection (optional)
	return strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "unique constraint")
}

// IsCheckConstraintViolation checks for check constraint violation (Postgres).
// Used to detect invalid data errors from the database.
func IsCheckConstraintViolation(err error) bool {
	pqErr, ok := err.(*pq.Error)
	if !ok {
		return false
	}
	return pqErr.Code == "23514" // check_violation
}

func IsViolatingForeignKeyConstraints(err error) bool {

	// Check lib/pq error
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		if pqErr.Code == "23503" {
			return true
		}
	}
	return false
	// fallback
}
