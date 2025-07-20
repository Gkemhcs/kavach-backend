package server

import (
	"fmt"

	"github.com/Gkemhcs/kavach-backend/internal/auth"
	"github.com/Gkemhcs/kavach-backend/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Server struct {
	cfg    *config.Config
	log    *logrus.Logger
	engine *gin.Engine
}

func (s *Server) SetupRoutes(authHandler *auth.AuthHandler) {
	// Create API v1 router group
	v1 := s.engine.Group("/api/v1")
	
	// Register auth routes
	auth.RegisterAuthRoutes(authHandler, v1)

	// Add other route groups here as needed
	// Example: secrets.RegisterSecretRoutes(secretHandler, v1)
	// Example: orgs.RegisterOrgRoutes(orgHandler, v1)
}

func (s *Server) routes() {
	s.engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Kavach backend is healthy",
		})
	})
}

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

func (s *Server) Start() error {
	s.routes()
	addr := fmt.Sprintf(":%s", s.cfg.Port)
	s.log.Infof("starting server on %s", addr)
	return s.engine.Run(addr)
}
