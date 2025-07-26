package authz

import (
	"database/sql"
	"log"

	"github.com/Gkemhcs/kavach-backend/internal/iam"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ExampleIntegration shows how to integrate the authorization system
// with your existing Kavach backend
func ExampleIntegration() {
	// Initialize database connection (replace with your actual DB setup)
	db, err := sql.Open("postgres", "your-database-url")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Initialize authorization system
	authSystem, err := NewSystem(db, nil, logger)
	if err != nil {
		log.Fatal("Failed to initialize authorization system:", err)
	}

	// Setup Gin router
	router := gin.Default()

	// Apply authorization middleware to all API routes
	authSystem.SetupRoutes(router)

	// Initialize your existing services (replace with actual constructors)
	// orgService := org.NewOrganizationService(querier, logger, iamService)
	// secretGroupService := secretgroup.NewSecretGroupService(querier, logger, iamService)
	// environmentService := environment.NewEnvironmentService(querier, logger, iamService)
	// userGroupService := groups.NewUserGroupService(logger, querier, userInfoGetter)
	// iamService := iam.NewIamService(querier, userResolver, userGroupResolver, logger)

	// Initialize handlers (replace with actual constructors)
	// orgHandler := org.NewOrganizationHandler(orgService, logger)
	// secretGroupHandler := secretgroup.NewSecretGroupHandler(secretGroupService, logger)
	// environmentHandler := environment.NewEnvironmentHandler(environmentService, logger)
	// userGroupHandler := groups.NewUserGroupHandler(logger, userGroupService)
	// iamHandler := iam.NewIamHandler(iamService, logger)

	// Setup JWT middleware (your existing auth middleware)
	// jwtMiddleware := auth.JWTMiddleware()

	// Register routes (your existing route registration)
	// apiGroup := router.Group("/api/v1")

	// Auth routes (no authorization needed)
	// authGroup := apiGroup.Group("/auth")
	// auth.RegisterAuthRoutes(authHandler, authGroup)

	// IAM routes (for granting/revoking roles)
	// iam.RegisterIamRoutes(iamHandler, apiGroup)

	// Organization routes (with authorization middleware)
	// org.RegisterOrganizationRoutes(
	// 	orgHandler,
	// 	apiGroup,
	// 	secretGroupHandler,
	// 	environmentHandler,
	// 	userGroupHandler,
	// 	jwtMiddleware,
	// )

	// Start server
	logger.Info("Starting Kavach backend with authorization system")
	router.Run(":8080")
}

// ExampleIAMHandler shows how to update your IAM handler to use the authz system
type ExampleIAMHandler struct {
	iamService *iam.IamService
	authSystem *System
	logger     *logrus.Logger
}

func NewExampleIAMHandler(iamService *iam.IamService, authSystem *System, logger *logrus.Logger) *ExampleIAMHandler {
	return &ExampleIAMHandler{
		iamService: iamService,
		authSystem: authSystem,
		logger:     logger,
	}
}

// GrantRoleBinding shows how to use the authz system in your IAM handler
func (h *ExampleIAMHandler) GrantRoleBinding(c *gin.Context) {
	var req iam.GrantRoleBindingRequest
	if err := c.ShouldBindBodyWithJSON(&req); err != nil {
		utils.RespondError(c, 422, "invalid_request_body", "unable to parse request body")
		return
	}

	// Validate request (your existing validation logic)
	if req.UserName != "" && req.GroupName != "" {
		utils.RespondError(c, 409, "only_one_of_users_and_user_groups", "only one of username and usergroup are allowed")
		return
	}

	if req.GroupName == "" && req.UserName == "" {
		utils.RespondError(c, 400, "none_of_user_and_usr_group_passed", "none of user and user groups are passed")
		return
	}

	// Use the authorization system to grant the role
	err := h.authSystem.GrantRole(
		req.UserName,                // userID (empty string if not provided)
		req.GroupName,               // groupID (empty string if not provided)
		req.Role,                    // role
		req.ResourceType,            // resourceType
		req.ResourceID.String(),     // resourceID
		req.OrganizationID.String(), // organizationID
	)

	if err != nil {
		h.logger.WithError(err).Error("Failed to grant role binding")
		utils.RespondError(c, 500, "internal_server_error", "failed to grant role")
		return
	}

	utils.RespondSuccess(c, 200, map[string]string{
		"message": "role granted successfully",
	})
}

// RevokeRoleBinding shows how to use the authz system to revoke roles
func (h *ExampleIAMHandler) RevokeRoleBinding(c *gin.Context) {
	var req iam.RevokeRoleBindingRequest
	if err := c.ShouldBindBodyWithJSON(&req); err != nil {
		utils.RespondError(c, 422, "invalid_request_body", "unable to parse request body")
		return
	}

	// Validate request (your existing validation logic)
	if req.UserName != "" && req.GroupName != "" {
		utils.RespondError(c, 409, "only_one_of_users_and_user_groups", "only one of username and usergroup are allowed")
		return
	}

	if req.GroupName == "" && req.UserName == "" {
		utils.RespondError(c, 400, "none_of_user_and_usr_group_passed", "none of user and user groups are passed")
		return
	}

	// Use the authorization system to revoke the role
	err := h.authSystem.RevokeRole(
		req.UserName,            // userID (empty string if not provided)
		req.GroupName,           // groupID (empty string if not provided)
		req.Role,                // role
		req.ResourceType,        // resourceType
		req.ResourceID.String(), // resourceID
	)

	if err != nil {
		h.logger.WithError(err).Error("Failed to revoke role binding")
		utils.RespondError(c, 500, "internal_server_error", "failed to revoke role")
		return
	}

	utils.RespondSuccess(c, 200, map[string]string{
		"message": "role revoked successfully",
	})
}

// ExampleUsage shows how to use the authorization system programmatically
func ExampleUsage(authSystem *System) {
	// Grant admin role to user on organization
	err := authSystem.GrantRole(
		"user-uuid-123", // userID
		"",              // groupID (empty for user)
		"admin",         // role
		"organization",  // resourceType
		"org-uuid-456",  // resourceID
		"org-uuid-456",  // organizationID
	)
	if err != nil {
		log.Printf("Failed to grant role: %v", err)
	}

	// Grant viewer role to group on secret group
	err = authSystem.GrantRole(
		"",                  // userID (empty for group)
		"group-uuid-789",    // groupID
		"viewer",            // role
		"secret_group",      // resourceType
		"secret-group-uuid", // resourceID
		"org-uuid-456",      // organizationID
	)
	if err != nil {
		log.Printf("Failed to grant role: %v", err)
	}

	// Check if user has permission
	allowed, err := authSystem.CheckPermission(
		"user-uuid-123",
		"/organizations/org-uuid-456/secret-groups/secret-group-uuid",
		"read",
	)
	if err != nil {
		log.Printf("Failed to check permission: %v", err)
	} else {
		log.Printf("User has read permission: %v", allowed)
	}

	// Get all roles for a user
	roles, err := authSystem.GetUserRoles("user-uuid-123")
	if err != nil {
		log.Printf("Failed to get user roles: %v", err)
	} else {
		log.Printf("User roles: %v", roles)
	}

	// Sync policies from database (useful after manual DB changes)
	err = authSystem.SyncPolicies()
	if err != nil {
		log.Printf("Failed to sync policies: %v", err)
	}
}
