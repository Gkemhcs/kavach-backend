package utils

import (
	"github.com/gin-gonic/gin"
)

// APIResponse is a generic API response wrapper for success and error responses.
// Used to standardize API responses across endpoints.
type APIResponse[T any] struct {
	Success   bool   `json:"success"`
	Data      T      `json:"data,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`
	ErrorMsg  string `json:"error_msg,omitempty"`
}

// RespondSuccess sends a standardized success response with the given data and status code.
func RespondSuccess[T any](c *gin.Context, statusCode int, data T) {
	c.JSON(statusCode, APIResponse[T]{
		Success: true,
		Data:    data,
	})
}

// RespondError sends a standardized error response with the given error code and message.
func RespondError(c *gin.Context, status int, errorCode, errorMsg string) {
	c.AbortWithStatusJSON(status, APIResponse[any]{
		Success:   false,
		ErrorCode: errorCode,
		ErrorMsg:  errorMsg,
	})
}
