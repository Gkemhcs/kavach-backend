package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds all configuration values for the application, loaded from environment variables or config files.
// This struct centralizes configuration for maintainability and testability.
type Config struct {
	Port                  string // HTTP server port
	Env                   string // Application environment (e.g., development, production)
	GitHubClientID        string // GitHub OAuth client ID
	GitHubClientSecret    string // GitHub OAuth client secret
	GitHubRedirectURL     string // GitHub OAuth redirect URL
	DBUser                string // Database user
	DBPort                string // Database port
	DBHost                string // Database host
	DBName                string // Database name
	DBPassword            string // Database password
	JWTSecret             string // Secret key for signing JWTs
	JWTDuration           int    // JWT token duration in minutes (deprecated, use AccessTokenDuration)
	AccessTokenDuration   int    // Access token duration in minutes
	RefreshTokenDuration  int    // Refresh token duration in minutes
	ModelFilePath         string
	SecretEncryptionKey   string
	ProviderEncryptionKey string
}

// Load reads configuration from the .env file and environment variables, returning a Config struct.
// This function enables flexible configuration for different environments (dev, prod, test).
func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	// Set comprehensive defaults for all configuration values
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("ENV", "development")
	viper.SetDefault("DB_USER", "kavach_user")
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PASSWORD", "gkem1234")
	viper.SetDefault("DB_NAME", "kavach_db")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("JWT_SECRET", "default-jwt-secret-change-in-production")
	viper.SetDefault("ACCESS_TOKEN_DURATION", 1000)  // 10 minutes
	viper.SetDefault("REFRESH_TOKEN_DURATION", 1440) // 1 day in minutes
	viper.SetDefault("MODEL_FILE_PATH", "internal/authz/model.conf")
	viper.SetDefault("ENCRYPTION_KEY", "RhK7KoKSwOuFOHxONMNaO9Z9pDgJKwZjaNhcbgZ7Qqc=")
	viper.SetDefault("GITHUB_REDIRECT_URL", "http://localhost:8080/api/v1/auth/github/callback")

	// Try to read .env file, but don't fail if it doesn't exist
	if err := viper.ReadInConfig(); err != nil {
		// Log the error but continue with defaults and environment variables
		// This allows the app to run without a .env file
	}

	config := &Config{
		Port:                  viper.GetString("PORT"),
		Env:                   viper.GetString("ENV"),
		GitHubClientID:        viper.GetString("GITHUB_CLIENT_ID"),
		GitHubClientSecret:    viper.GetString("GITHUB_CLIENT_SECRET"),
		GitHubRedirectURL:     viper.GetString("GITHUB_REDIRECT_URL"),
		DBUser:                viper.GetString("DB_USER"),
		DBPort:                viper.GetString("DB_PORT"),
		DBHost:                viper.GetString("DB_HOST"),
		DBName:                viper.GetString("DB_NAME"),
		DBPassword:            viper.GetString("DB_PASSWORD"),
		JWTSecret:             viper.GetString("JWT_SECRET"),
		JWTDuration:           viper.GetInt("JWT_DURATION"),
		AccessTokenDuration:   viper.GetInt("ACCESS_TOKEN_DURATION"),
		RefreshTokenDuration:  viper.GetInt("REFRESH_TOKEN_DURATION"),
		ModelFilePath:         viper.GetString("MODEL_FILE_PATH"),
		SecretEncryptionKey:   viper.GetString("ENCRYPTION_KEY"),
		ProviderEncryptionKey: viper.GetString("ENCRYPTION_KEY"),
	}

	// Validate configuration for production environment
	if config.Env == "production" {
		if err := validateProductionConfig(config); err != nil {
			return nil, err
		}
	}

	return config, nil
}

// validateProductionConfig ensures all required configuration is set for production
func validateProductionConfig(config *Config) error {
	if config.JWTSecret == "default-jwt-secret-change-in-production" {
		return fmt.Errorf("JWT_SECRET must be set to a secure value in production")
	}
	if config.GitHubClientID == "" {
		return fmt.Errorf("GITHUB_CLIENT_ID must be set in production")
	}
	if config.GitHubClientSecret == "" {
		return fmt.Errorf("GITHUB_CLIENT_SECRET must be set in production")
	}
	return nil
}
