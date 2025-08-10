package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/Gkemhcs/kavach-backend/internal/config"
	_ "github.com/lib/pq"

	"github.com/sirupsen/logrus"
)

// InitDB initializes the PostgreSQL database connection with connection pooling using the provided logger and config.
// Returns a *sql.DB instance for database operations. Ensures the connection is valid before returning.
func InitDB(logger *logrus.Logger, config *config.Config) *sql.DB {
	// Build the PostgreSQL connection URL from config values
	// URL-encode the password to handle special characters
	encodedPassword := url.QueryEscape(config.DBPassword)
	dbURL := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		config.DBUser, encodedPassword, config.DBHost, config.DBPort, config.DBName)

	// Open a new database connection
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		logger.Fatal("Cannot open DB: ", err)
	}

	// Configure connection pooling settings
	configureConnectionPool(conn, config, logger)

	// Ping the database to ensure the connection is valid
	if err := conn.Ping(); err != nil {
		logger.Fatal("Cannot ping DB: ", err)
	}

	// Log connection pool configuration
	logger.WithFields(logrus.Fields{
		"max_open_conns":     config.DBMaxOpenConns,
		"max_idle_conns":     config.DBMaxIdleConns,
		"conn_max_lifetime":  fmt.Sprintf("%dm", config.DBConnMaxLifetime),
		"conn_max_idle_time": fmt.Sprintf("%dm", config.DBConnMaxIdleTime),
	}).Info("Database connection pool configured")

	// Return the database connection for use by repositories and services
	return conn
}

// configureConnectionPool sets up the connection pool with optimal settings for the environment
func configureConnectionPool(db *sql.DB, config *config.Config, logger *logrus.Logger) {
	// Set maximum number of open connections to the database
	db.SetMaxOpenConns(config.DBMaxOpenConns)

	// Set maximum number of idle connections in the pool
	db.SetMaxIdleConns(config.DBMaxIdleConns)

	// Set maximum amount of time a connection may be reused
	// This helps prevent issues with stale connections
	db.SetConnMaxLifetime(time.Duration(config.DBConnMaxLifetime) * time.Minute)

	// Set maximum amount of time an idle connection may be reused
	// This helps prevent issues with idle connections being closed by the database
	db.SetConnMaxIdleTime(time.Duration(config.DBConnMaxIdleTime) * time.Minute)

	// Log the configuration for debugging
	logger.WithFields(logrus.Fields{
		"environment":        config.Env,
		"max_open_conns":     config.DBMaxOpenConns,
		"max_idle_conns":     config.DBMaxIdleConns,
		"conn_max_lifetime":  fmt.Sprintf("%dm", config.DBConnMaxLifetime),
		"conn_max_idle_time": fmt.Sprintf("%dm", config.DBConnMaxIdleTime),
	}).Debug("Database connection pool settings applied")
}

// GetConnectionStats returns current connection pool statistics for monitoring
func GetConnectionStats(db *sql.DB) map[string]interface{} {
	stats := db.Stats()
	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration.String(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}
}

// ValidateConnectionPool validates that the connection pool is working correctly
func ValidateConnectionPool(db *sql.DB, logger *logrus.Logger) error {
	// Test the connection pool by executing a simple query
	_, err := db.Exec("SELECT 1")
	if err != nil {
		return fmt.Errorf("connection pool validation failed: %w", err)
	}

	// Log connection pool statistics
	stats := GetConnectionStats(db)
	logger.WithFields(logrus.Fields{
		"stats": stats,
	}).Info("Connection pool validation successful")

	return nil
}
