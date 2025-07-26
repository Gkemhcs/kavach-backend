package authz

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Resolver extracts authorization parameters from HTTP requests
type Resolver struct {
	logger *logrus.Logger
}

// NewResolver creates a new authorization resolver
func NewResolver(logger *logrus.Logger) *Resolver {
	return &Resolver{
		logger: logger,
	}
}

// AuthorizationRequest represents the extracted authorization parameters
type AuthorizationRequest struct {
	Subject Subject
	Object  string
	Action  Action
}

// Resolve extracts authorization parameters from the Gin context
func (r *Resolver) Resolve(c *gin.Context) (*AuthorizationRequest, error) {
	// Extract subject (user or group)
	subject, err := r.extractSubject(c)
	if err != nil {
		return nil, fmt.Errorf("failed to extract subject: %w", err)
	}

	// Extract object (resource path)
	object, err := r.extractObject(c)
	if err != nil {
		return nil, fmt.Errorf("failed to extract object: %w", err)
	}

	// Extract action from HTTP method
	action := GetActionFromMethod(c.Request.Method)

	r.logger.WithFields(logrus.Fields{
		"subject": subject,
		"object":  object,
		"action":  action,
		"method":  c.Request.Method,
		"path":    c.Request.URL.Path,
	}).Debug("Resolved authorization request")

	return &AuthorizationRequest{
		Subject: subject,
		Object:  object,
		Action:  action,
	}, nil
}

// extractSubject extracts the subject (user or group) from the request
func (r *Resolver) extractSubject(c *gin.Context) (Subject, error) {
	// First check if subject is already set in context
	if subject, exists := GetSubject(c); exists {
		return subject, nil
	}

	// Extract user ID from JWT context (JWT middleware only injects user_id)
	userID := GetUserID(c)
	if userID == "" {
		return Subject{}, fmt.Errorf("no user ID found in context")
	}

	// Always create a user subject - group membership will be checked during authorization
	subject := Subject{
		ID:   userID,
		Type: "user",
	}

	// Set subject in context for future use
	SetSubject(c, subject)

	return subject, nil
}

// extractObject extracts the resource object from the request path
func (r *Resolver) extractObject(c *gin.Context) (string, error) {
	path := c.Request.URL.Path

	// Handle IAM routes (grant/revoke permissions) - extract from request body
	if r.isIAMRoute(path) {
		return r.resolveIAMObject(c, path)
	}

	// Handle special cases for by-name routes
	if r.isByNameRoute(path) {
		return r.resolveByNameObject(c, path)
	}

	// Convert path to Casbin object format
	object := r.pathToObject(path)

	// Extract and set organization ID if present
	if orgID := r.extractOrgIDFromPath(path); orgID != "" {
		SetOrgID(c, orgID)
	}

	return object, nil
}

// isByNameRoute checks if the route is a by-name route
func (r *Resolver) isByNameRoute(path string) bool {
	return strings.Contains(path, "/by-name/")
}

// isIAMRoute checks if the route is an IAM route (grant/revoke permissions)
func (r *Resolver) isIAMRoute(path string) bool {
	return strings.Contains(path, "/permissions/grant") || strings.Contains(path, "/permissions/revoke")
}

// resolveIAMObject handles IAM routes by extracting resource info from request body
func (r *Resolver) resolveIAMObject(c *gin.Context, path string) (string, error) {
	// For IAM routes, we need to extract resource information from the request body
	// This is because the resource details are in the JSON body, not the URL path

	// Get the request body
	var requestBody map[string]interface{}
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		return "", fmt.Errorf("failed to parse request body for IAM route: %w", err)
	}

	// Extract resource information from the request body
	resourceType, ok := requestBody["resource_type"].(string)
	if !ok {
		return "", fmt.Errorf("resource_type not found in request body")
	}

	resourceID, ok := requestBody["resource_id"].(string)
	if !ok {
		return "", fmt.Errorf("resource_id not found in request body")
	}

	orgID, ok := requestBody["organization_id"].(string)
	if !ok {
		return "", fmt.Errorf("organization_id not found in request body")
	}

	// Set organization ID in context
	SetOrgID(c, orgID)

	// Build the object path based on resource type
	switch resourceType {
	case "organization":
		return fmt.Sprintf("/organizations/%s/*", resourceID), nil
	case "secret_group":
		return fmt.Sprintf("/organizations/%s/secret-groups/%s/*", orgID, resourceID), nil
	case "environment":
		// For environments, we need secret_group_id from the body
		secretGroupID, ok := requestBody["secret_group_id"].(string)
		if !ok {
			return "", fmt.Errorf("secret_group_id not found in request body for environment resource")
		}
		return fmt.Sprintf("/organizations/%s/secret-groups/%s/environments/%s/*", orgID, secretGroupID, resourceID), nil
	case "user_group":
		return fmt.Sprintf("/organizations/%s/user-groups/%s/*", orgID, resourceID), nil
	default:
		return "", fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// resolveByNameObject handles special authorization for by-name routes
// Users can access by-name routes if they have at least viewer access on any parent or child
func (r *Resolver) resolveByNameObject(c *gin.Context, path string) (string, error) {
	// Extract resource type and name from path
	parts := strings.Split(path, "/")

	// Handle different by-name patterns
	switch {
	case strings.Contains(path, "/organizations/by-name/"):
		// /organizations/by-name/:orgName
		orgName := parts[len(parts)-1]
		return fmt.Sprintf("/organizations/by-name/%s", orgName), nil

	case strings.Contains(path, "/secret-groups/by-name/"):
		// /organizations/:orgID/secret-groups/by-name/:groupName
		orgID := parts[2] // /organizations/:orgID/...
		groupName := parts[len(parts)-1]
		return fmt.Sprintf("/organizations/%s/secret-groups/by-name/%s", orgID, groupName), nil

	case strings.Contains(path, "/environments/by-name/"):
		// /organizations/:orgID/secret-groups/:groupID/environments/by-name/:envName
		orgID := parts[2]
		groupID := parts[4]
		envName := parts[len(parts)-1]
		return fmt.Sprintf("/organizations/%s/secret-groups/%s/environments/by-name/%s", orgID, groupID, envName), nil

	case strings.Contains(path, "/user-groups/by-name"):
		// /organizations/:orgID/user-groups/by-name?name=...
		orgID := parts[2]
		groupName := c.Query("name")
		return fmt.Sprintf("/organizations/%s/user-groups/by-name/%s", orgID, groupName), nil

	default:
		return "", fmt.Errorf("unknown by-name route pattern: %s", path)
	}
}

// pathToObject converts a Gin path to Casbin object format
func (r *Resolver) pathToObject(path string) string {
	// Remove leading slash and convert to Casbin format
	object := strings.TrimPrefix(path, "/")

	// Ensure it starts with a slash for Casbin
	if !strings.HasPrefix(object, "/") {
		object = "/" + object
	}

	return object
}

// extractOrgIDFromPath extracts organization ID from the path
func (r *Resolver) extractOrgIDFromPath(path string) string {
	parts := strings.Split(path, "/")

	// Look for organization ID in common patterns
	for i, part := range parts {
		if part == "organizations" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	return ""
}

// GetSubjectString returns the subject in Casbin format
func (ar *AuthorizationRequest) GetSubjectString() string {
	switch ar.Subject.Type {
	case "user":
		return fmt.Sprintf("user:%s", ar.Subject.ID)
	case "group":
		return fmt.Sprintf("group:%s", ar.Subject.ID)
	default:
		return fmt.Sprintf("user:%s", ar.Subject.ID) // Default to user
	}
}

// GetRoleString returns the role in Casbin format
func (ar *AuthorizationRequest) GetRoleString(resourceType, resourceID, role string) string {
	return fmt.Sprintf("%s:%s:%s", resourceType, resourceID, role)
}
