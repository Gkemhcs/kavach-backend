package errors

import (
	"errors"
	"net/http"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (e *APIError) Error() string {
	return e.Message
}

func NewAPIError(code, message string, status int) *APIError {
	return &APIError{Code: code, Message: message, Status: status}
}

var (
	ErrInvalidToken          = NewAPIError("invalid_token", "Invalid token", http.StatusUnauthorized)
	ErrExpiredToken          = NewAPIError("expired_token", "Expired token", http.StatusUnauthorized)
	ErrDuplicateOrganization = NewAPIError("duplicate_organization", "Organization already exists", http.StatusConflict)
	ErrDuplicateSecretGroup  = NewAPIError("duplicate_secret_group", "Secret group already exists", http.StatusConflict)
	ErrDuplicateEnvironment  = NewAPIError("duplicate_environment", "Environment already exists", http.StatusConflict)
	ErrNotFound              = NewAPIError("not_found", "Resource not found", http.StatusNotFound)
	ErrInternalServer        = NewAPIError("internal_error", "Internal server error", http.StatusInternalServerError)
)

// Helper to check for unique constraint violation (Postgres)
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrDuplicateOrganization) ||
		errors.Is(err, ErrDuplicateSecretGroup) ||
		errors.Is(err, ErrDuplicateEnvironment) ||
		(err.Error() != "" && (contains(err.Error(), "unique constraint") || contains(err.Error(), "duplicate key")))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr))))
}
