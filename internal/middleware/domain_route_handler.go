package middleware

import (
	"fmt"
	"strings"

	"github.com/Gkemhcs/kavach-backend/internal/authz"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// DomainRouteHandler handles authorization for regular domain routes
type DomainRouteHandler struct {
	enforcer *authz.Enforcer
	logger   *logrus.Logger
}

// NewDomainRouteHandler creates a new domain route handler
func NewDomainRouteHandler(enforcer *authz.Enforcer, logger *logrus.Logger) *DomainRouteHandler {
	return &DomainRouteHandler{
		enforcer: enforcer,
		logger:   logger,
	}
}

// HandleDomainRoute handles authorization for regular domain routes
func (drh *DomainRouteHandler) HandleDomainRoute(c *gin.Context, userID string) error {
	logEntry := drh.logger.WithFields(logrus.Fields{
		"operation": "domain_route",
		"user_id":   userID,
		"method":    c.Request.Method,
	})

	path := drh.trimAPIPrefix(c.Request.URL.Path)
	logEntry = logEntry.WithField("path", path)

	// Skip authorization for /by-name and /my routes (allow all users)
	if strings.Contains(path, "/by-name") || strings.Contains(path, "/organizations/my") || strings.HasSuffix(path, "/my") {
		return nil
	}

	// Determine action based on HTTP method
	action := drh.getActionFromMethod(c.Request.Method)

	// Handle resource creation (POST requests without ID at the end) or listing (GET requests to creation paths)
	if drh.isResourceCreation(path) && (c.Request.Method == "POST" || c.Request.Method == "GET") {
		return drh.handleResourceCreation(c, userID, path)
	}

	// Handle regular resource access
	resource := path
	hasPermission, explanations, err := drh.enforcer.CheckPermissionEx(userID, action, resource)
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

// handleResourceCreation handles authorization for resource creation and listing
func (drh *DomainRouteHandler) handleResourceCreation(c *gin.Context, userID string, path string) error {
	logEntry := drh.logger.WithFields(logrus.Fields{
		"operation": "resource_creation",
		"user_id":   userID,
		"method":    c.Request.Method,
		"path":      path,
	})

	// For resource creation/listing, check permission on the parent resource
	parentResource := drh.getParentResource(path)

	// Determine action based on HTTP method
	var action string
	switch c.Request.Method {
	case "GET":
		action = "read" // For listing resources
	case "POST":
		action = "create" // For creating resources
	default:
		action = "read" // Default to read
	}

	hasPermission, explanations, err := drh.enforcer.CheckPermissionEx(userID, action, parentResource)
	if err != nil {
		logEntry.WithFields(logrus.Fields{
			"error":      "permission_check_failed",
			"permission": action,
			"resource":   parentResource,
		}).Error("Failed to check permission")
		return fmt.Errorf("failed to check %s permission: %v", action, err)
	}

	if !hasPermission {
		logEntry.WithFields(logrus.Fields{
			"permission": action,
			"resource":   parentResource,
			"result":     "denied",
			"reason":     explanations,
		}).Warn("User does not have required permission on parent resource")
		return fmt.Errorf("user %s does not have %s permission on parent %s", userID, action, parentResource)
	}

	return nil
}

// isResourceCreation checks if the request is for creating a new resource
func (drh *DomainRouteHandler) isResourceCreation(path string) bool {
	// Resource creation paths end with "/" (trailing slash)
	// Examples: /organizations/, /organizations/123/secret-groups/, etc.
	return strings.HasSuffix(path, "/")
}

// getParentResource gets the parent resource for creation requests
func (drh *DomainRouteHandler) getParentResource(path string) string {
	// Remove trailing slash for creation paths
	path = strings.TrimSuffix(path, "/")

	// If we're creating an organization, there's no parent
	if path == "/organizations" {
		return "/"
	}

	// Parse the path to understand the hierarchy
	parts := strings.Split(path, "/")

	// Handle different resource types
	// Check for environment creation first (more specific path)
	if len(parts) >= 6 && parts[1] == "organizations" && parts[3] == "secret-groups" && parts[5] == "environments" {
		// Creating an environment: /organizations/{orgID}/secret-groups/{secretGroupID}/environments/
		// Parent should be: /organizations/{orgID}/secret-groups/{secretGroupID}
		orgID := parts[2]
		secretGroupID := parts[4]
		parent := fmt.Sprintf("/organizations/%s/secret-groups/%s", orgID, secretGroupID)
		return parent
	}

	if len(parts) >= 4 && parts[1] == "organizations" && parts[3] == "secret-groups" {
		// Creating a secret group: /organizations/{orgID}/secret-groups/
		// Parent should be: /organizations/{orgID}
		orgID := parts[2]
		parent := fmt.Sprintf("/organizations/%s", orgID)
		return parent
	}

	if len(parts) >= 4 && parts[1] == "organizations" && parts[3] == "user-groups" {
		// Creating a user group: /organizations/{orgID}/user-groups/
		// Parent should be: /organizations/{orgID}
		orgID := parts[2]
		parent := fmt.Sprintf("/organizations/%s", orgID)
		return parent
	}

	// Default case: return the path as is (for unknown resource types)
	return path
}

// getActionFromMethod converts HTTP method to action
func (drh *DomainRouteHandler) getActionFromMethod(method string) string {
	switch method {
	case "GET":
		return "read"
	case "POST":
		return "create"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return "read"
	}
}

// trimAPIPrefix removes the API version prefix from the URL path
func (drh *DomainRouteHandler) trimAPIPrefix(path string) string {
	// Remove /api/v1 prefix
	if strings.HasPrefix(path, "/api/v1") {
		return strings.TrimPrefix(path, "/api/v1")
	}
	return path
}
