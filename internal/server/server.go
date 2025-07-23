package server

import (
	"database/sql"
	"fmt"

	"github.com/Gkemhcs/kavach-backend/internal/auth"
	"github.com/Gkemhcs/kavach-backend/internal/auth/jwt"
	"github.com/Gkemhcs/kavach-backend/internal/config"
	environmentdb "github.com/Gkemhcs/kavach-backend/internal/environment/gen"
	"github.com/Gkemhcs/kavach-backend/internal/middleware"
	"github.com/Gkemhcs/kavach-backend/internal/org"
	secretgroupdb "github.com/Gkemhcs/kavach-backend/internal/secretgroup/gen"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Server represents the HTTP server for the Kavach backend API.
type Server struct {
	cfg    *config.Config
	log    *logrus.Logger
	engine *gin.Engine
}

// SetupRoutes registers all API routes and middleware for the server.
// This function centralizes route registration for maintainability.
func (s *Server) SetupRoutes(authHandler *auth.AuthHandler,orgHandler *org.OrganizationHandler,dbConn *sql.DB,jwter *jwt.Manager) {
	// Create API v1 router group
	v1 := s.engine.Group("/api/v1")
	jwtMiddleware:=middleware.JWTAuthMiddleware(jwter)

	// Register auth routes
	auth.RegisterAuthRoutes(authHandler, v1)
	
	org.RegisterOrganizationRoutes(orgHandler,v1,
		secretgroupdb.New(dbConn),
	environmentdb.New(dbConn),jwtMiddleware)
	// Add other route groups here as needed
	// Example: secrets.RegisterSecretRoutes(secretHandler, v1)
	// Example: orgs.RegisterOrgRoutes(orgHandler, v1)
}

// routes registers health check and other non-API routes.
func (s *Server) routes() {
	s.engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Kavach backend is healthy",
		})
	})
}

// New creates a new Server instance with the given config and logger.
func New(cfg *config.Config, log *logrus.Logger) *Server {
	engine := gin.New()
	engine.Use(gin.Recovery())
	gin.SetMode(gin.ReleaseMode)

	return &Server{
		cfg:    cfg,
		log:    log,
		engine: engine,
	}
}

// Start runs the HTTP server on the configured port.
func (s *Server) Start() error {
	s.routes()
	addr := fmt.Sprintf(":%s", s.cfg.Port)
	s.log.Infof("starting server on %s", addr)
	return s.engine.Run(addr)
}
