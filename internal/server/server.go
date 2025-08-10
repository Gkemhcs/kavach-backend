package server

import (
	"database/sql"
	"fmt"

	"github.com/Gkemhcs/kavach-backend/internal/auth"
	"github.com/Gkemhcs/kavach-backend/internal/auth/jwt"
	"github.com/Gkemhcs/kavach-backend/internal/config"
	"github.com/Gkemhcs/kavach-backend/internal/db"
	"github.com/Gkemhcs/kavach-backend/internal/environment"
	"github.com/Gkemhcs/kavach-backend/internal/groups"
	"github.com/Gkemhcs/kavach-backend/internal/iam"
	"github.com/Gkemhcs/kavach-backend/internal/middleware"
	"github.com/Gkemhcs/kavach-backend/internal/org"
	"github.com/Gkemhcs/kavach-backend/internal/provider"
	"github.com/Gkemhcs/kavach-backend/internal/secret"
	"github.com/Gkemhcs/kavach-backend/internal/secretgroup"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Server represents the HTTP server for the Kavach backend API.
type Server struct {
	cfg    *config.Config
	log    *logrus.Logger
	engine *gin.Engine
	db     *sql.DB // Database connection for health checks
}

// SetupRoutes registers all API routes and middleware for the server.
// This function centralizes route registration for maintainability.
func (s *Server) SetupRoutes(authHandler *auth.AuthHandler,
	iamHandler *iam.IamHandler,
	orgHandler *org.OrganizationHandler,
	secretgroupHandler *secretgroup.SecretGroupHandler,
	environmentHandler *environment.EnvironmentHandler,
	userGroupHandler *groups.UserGroupHandler,
	secretHandler *secret.SecretHandler,
	providerHandler *provider.ProviderHandler,
	jwter *jwt.Manager,
	cfg *config.Config,
	logger *logrus.Logger,
	authzMiddleware *middleware.AuthMiddleware) {
	// Create API v1 router group
	v1 := s.engine.Group("/api/v1")

	jwtMiddleware := middleware.JWTAuthMiddleware(jwter)

	// Register auth routes FIRST (no middleware - these are public)
	auth.RegisterAuthRoutes(authHandler, v1)

	// Create a new group for protected routes that need JWT
	protected := v1.Group("")
	protected.Use(jwtMiddleware)
	protected.Use(authzMiddleware.Middleware())
	// Register all other routes under the protected group
	iam.RegisterIamRoutes(iamHandler, protected)
	org.RegisterOrganizationRoutes(orgHandler, protected, secretgroupHandler, environmentHandler,
		userGroupHandler, secretHandler, providerHandler,
		jwtMiddleware)

	// Add other route groups here as needed
	// Example: secrets.RegisterSecretRoutes(secretHandler, v1)
	// Example: orgs.RegisterOrgRoutes(orgHandler, v1)
}

// routes registers health check and other non-API routes.
func (s *Server) routes() {
	s.engine.GET("/healthz", func(c *gin.Context) {
		// Basic health check
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Kavach backend is healthy",
		})
	})

	// Detailed health check with database connection pool stats
	s.engine.GET("/healthz/detailed", func(c *gin.Context) {
		// Check database connectivity
		if err := s.db.Ping(); err != nil {
			c.JSON(503, gin.H{
				"status":  "error",
				"message": "Database connection failed",
				"error":   err.Error(),
			})
			return
		}

		// Get connection pool statistics
		poolStats := db.GetConnectionStats(s.db)

		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Kavach backend is healthy",
			"database": gin.H{
				"status": "connected",
				"pool":   poolStats,
			},
			"timestamp": gin.H{
				"server_time": "2024-01-01T00:00:00Z", // You can add actual timestamp
			},
		})
	})
}

// New creates a new Server instance with the given config and logger.
func New(cfg *config.Config, log *logrus.Logger, db *sql.DB) *Server {
	engine := gin.New()
	engine.Use(gin.Recovery())
	gin.SetMode(gin.ReleaseMode)

	return &Server{
		cfg:    cfg,
		log:    log,
		engine: engine,
		db:     db,
	}
}

// Start runs the HTTP server on the configured port.
func (s *Server) Start() error {
	s.routes()
	addr := fmt.Sprintf(":%s", s.cfg.Port)
	s.log.Infof("starting server on %s", addr)
	return s.engine.Run(addr)
}
