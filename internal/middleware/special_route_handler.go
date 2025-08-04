package middleware

import (
	"fmt"
	"strings"

	"github.com/Gkemhcs/kavach-backend/internal/authz"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SpecialRouteHandler handles authorization for special routes
type SpecialRouteHandler struct {
	enforcer          authz.Enforcer
	logger            *logrus.Logger
	permissionHandler *PermissionHandler
}

// NewSpecialRouteHandler creates a new special route handler
func NewSpecialRouteHandler(enforcer authz.Enforcer, logger *logrus.Logger) *SpecialRouteHandler {
	return &SpecialRouteHandler{
		enforcer:          enforcer,
		logger:            logger,
		permissionHandler: NewPermissionHandler(enforcer, logger),
	}
}

// HandleSpecialRoute handles authorization for special routes
func (srh *SpecialRouteHandler) HandleSpecialRoute(c *gin.Context, userID string) error {
	path := srh.trimAPIPrefix(c.Request.URL.Path)

	// Handle permissions grant/revoke routes
	if strings.Contains(path, "/permissions/grant") {
		return srh.permissionHandler.HandleGrantPermission(c, userID)
	}

	if strings.Contains(path, "/permissions/revoke") {
		return srh.permissionHandler.HandleRevokePermission(c, userID)
	}

	// Handle user group members route
	if strings.Contains(path, "/members") {
		return srh.handleUserGroupMembers(c, userID)
	}

	// Handle secret routes
	if strings.Contains(path, "/secrets") {
		return srh.handleSecretRoutes(c, userID)
	}

	// Handle provider routes
	if strings.Contains(path, "/providers") {
		return srh.handleProviderRoutes(c, userID)
	}

	return fmt.Errorf("unknown special route: %s", path)
}

// handleProviderRoutes handles authorization for provider routes
func (srh *SpecialRouteHandler) handleProviderRoutes(c *gin.Context, userID string) error {
	logEntry := srh.logger.WithFields(logrus.Fields{
		"operation": "provider_routes",
		"user_id":   userID,
		"method":    c.Request.Method,
	})

	path := srh.trimAPIPrefix(c.Request.URL.Path)
	logEntry = logEntry.WithField("path", path)

	// Extract organization ID, secret group ID, and environment ID from path
	// Path format: /organizations/{orgID}/secret-groups/{secretGroupID}/environments/{envID}/providers/*
	parts := strings.Split(path, "/")
	if len(parts) < 7 {
		logEntry.WithField("error", "invalid_path_format").Error("Invalid provider route path")
		return fmt.Errorf("invalid provider route path: %s", path)
	}

	orgID := parts[2]
	secretGroupID := parts[4]
	envID := parts[6]
	parentResource := fmt.Sprintf("/organizations/%s/secret-groups/%s/environments/%s", orgID, secretGroupID, envID)

	// Determine action based on HTTP method
	var action string
	switch c.Request.Method {
	case "GET":
		action = "view_provider_config" // For viewing provider configurations
	default:
		action = "manage_provider_config" // For creating, updating, deleting provider configurations
	}

	hasPermission, explanations, err := srh.enforcer.CheckPermissionEx(userID, action, parentResource)
	if err != nil {
		logEntry.WithFields(logrus.Fields{
			"error":      "permission_check_failed",
			"permission": action,
			"resource":   parentResource,
		}).Error("Failed to check permission")
		return fmt.Errorf("failed to check permission: %v", err)
	}

	if !hasPermission {
		logEntry.WithFields(logrus.Fields{
			"permission": action,
			"resource":   parentResource,
			"result":     "denied",
			"reason":     explanations,
		}).Warn("User does not have required permission on parent environment")
		return fmt.Errorf("user %s does not have %s permission on parent environment %s", userID, action, parentResource)
	}

	return nil
}

// handleUserGroupMembers handles authorization for user group members operations
func (srh *SpecialRouteHandler) handleUserGroupMembers(c *gin.Context, userID string) error {
	logEntry := srh.logger.WithFields(logrus.Fields{
		"operation": "user_group_members",
		"user_id":   userID,
		"method":    c.Request.Method,
	})

	path := srh.trimAPIPrefix(c.Request.URL.Path)

	// Extract organization ID and user group ID from path
	// Path format: /organizations/{orgID}/user-groups/{userGroupID}/members
	parts := strings.Split(path, "/")
	if len(parts) < 6 {
		logEntry.WithField("error", "invalid_path_format").Error("Invalid user group members path")
		return fmt.Errorf("invalid user group members path: %s", path)
	}

	orgID := parts[2]
	userGroupID := parts[4]
	resource := fmt.Sprintf("/organizations/%s/user-groups/%s", orgID, userGroupID)

	// Determine action based on HTTP method
	var action string
	switch c.Request.Method {
	case "GET":
		action = "create" // User needs create permission to view members
	case "POST":
		action = "create"
	case "DELETE":
		action = "delete"
	default:
		logEntry.WithField("error", "unsupported_method").Error("Unsupported method for user group members")
		return fmt.Errorf("unsupported method for user group members: %s", c.Request.Method)
	}

	hasPermission, explanations, err := srh.enforcer.CheckPermissionEx(userID, action, resource)
	if err != nil {
		logEntry.WithFields(logrus.Fields{
			"error":      "permission_check_failed",
			"permission": action,
			"resource":   resource,
		}).Error("Failed to check permission")
		return fmt.Errorf("failed to check permission: %v", err)
	}

	if !hasPermission {
		logEntry.WithFields(logrus.Fields{
			"permission": action,
			"resource":   resource,
			"result":     "denied",
			"reason":     explanations,
		}).Warn("User does not have required permission")
		return fmt.Errorf("user %s does not have %s permission on %s", userID, action, resource)
	}

	return nil
}

// handleSecretRoutes handles authorization for secret routes
func (srh *SpecialRouteHandler) handleSecretRoutes(c *gin.Context, userID string) error {
	logEntry := srh.logger.WithFields(logrus.Fields{
		"operation": "secret_routes",
		"user_id":   userID,
		"method":    c.Request.Method,
	})

	path := srh.trimAPIPrefix(c.Request.URL.Path)
	logEntry = logEntry.WithField("path", path)

	// Extract organization ID, secret group ID, and environment ID from path
	// Path format: /organizations/{orgID}/secret-groups/{secretGroupID}/environments/{envID}/secrets/*
	parts := strings.Split(path, "/")
	if len(parts) < 7 {
		logEntry.WithField("error", "invalid_path_format").Error("Invalid secret route path")
		return fmt.Errorf("invalid secret route path: %s", path)
	}

	orgID := parts[2]
	secretGroupID := parts[4]
	envID := parts[6]
	parentResource := fmt.Sprintf("/organizations/%s/secret-groups/%s/environments/%s", orgID, secretGroupID, envID)

	// Determine action based on HTTP method and specific route
	var action string
	switch c.Request.Method {
	case "GET":
		action = "read" // For viewing secret versions and contents
	default:
		// Check if this is a sync operation
		if strings.Contains(path, "/sync") {
			action = "sync" // For syncing secrets
		} else {
			action = "create" // For creating new secret versions, rollback, etc.
		}
	}

	hasPermission, explanations, err := srh.enforcer.CheckPermissionEx(userID, action, parentResource)
	if err != nil {
		logEntry.WithFields(logrus.Fields{
			"error":      "permission_check_failed",
			"permission": action,
			"resource":   parentResource,
		}).Error("Failed to check permission")
		return fmt.Errorf("failed to check permission: %v", err)
	}

	if !hasPermission {
		logEntry.WithFields(logrus.Fields{
			"permission": action,
			"resource":   parentResource,
			"result":     "denied",
			"reason":     explanations,
		}).Warn("User does not have required permission on parent environment")
		return fmt.Errorf("user %s does not have %s permission on parent environment %s", userID, action, parentResource)
	}

	return nil
}

// trimAPIPrefix removes the API version prefix from the URL path
func (srh *SpecialRouteHandler) trimAPIPrefix(path string) string {
	// Remove /api/v1 prefix
	if strings.HasPrefix(path, "/api/v1") {
		return strings.TrimPrefix(path, "/api/v1")
	}
	return path
}
