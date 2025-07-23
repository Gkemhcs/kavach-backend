package main

import (
	"time"

	"github.com/Gkemhcs/kavach-backend/internal/auth"
	userdb "github.com/Gkemhcs/kavach-backend/internal/auth/gen"
	"github.com/Gkemhcs/kavach-backend/internal/auth/jwt"
	"github.com/Gkemhcs/kavach-backend/internal/auth/provider"
	"github.com/Gkemhcs/kavach-backend/internal/config"
	"github.com/Gkemhcs/kavach-backend/internal/db"

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

	// Organization service and handler setup
	orgService := org.NewOrganizationService(orgdb.New(dbConn), logger)
	orgHandler := org.NewOrganizationHandler(orgService, logger)

	// Create the HTTP server
	s := server.New(cfg, logger)

	// Register all routes (auth, org, etc.)
	s.SetupRoutes(authHandler, orgHandler, dbConn, jwter)

	// Start the server and log fatal on error
	if err := s.Start(); err != nil {
		logger.Fatal("server failed to start", err)
	}
}
