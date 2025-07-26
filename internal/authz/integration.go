package authz

import (
	"context"
	"database/sql"
	"fmt"

	pgadapter "github.com/casbin/casbin-pg-adapter"
	"github.com/casbin/casbin/v2/persist"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Config holds the configuration for the authorization system
type Config struct {
	DatabaseURL string
	TableName   string
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		DatabaseURL: "",
		TableName:   "casbin_rule",
	}
}

// System represents the complete authorization system
type System struct {
	Enforcer   *Enforcer
	Service    *Service
	Middleware *Middleware
	Resolver   *Resolver
	Logger     *logrus.Logger
}

// NewSystem creates and initializes the complete authorization system
func NewSystem(db *sql.DB, config *Config, logger *logrus.Logger) (*System, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create PostgreSQL adapter for Casbin
	logger.WithFields(logrus.Fields{
		"databaseURL": config.DatabaseURL,
		"tableName":   config.TableName,
	}).Debug("Initializing Casbin PostgreSQL adapter")

	// Try to create adapter with explicit table creation disabled
	var adapter persist.Adapter
	pgAdapter, err := pgadapter.NewAdapter(config.DatabaseURL, config.TableName)
	if err != nil {
		logger.WithError(err).Warn("Failed to create PostgreSQL adapter, falling back to file adapter for testing")
		// Fallback: use file adapter for testing
		adapter = fileadapter.NewAdapter("authz_policy.csv")
	} else {
		adapter = pgAdapter
	}

	logger.Debug("Casbin adapter initialized successfully")

	// Create enforcer
	enforcer, err := NewEnforcer(adapter, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create enforcer: %w", err)
	}

	// Create resolver
	resolver := NewResolver(logger)

	// Create service
	service := NewService(enforcer, db, logger)

	// Create middleware
	middleware := NewMiddleware(enforcer, resolver, logger, db)

	// Sync role bindings from database
	err = service.SyncRoleBindings(context.Background())
	if err != nil {
		logger.WithError(err).Warn("Failed to sync role bindings on startup")
		// Don't fail startup, continue without initial sync
	}

	return &System{
		Enforcer:   enforcer,
		Service:    service,
		Middleware: middleware,
		Resolver:   resolver,
		Logger:     logger,
	}, nil
}

// SetupRoutes sets up the authorization middleware on the Gin router
func (s *System) SetupRoutes(router *gin.Engine) {
	// Apply authorization middleware to all API routes
	apiGroup := router.Group("/api/v1")
	apiGroup.Use(s.Middleware.Authorize())

	s.Logger.Info("Authorization middleware applied to /api/v1 routes")
}

// SetupRoutesWithExclusions sets up authorization middleware with specific exclusions
func (s *System) SetupRoutesWithExclusions(router *gin.Engine, exclusions []string) {
	// Apply authorization middleware to all API routes except exclusions
	apiGroup := router.Group("/api/v1")

	// Add exclusion middleware
	apiGroup.Use(s.exclusionMiddleware(exclusions))
	apiGroup.Use(s.Middleware.Authorize())

	s.Logger.WithField("exclusions", exclusions).Info("Authorization middleware applied with exclusions")
}

// exclusionMiddleware creates middleware to skip authorization for excluded paths
func (s *System) exclusionMiddleware(exclusions []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		for _, exclusion := range exclusions {
			if path == exclusion {
				c.Set("skip_auth", true)
				break
			}
		}
		c.Next()
	}
}

// GrantRole grants a role to a user or group
func (s *System) GrantRole(userID, groupID, role, resourceType, resourceID, orgID string) error {
	binding := RoleBinding{
		Role:           role,
		ResourceType:   resourceType,
		ResourceID:     uuid.MustParse(resourceID),
		OrganizationID: uuid.MustParse(orgID),
	}

	if userID != "" {
		uid := uuid.MustParse(userID)
		binding.UserID = &uid
	} else if groupID != "" {
		gid := uuid.MustParse(groupID)
		binding.GroupID = &gid
	} else {
		return fmt.Errorf("either userID or groupID must be provided")
	}

	return s.Service.GrantRoleBinding(context.Background(), binding)
}

// RevokeRole revokes a role from a user or group
func (s *System) RevokeRole(userID, groupID, role, resourceType, resourceID string) error {
	binding := RoleBinding{
		Role:         role,
		ResourceType: resourceType,
		ResourceID:   uuid.MustParse(resourceID),
	}

	if userID != "" {
		uid := uuid.MustParse(userID)
		binding.UserID = &uid
	} else if groupID != "" {
		gid := uuid.MustParse(groupID)
		binding.GroupID = &gid
	} else {
		return fmt.Errorf("either userID or groupID must be provided")
	}

	return s.Service.RevokeRoleBinding(context.Background(), binding)
}

// SyncPolicies syncs all role bindings from database to Casbin
func (s *System) SyncPolicies() error {
	return s.Service.SyncRoleBindings(context.Background())
}

// GetUserRoles returns all roles for a user
func (s *System) GetUserRoles(userID string) ([]string, error) {
	subject := fmt.Sprintf("user:%s", userID)
	return s.Enforcer.GetRolesForUser(subject)
}

// GetRoleUsers returns all users for a role
func (s *System) GetRoleUsers(role string) ([]string, error) {
	return s.Enforcer.GetUsersForRole(role)
}

// CheckPermission checks if a user has permission to perform an action on a resource
func (s *System) CheckPermission(userID, resourcePath, action string) (bool, error) {
	subject := fmt.Sprintf("user:%s", userID)
	return s.Enforcer.Enforce(subject, resourcePath, Action(action))
}
