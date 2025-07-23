package errors

import (
	"errors"
	"net/http"

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
	ErrInvalidToken             = NewAPIError("invalid_token", "Invalid token", http.StatusUnauthorized)
	ErrExpiredToken             = NewAPIError("expired_token", "Expired token", http.StatusUnauthorized)
	ErrOrganizationNotFound     = NewAPIError("organisation_not_exist", "the organisation you are trying to delete not exist", http.StatusBadRequest)
	ErrSecretGroupNotFound      = NewAPIError("secretgroup_not_exist", "the secret group you are trying to delete not exist", http.StatusBadRequest)
	ErrEnvironmentNotFound      = NewAPIError("environment_not_exist", "the environment you are trying to delete not exist", http.StatusBadRequest)
	ErrDuplicateOrganization    = NewAPIError("duplicate_organization", "Organization already exists", http.StatusConflict)
	ErrDuplicateSecretGroup     = NewAPIError("duplicate_secret_group", "Secret group already exists", http.StatusConflict)
	ErrDuplicateEnvironment     = NewAPIError("duplicate_environment", "Environment already exists", http.StatusConflict)
	ErrNotFound                 = NewAPIError("not_found", "Resource not found", http.StatusNotFound)
	ErrInternalServer           = NewAPIError("internal_error", "Internal server error", http.StatusInternalServerError)
	ErrEnvironmenNameNotAllowed = NewAPIError("environment_name_not_allowed", "environment name  you entered is not allowed allowed names are:-prod,dev,staging", http.StatusConflict)
)

// IsUniqueViolation checks for unique constraint violation (Postgres).
// Used to detect duplicate resource errors from the database.
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrDuplicateOrganization) ||
		errors.Is(err, ErrDuplicateSecretGroup) ||
		errors.Is(err, ErrDuplicateEnvironment) ||
		(err.Error() != "" && (contains(err.Error(), "unique constraint") || contains(err.Error(), "duplicate key")))
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

// contains checks if substr is in s. Used for error string matching.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr))))
}
