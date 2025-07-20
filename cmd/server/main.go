package main

import (
	"time"

	"github.com/Gkemhcs/kavach-backend/internal/auth"
	userdb "github.com/Gkemhcs/kavach-backend/internal/auth/gen"
	"github.com/Gkemhcs/kavach-backend/internal/auth/jwt"
	"github.com/Gkemhcs/kavach-backend/internal/auth/provider"
	"github.com/Gkemhcs/kavach-backend/internal/config"
	"github.com/Gkemhcs/kavach-backend/internal/db"
	"github.com/Gkemhcs/kavach-backend/internal/server"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	logger := utils.New(cfg)

	// Initialize auth service and handler
	githubProvider := provider.NewGitHubProvider(
		cfg.GitHubClientID,
		cfg.GitHubClientSecret,
		cfg.GitHubRedirectURL,
	)
	authRepository := db.InitDB(logger, cfg)
	// JWT manager setup
	jwter := jwt.NewManager(cfg.JWTSecret, time.Duration(cfg.JWTDuration)*time.Minute)
	authService := auth.NewAuthService(githubProvider, userdb.New(authRepository), jwter)
	authHandler := auth.NewAuthHandler(authService, logger)

	s := server.New(cfg, logger)

	// Initialize auth routes
	s.SetupRoutes(authHandler)

	if err := s.Start(); err != nil {
		logger.Fatal("server failed to start", err)
	}
}
