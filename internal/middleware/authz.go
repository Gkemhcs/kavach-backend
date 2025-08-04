package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Gkemhcs/kavach-backend/internal/authz"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AuthMiddleware handles authorization for all API endpoints
type AuthMiddleware struct {
	enforcer authz.Enforcer
	logger   *logrus.Logger
	// Specialized handlers
	domainHandler     *DomainRouteHandler
	specialHandler    *SpecialRouteHandler
	permissionHandler *PermissionHandler
}

// NewAuthMiddleware creates a new authorization middleware
func NewAuthMiddleware(enforcer authz.Enforcer, logger *logrus.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		enforcer:          enforcer,
		logger:            logger,
		domainHandler:     NewDomainRouteHandler(enforcer, logger),
		specialHandler:    NewSpecialRouteHandler(enforcer, logger),
		permissionHandler: NewPermissionHandler(enforcer, logger),
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
			c.Next()
			return
		}

		// Handle special routes
		if am.isSpecialRoute(c.Request) {
			if err := am.specialHandler.HandleSpecialRoute(c, userIDStr); err != nil {
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
			c.Next()
			return
		}

		// Handle regular domain routes
		if err := am.domainHandler.HandleDomainRoute(c, userIDStr); err != nil {
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
		strings.Contains(path, "/members") ||
		strings.Contains(path, "/secrets") ||
		strings.Contains(path, "/providers")
}

// trimAPIPrefix removes the API version prefix from the URL path
func (am *AuthMiddleware) trimAPIPrefix(path string) string {
	// Remove /api/v1 prefix
	if strings.HasPrefix(path, "/api/v1") {
		return strings.TrimPrefix(path, "/api/v1")
	}
	return path
}
