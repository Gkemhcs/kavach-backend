package errors

import (
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
	ErrInvalidBody                = NewAPIError("invalid_body_format", "unable to parse the request body", http.StatusUnprocessableEntity)
	ErrInvalidToken               = NewAPIError("invalid_token", "Invalid token", http.StatusUnauthorized)
	ErrExpiredToken               = NewAPIError("expired_token", "Expired token", http.StatusUnauthorized)
	ErrUserNotFound               = NewAPIError("user_not_exist", "user not found or created", http.StatusBadRequest)
	ErrOrganizationNotFound       = NewAPIError("organisation_not_exist", "the organisation you are trying to operate not exist", http.StatusBadRequest)
	ErrSecretGroupNotFound        = NewAPIError("secretgroup_not_exist", "the secret group you are trying to operate not exist", http.StatusBadRequest)
	ErrEnvironmentNotFound        = NewAPIError("environment_not_exist", "the environment you are trying to operate not exist", http.StatusBadRequest)
	ErrUserGroupNotFound          = NewAPIError("user_group_not_exist", "the user group you are trying to operate not exist", http.StatusBadRequest)
	ErrUserMembershipNotFound     = NewAPIError("user_membership_not_exist", "the user groups doesnt contain the user", http.StatusBadRequest)
	ErrRoleBindingNotFound        = NewAPIError("role_binding_not_found", "the role binding not found", http.StatusBadRequest)
	ErrDuplicateOrganization      = NewAPIError("duplicate_organization", "Organization already exists", http.StatusConflict)
	ErrDuplicateSecretGroup       = NewAPIError("duplicate_secret_group", "Secret group already exists", http.StatusConflict)
	ErrDuplicateEnvironment       = NewAPIError("duplicate_environment", "Environment already exists", http.StatusConflict)
	ErrDuplicateUserGroup         = NewAPIError("duplicate_user_group", "User Group Already Exist in Organization", http.StatusBadRequest)
	ErrDuplicateMemberOfUserGroup = NewAPIError("duplicate_user_group_membership", "The User is already added to the group", http.StatusBadRequest)
	ErrDuplicateRoleBinding       = NewAPIError("duplicate_role_binding", "the role binding was already present", http.StatusBadRequest)
	ErrNotFound                   = NewAPIError("not_found", "Resource not found", http.StatusNotFound)
	ErrInternalServer             = NewAPIError("internal_error", "Internal server error", http.StatusInternalServerError)
	ErrEnvironmenNameNotAllowed   = NewAPIError("environment_name_not_allowed", "environment name  you entered is not allowed allowed names are:-prod,dev,staging", http.StatusConflict)
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
