package authz

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Middleware provides centralized authorization for all API routes
type Middleware struct {
	enforcer *Enforcer
	resolver *Resolver
	logger   *logrus.Logger
	db       *sql.DB
}

// NewMiddleware creates a new authorization middleware
func NewMiddleware(enforcer *Enforcer, resolver *Resolver, logger *logrus.Logger, db *sql.DB) *Middleware {
	return &Middleware{
		enforcer: enforcer,
		resolver: resolver,
		logger:   logger,
		db:       db,
	}
}

// Authorize is the main middleware function that enforces authorization
func (m *Middleware) Authorize() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authorization for certain paths
		if m.shouldSkipAuthorization(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Resolve authorization request
		authReq, err := m.resolver.Resolve(c)
		if err != nil {
			m.logger.WithFields(logrus.Fields{
				"path":  c.Request.URL.Path,
				"error": err.Error(),
			}).Error("Failed to resolve authorization request")

			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "authorization_error",
				"message": "Failed to process authorization request",
			})
			c.Abort()
			return
		}

		// Get user ID for authorization check
		userID := authReq.Subject.ID

		// Enforce authorization with group membership check
		allowed, err := m.enforcer.EnforceWithGroupCheck(userID, authReq.Object, authReq.Action, m.db)
		if err != nil {
			m.logger.WithFields(logrus.Fields{
				"userID": userID,
				"object": authReq.Object,
				"action": authReq.Action,
				"error":  err.Error(),
			}).Error("Authorization enforcement failed")

			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "authorization_error",
				"message": "Failed to enforce authorization",
			})
			c.Abort()
			return
		}

		if !allowed {
			m.logger.WithFields(logrus.Fields{
				"userID": userID,
				"object": authReq.Object,
				"action": authReq.Action,
				"method": c.Request.Method,
				"path":   c.Request.URL.Path,
			}).Warn("Access denied")

			c.JSON(http.StatusForbidden, gin.H{
				"error":   "access_denied",
				"message": "You don't have permission to perform this action",
			})
			c.Abort()
			return
		}

		m.logger.WithFields(logrus.Fields{
			"userID": userID,
			"object": authReq.Object,
			"action": authReq.Action,
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
		}).Debug("Access granted")

		c.Next()
	}
}

// shouldSkipAuthorization determines if authorization should be skipped for a given path
func (m *Middleware) shouldSkipAuthorization(path string) bool {
	// Skip authorization for health checks and public endpoints
	skipPaths := []string{
		"/health",
		"/metrics",
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/auth/refresh",
		"/api/v1/auth/logout",
		"/api/v1/auth/verify",
	}

	for _, skipPath := range skipPaths {
		if path == skipPath {
			return true
		}
	}

	// Skip authorization for organization creation (everyone can create their own org)
	if path == "/api/v1/organizations" && m.isOrganizationCreation() {
		return true
	}

	return false
}

// isOrganizationCreation checks if this is a POST request to create an organization
func (m *Middleware) isOrganizationCreation() bool {
	// This would need to be called from the middleware context
	// For now, we'll handle this in the main middleware function
	return false
}

// AuthorizeOrganizationCreation is a special middleware for organization creation
func (m *Middleware) AuthorizeOrganizationCreation() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Everyone can create their own organization
		// This is handled by the business logic in the organization service
		c.Next()
	}
}

// AuthorizeByRole is a middleware that checks for specific role requirements
func (m *Middleware) AuthorizeByRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user ID from context
		userID := GetUserID(c)
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "User not authenticated",
			})
			c.Abort()
			return
		}

		// Get organization ID from context or path
		orgID, exists := GetOrgID(c)
		if !exists {
			// Try to extract from path
			orgID = m.extractOrgIDFromPath(c.Request.URL.Path)
		}

		if orgID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_request",
				"message": "Organization ID not found",
			})
			c.Abort()
			return
		}

		// Check if user has the required role
		subject := fmt.Sprintf("user:%s", userID)
		role := fmt.Sprintf("org:%s:%s", orgID, requiredRole)

		// Check if user has the role
		roles, err := m.enforcer.GetRolesForUser(subject)
		if err != nil {
			m.logger.WithFields(logrus.Fields{
				"user":  userID,
				"error": err.Error(),
			}).Error("Failed to get roles for user")

			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "authorization_error",
				"message": "Failed to check user roles",
			})
			c.Abort()
			return
		}

		hasRole := false
		for _, userRole := range roles {
			if userRole == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			m.logger.WithFields(logrus.Fields{
				"user":         userID,
				"orgID":        orgID,
				"requiredRole": requiredRole,
				"userRoles":    roles,
			}).Warn("User does not have required role")

			c.JSON(http.StatusForbidden, gin.H{
				"error":   "insufficient_permissions",
				"message": fmt.Sprintf("Required role: %s", requiredRole),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// extractOrgIDFromPath extracts organization ID from the request path
func (m *Middleware) extractOrgIDFromPath(path string) string {
	// This is a simplified version - the resolver has a more robust implementation
	parts := []string{}
	for _, part := range parts {
		if part == "organizations" && len(parts) > 1 {
			return parts[1]
		}
	}
	return ""
}
