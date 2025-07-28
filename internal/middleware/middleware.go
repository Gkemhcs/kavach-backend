package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Gkemhcs/kavach-backend/internal/authz"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// AuthMiddleware handles authorization for all API endpoints
type AuthMiddleware struct {
	enforcer *authz.Enforcer
	logger   *logrus.Logger
}

// NewAuthMiddleware creates a new authorization middleware
func NewAuthMiddleware(enforcer *authz.Enforcer, logger *logrus.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		enforcer: enforcer,
		logger:   logger,
	}
}

// Middleware is the main authorization middleware function for Gin
func (am *AuthMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create structured log entry with request context
		path := am.trimAPIPrefix(c.Request.URL.Path)
		logEntry := am.logger.WithFields(logrus.Fields{
			"component":   "authz_middleware",
			"request_id":  c.GetString("request_id"),
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"resource":    path,
			"remote_addr": c.ClientIP(),
			"user_agent":  c.Request.UserAgent(),
		})

		logEntry.Info("Processing authorization request")

		// Extract user ID from context (assuming it's set by authentication middleware)
		userID, exists := c.Get("user_id")
		if !exists {
			logEntry.WithField("error", "user_id_not_found").Error("Authorization failed: user ID not found in context")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		userIDStr, ok := userID.(string)
		if !ok {
			logEntry.WithFields(logrus.Fields{
				"error":     "invalid_user_id_type",
				"user_id":   userID,
				"user_type": fmt.Sprintf("%T", userID),
			}).Error("Authorization failed: user ID is not a string")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Add user ID to log context
		logEntry = logEntry.WithField("user_id", userIDStr)

		// Skip authorization for organization creation (anyone can create orgs)
		if am.isOrganizationCreation(c.Request) {
			logEntry.WithField("route_type", "organization_creation").Info("Authorization bypassed for organization creation")
			c.Next()
			return
		}

		// Handle special routes
		if am.isSpecialRoute(c.Request) {
			logEntry.WithFields(logrus.Fields{
				"route_type": "special_route",
				"resource":   path,
			}).Info("Processing special route authorization")
			if err := am.handleSpecialRoute(c, userIDStr); err != nil {
				logEntry.WithFields(logrus.Fields{
					"error":         err.Error(),
					"route_type":    "special_route",
					"resource":      path,
					"authorization": "denied",
				}).Error("Special route authorization failed")
				c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
				c.Abort()
				return
			}
			logEntry.WithFields(logrus.Fields{
				"authorization": "granted",
				"resource":      path,
			}).Info("Special route authorization successful")
			c.Next()
			return
		}

		// Handle regular domain routes
		logEntry.WithFields(logrus.Fields{
			"route_type": "domain_route",
			"resource":   path,
		}).Info("Processing domain route authorization")
		if err := am.handleDomainRoute(c, userIDStr); err != nil {
			logEntry.WithFields(logrus.Fields{
				"error":         err.Error(),
				"route_type":    "domain_route",
				"resource":      path,
				"authorization": "denied",
			}).Error("Domain route authorization failed")
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			c.Abort()
			return
		}

		logEntry.WithFields(logrus.Fields{
			"authorization": "granted",
			"resource":      path,
		}).Info("Domain route authorization successful")
		c.Next()
	}
}

// isOrganizationCreation checks if the request is for creating an organization
func (am *AuthMiddleware) isOrganizationCreation(r *http.Request) bool {
	path := am.trimAPIPrefix(r.URL.Path)
	// Handle both with and without trailing slash
	return r.Method == "POST" && (path == "/organizations" || path == "/organizations/")
}

// isSpecialRoute checks if the request is for a special route that needs custom handling
func (am *AuthMiddleware) isSpecialRoute(r *http.Request) bool {
	path := am.trimAPIPrefix(r.URL.Path)
	return strings.Contains(path, "/permissions/grant") ||
		strings.Contains(path, "/permissions/revoke") ||
		strings.Contains(path, "/members")
}

// handleSpecialRoute handles authorization for special routes
func (am *AuthMiddleware) handleSpecialRoute(c *gin.Context, userID string) error {
	path := am.trimAPIPrefix(c.Request.URL.Path)

	// Handle permissions grant/revoke routes
	if strings.Contains(path, "/permissions/grant") {
		return am.handleGrantPermission(c, userID)
	}

	if strings.Contains(path, "/permissions/revoke") {
		return am.handleRevokePermission(c, userID)
	}

	// Handle user group members route
	if strings.Contains(path, "/members") {
		return am.handleUserGroupMembers(c, userID)
	}

	return fmt.Errorf("unknown special route: %s", path)
}

// handleGrantPermission handles authorization for granting permissions
func (am *AuthMiddleware) handleGrantPermission(c *gin.Context, userID string) error {
	logEntry := am.logger.WithFields(logrus.Fields{
		"operation": "grant_permission",
		"user_id":   userID,
	})

	logEntry.Info("Processing grant permission authorization")

	// Read the request body without consuming it
	body, err := c.GetRawData()
	if err != nil {
		logEntry.WithField("error", "failed_to_read_body").Error("Failed to read request body")
		return fmt.Errorf("failed to read request body: %v", err)
	}

	// Re-set the body so it can be read by subsequent handlers
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// Parse only the fields we need for authorization
	var req struct {
		ResourceType   string        `json:"resource_type"`
		ResourceID     uuid.UUID     `json:"resource_id"`
		OrganizationID uuid.UUID     `json:"organization_id"`
		SecretGroupID  uuid.NullUUID `json:"secret_group_id"`
		EnvironmentID  uuid.NullUUID `json:"environment_id"`
	}

	if err := json.Unmarshal(body, &req); err != nil {
		logEntry.WithFields(logrus.Fields{
			"error": "failed_to_decode_body",
			"body":  string(body),
		}).Error("Failed to decode request body")
		return fmt.Errorf("failed to decode request body: %v", err)
	}

	// Add request details to log context
	logEntry = logEntry.WithFields(logrus.Fields{
		"resource_type":   req.ResourceType,
		"resource_id":     req.ResourceID.String(),
		"organization_id": req.OrganizationID.String(),
		"secret_group_id": req.SecretGroupID.UUID.String(),
		"environment_id":  req.EnvironmentID.UUID.String(),
	})

	logEntry.Info("Request details parsed successfully")

	// Construct resource object based on resource type
	resource, err := am.constructResourceFromRequest(GrantRoleBindingRequest{
		ResourceType:   req.ResourceType,
		ResourceID:     req.ResourceID,
		OrganizationID: req.OrganizationID,
		SecretGroupID:  req.SecretGroupID,
		EnvironmentID:  req.EnvironmentID,
	})
	if err != nil {
		logEntry.WithField("error", "failed_to_construct_resource").Error("Failed to construct resource")
		return fmt.Errorf("failed to construct resource: %v", err)
	}

	logEntry.WithField("resource", resource).Info("Resource constructed successfully")

	// Check if user has grant permission on the resource
	logEntry.WithField("permission", "grant").Info("Checking user permission")
	hasPermission, explanations, err := am.enforcer.CheckPermissionEx(userID, "grant", resource)
	if err != nil {
		logEntry.WithFields(logrus.Fields{
			"error":      "permission_check_failed",
			"permission": "grant",
			"resource":   resource,
		}).Error("Failed to check permission")
		return fmt.Errorf("failed to check permission: %v", err)
	}

	if !hasPermission {
		logEntry.WithFields(logrus.Fields{
			"permission": "grant",
			"resource":   resource,
			"result":     "denied",
			"reason":     explanations,
		}).Warn("User does not have grant permission")
		return fmt.Errorf("user %s does not have grant permission on %s", userID, resource)
	}

	logEntry.WithFields(logrus.Fields{
		"permission": "grant",
		"resource":   resource,
		"result":     "granted",
		"reason":     explanations,
	}).Info("Grant permission authorization successful")
	return nil
}

// handleRevokePermission handles authorization for revoking permissions
func (am *AuthMiddleware) handleRevokePermission(c *gin.Context, userID string) error {
	logEntry := am.logger.WithFields(logrus.Fields{
		"operation": "revoke_permission",
		"user_id":   userID,
	})

	logEntry.Info("Processing revoke permission authorization")

	// Read the request body without consuming it
	body, err := c.GetRawData()
	if err != nil {
		logEntry.WithField("error", "failed_to_read_body").Error("Failed to read request body")
		return fmt.Errorf("failed to read request body: %v", err)
	}

	// Re-set the body so it can be read by subsequent handlers
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// Parse only the fields we need for authorization
	var req struct {
		ResourceType   string        `json:"resource_type"`
		ResourceID     uuid.UUID     `json:"resource_id"`
		OrganizationID uuid.UUID     `json:"organization_id"`
		SecretGroupID  uuid.NullUUID `json:"secret_group_id"`
		EnvironmentID  uuid.NullUUID `json:"environment_id"`
	}

	if err := json.Unmarshal(body, &req); err != nil {
		logEntry.WithFields(logrus.Fields{
			"error": "failed_to_decode_body",
			"body":  string(body),
		}).Error("Failed to decode request body")
		return fmt.Errorf("failed to decode request body: %v", err)
	}

	// Add request details to log context
	logEntry = logEntry.WithFields(logrus.Fields{
		"resource_type":   req.ResourceType,
		"resource_id":     req.ResourceID.String(),
		"organization_id": req.OrganizationID.String(),
		"secret_group_id": req.SecretGroupID.UUID.String(),
		"environment_id":  req.EnvironmentID.UUID.String(),
	})

	logEntry.Info("Request details parsed successfully")

	// Construct resource object based on resource type
	resource, err := am.constructResourceFromRequest(GrantRoleBindingRequest{
		ResourceType:   req.ResourceType,
		ResourceID:     req.ResourceID,
		OrganizationID: req.OrganizationID,
		SecretGroupID:  req.SecretGroupID,
		EnvironmentID:  req.EnvironmentID,
	})
	if err != nil {
		logEntry.WithField("error", "failed_to_construct_resource").Error("Failed to construct resource")
		return fmt.Errorf("failed to construct resource: %v", err)
	}

	logEntry.WithField("resource", resource).Info("Resource constructed successfully")

	// Check if user has revoke permission on the resource
	logEntry.WithField("permission", "revoke").Info("Checking user permission")
	hasPermission, explanations, err := am.enforcer.CheckPermissionEx(userID, "revoke", resource)
	if err != nil {
		logEntry.WithFields(logrus.Fields{
			"error":      "permission_check_failed",
			"permission": "revoke",
			"resource":   resource,
		}).Error("Failed to check permission")
		return fmt.Errorf("failed to check permission: %v", err)
	}

	if !hasPermission {
		logEntry.WithFields(logrus.Fields{
			"permission": "revoke",
			"resource":   resource,
			"result":     "denied",
			"reason":     explanations,
		}).Warn("User does not have revoke permission")
		return fmt.Errorf("user %s does not have revoke permission on %s", userID, resource)
	}

	logEntry.WithFields(logrus.Fields{
		"permission": "revoke",
		"resource":   resource,
		"result":     "granted",
		"reason":     explanations,
	}).Info("Revoke permission authorization successful")
	return nil
}

// handleUserGroupMembers handles authorization for user group members operations
func (am *AuthMiddleware) handleUserGroupMembers(c *gin.Context, userID string) error {
	logEntry := am.logger.WithFields(logrus.Fields{
		"operation": "user_group_members",
		"user_id":   userID,
		"method":    c.Request.Method,
	})

	logEntry.Info("Processing user group members authorization")

	path := am.trimAPIPrefix(c.Request.URL.Path)
	logEntry = logEntry.WithField("path", path)

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

	logEntry = logEntry.WithFields(logrus.Fields{
		"organization_id": orgID,
		"user_group_id":   userGroupID,
		"resource":        resource,
	})

	logEntry.Info("Path parsed successfully")

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

	logEntry = logEntry.WithField("action", action)
	logEntry.Info("Action determined successfully")

	// Check if user has permission on the user group
	logEntry.Info("Checking user permission")
	hasPermission, explanations, err := am.enforcer.CheckPermissionEx(userID, action, resource)
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

	logEntry.WithFields(logrus.Fields{
		"permission": action,
		"resource":   resource,
		"result":     "granted",
		"reason":     explanations,
	}).Info("User group members authorization successful")
	return nil
}

// handleDomainRoute handles authorization for regular domain routes
func (am *AuthMiddleware) handleDomainRoute(c *gin.Context, userID string) error {
	logEntry := am.logger.WithFields(logrus.Fields{
		"operation": "domain_route",
		"user_id":   userID,
		"method":    c.Request.Method,
	})

	path := am.trimAPIPrefix(c.Request.URL.Path)
	logEntry = logEntry.WithField("path", path)

	logEntry.Info("Processing domain route authorization")

	// Skip authorization for /by-name and /my routes (allow all users)
	if strings.Contains(path, "/by-name") || strings.Contains(path, "/organizations/my") || strings.HasSuffix(path, "/my") {
		logEntry.WithField("route_type", "public_route").Info("Authorization bypassed for public route")
		return nil
	}

	// Determine action based on HTTP method
	action := am.getActionFromMethod(c.Request.Method)
	logEntry = logEntry.WithField("action", action)

	logEntry.Info("Action determined successfully")

	// Handle resource creation (POST requests without ID at the end) or listing (GET requests to creation paths)
	if am.isResourceCreation(path) && (c.Request.Method == "POST" || c.Request.Method == "GET") {
		logEntry.WithField("route_type", "resource_creation").Info("Detected resource creation/listing request")
		return am.handleResourceCreation(c, userID, path)
	}

	// Handle regular resource access
	resource := path
	logEntry = logEntry.WithField("resource", resource)

	logEntry.Info("Checking user permission")
	hasPermission, explanations, err := am.enforcer.CheckPermissionEx(userID, action, resource)
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

	logEntry.WithFields(logrus.Fields{
		"permission": action,
		"resource":   resource,
		"result":     "granted",
		"reason":     explanations,
	}).Info("Domain route authorization successful")
	return nil
}

// handleResourceCreation handles authorization for resource creation and listing
func (am *AuthMiddleware) handleResourceCreation(c *gin.Context, userID string, path string) error {
	logEntry := am.logger.WithFields(logrus.Fields{
		"operation": "resource_creation",
		"user_id":   userID,
		"method":    c.Request.Method,
		"path":      path,
	})

	logEntry.Info("Processing resource creation/listing authorization")

	// For resource creation/listing, check permission on the parent resource
	parentResource := am.getParentResource(path)
	logEntry = logEntry.WithField("parent_resource", parentResource)

	logEntry.Info("Parent resource determined successfully")

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

	logEntry = logEntry.WithField("action", action)

	logEntry.Info("Checking user permission on parent resource")
	hasPermission, explanations, err := am.enforcer.CheckPermissionEx(userID, action, parentResource)
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

	logEntry.WithFields(logrus.Fields{
		"permission": action,
		"resource":   parentResource,
		"result":     "granted",
		"reason":     explanations,
	}).Info("Resource creation/listing authorization successful")
	return nil
}

// isResourceCreation checks if the request is for creating a new resource
func (am *AuthMiddleware) isResourceCreation(path string) bool {
	// Resource creation paths end with "/" (trailing slash)
	// Examples: /organizations/, /organizations/123/secret-groups/, etc.
	return strings.HasSuffix(path, "/")
}

// getParentResource gets the parent resource for creation requests
func (am *AuthMiddleware) getParentResource(path string) string {
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
func (am *AuthMiddleware) getActionFromMethod(method string) string {
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

// constructResourceFromRequest constructs resource path from request body
func (am *AuthMiddleware) constructResourceFromRequest(req GrantRoleBindingRequest) (string, error) {
	switch req.ResourceType {
	case "organization":
		if req.OrganizationID == uuid.Nil {
			return "", fmt.Errorf("organization_id is required for organization resource type")
		}
		return fmt.Sprintf("/organizations/%s", req.OrganizationID), nil

	case "secret_group":
		if req.OrganizationID == uuid.Nil || req.ResourceID == uuid.Nil {
			return "", fmt.Errorf("organization_id and resource_id are required for secret_group resource type")
		}
		return fmt.Sprintf("/organizations/%s/secret-groups/%s", req.OrganizationID, req.ResourceID), nil

	case "environment":
		if req.OrganizationID == uuid.Nil || !req.SecretGroupID.Valid || !req.EnvironmentID.Valid {
			return "", fmt.Errorf("organization_id, secret_group_id, and environment_id are required for environment resource type")
		}
		return fmt.Sprintf("/organizations/%s/secret-groups/%s/environments/%s",
			req.OrganizationID, req.SecretGroupID.UUID, req.EnvironmentID.UUID), nil

	default:
		return "", fmt.Errorf("unsupported resource type: %s", req.ResourceType)
	}
}

// trimAPIPrefix removes the API version prefix from the URL path
func (am *AuthMiddleware) trimAPIPrefix(path string) string {
	// Remove /api/v1 prefix
	if strings.HasPrefix(path, "/api/v1") {
		return strings.TrimPrefix(path, "/api/v1")
	}
	return path
}
