package main

import (
	"time"

	"github.com/Gkemhcs/kavach-backend/internal/auth"
	userdb "github.com/Gkemhcs/kavach-backend/internal/auth/gen"
	"github.com/Gkemhcs/kavach-backend/internal/auth/jwt"
	"github.com/Gkemhcs/kavach-backend/internal/auth/provider"
	"github.com/Gkemhcs/kavach-backend/internal/config"
	"github.com/Gkemhcs/kavach-backend/internal/db"
	"github.com/Gkemhcs/kavach-backend/internal/environment"
	environmentdb "github.com/Gkemhcs/kavach-backend/internal/environment/gen"
	"github.com/Gkemhcs/kavach-backend/internal/groups"
	groupsdb "github.com/Gkemhcs/kavach-backend/internal/groups/gen"
	"github.com/Gkemhcs/kavach-backend/internal/iam"
	iam_db "github.com/Gkemhcs/kavach-backend/internal/iam/gen"
	"github.com/Gkemhcs/kavach-backend/internal/secretgroup"
	secretgroupdb "github.com/Gkemhcs/kavach-backend/internal/secretgroup/gen"

	"github.com/Gkemhcs/kavach-backend/internal/org"
	orgdb "github.com/Gkemhcs/kavach-backend/internal/org/gen"

	"github.com/Gkemhcs/kavach-backend/internal/server"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
)

// main is the entry point for the Kavach backend server.
func main() {
	// Load configuration from environment or config file
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Initialize logger
	logger := utils.New(cfg)

	// Initialize GitHub OAuth provider
	githubProvider := provider.NewGitHubProvider(
		cfg.GitHubClientID,
		cfg.GitHubClientSecret,
		cfg.GitHubRedirectURL,
	)
	// Initialize database connection
	// dbConn is used for user and organization data

	dbConn := db.InitDB(logger, cfg)

	// JWT manager setup for access and refresh tokens
	jwter := jwt.NewManager(
		cfg.JWTSecret,
		time.Duration(cfg.AccessTokenDuration)*time.Minute,
		time.Duration(cfg.RefreshTokenDuration)*time.Minute,
	)
	// Auth service and handler setup
	authService := auth.NewAuthService(githubProvider, userdb.New(dbConn), jwter, logger)
	authHandler := auth.NewAuthHandler(authService, logger)

	//UserGroup service and handler setup
	userGroupService := groups.NewUserGroupService(logger, groupsdb.New(dbConn), authService)
	userGroupHandler := groups.NewUserGroupHandler(logger, userGroupService)

	//IamService setup

	iamService := iam.NewIamService(iam_db.New(dbConn), authService, userGroupService, logger)
	iamHandler := iam.NewIamHandler(*iamService, logger)
	// Organization service and handler setup
	orgService := org.NewOrganizationService(orgdb.New(dbConn), logger, *iamService)
	orgHandler := org.NewOrganizationHandler(orgService, logger)

	//SecretGroup service and handler setup
	groupService := secretgroup.NewSecretGroupService(secretgroupdb.New(dbConn), logger, *iamService)
	groupHandler := secretgroup.NewSecretGroupHandler(groupService, logger)

	//Environment service and handler setup
	environmentService := environment.NewEnvironmentService(environmentdb.New(dbConn), logger, *iamService)
	environmentHandler := environment.NewEnvironmentHandler(environmentService, logger)

	// Create the HTTP server
	s := server.New(cfg, logger)

	// Register all routes (auth, org, etc.)
	s.SetupRoutes(authHandler, iamHandler, orgHandler, groupHandler, environmentHandler, userGroupHandler, jwter,dbConn,cfg)

	// Start the server and log fatal on error
	if err := s.Start(); err != nil {
		logger.Fatal("server failed to start", err)
	}
}
