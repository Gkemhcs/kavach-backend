package config

import (
	"github.com/spf13/viper"
)

// Config holds all configuration values for the application, loaded from environment variables or config files.
// This struct centralizes configuration for maintainability and testability.
type Config struct {
	Port                 string // HTTP server port
	Env                  string // Application environment (e.g., development, production)
	GitHubClientID       string // GitHub OAuth client ID
	GitHubClientSecret   string // GitHub OAuth client secret
	GitHubRedirectURL    string // GitHub OAuth redirect URL
	DBUser               string // Database user
	DBPort               string // Database port
	DBHost               string // Database host
	DBName               string // Database name
	DBPassword           string // Database password
	JWTSecret            string // Secret key for signing JWTs
	JWTDuration          int    // JWT token duration in minutes (deprecated, use AccessTokenDuration)
	AccessTokenDuration  int    // Access token duration in minutes
	RefreshTokenDuration int    // Refresh token duration in minutes
}

// Load reads configuration from the .env file and environment variables, returning a Config struct.
// This function enables flexible configuration for different environments (dev, prod, test).
func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	viper.SetDefault("DB_USER", "kavach_user")
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PASSWORD", "gkem1234")
	viper.SetDefault("DB_NAME", "kavach")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("ACCESS_TOKEN_DURATION", 10)    // 10 minutes
	viper.SetDefault("REFRESH_TOKEN_DURATION", 1440) // 1 day in minutes
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	return &Config{
		Port:                 viper.GetString("PORT"),
		Env:                  viper.GetString("ENV"),
		GitHubClientID:       viper.GetString("GITHUB_CLIENT_ID"),
		GitHubClientSecret:   viper.GetString("GITHUB_CLIENT_SECRET"),
		GitHubRedirectURL:    viper.GetString("GITHUB_REDIRECT_URL"),
		DBUser:               viper.GetString("DB_USER"),
		DBPort:               viper.GetString("DB_PORT"),
		DBHost:               viper.GetString("DB_HOST"),
		DBName:               viper.GetString("DB_NAME"),
		DBPassword:           viper.GetString("DB_PASSWORD"),
		JWTSecret:            viper.GetString("JWT_SECRET"),
		JWTDuration:          viper.GetInt("JWT_DURATION"),
		AccessTokenDuration:  viper.GetInt("ACCESS_TOKEN_DURATION"),
		RefreshTokenDuration: viper.GetInt("REFRESH_TOKEN_DURATION"),
	}, nil
}
