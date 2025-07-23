package db

import (
	"database/sql"
	"fmt"

	"github.com/Gkemhcs/kavach-backend/internal/config"
	_ "github.com/lib/pq"

	"github.com/sirupsen/logrus"
)

// InitDB initializes the PostgreSQL database connection using the provided logger and config.
// Returns a *sql.DB instance for database operations. Ensures the connection is valid before returning.
func InitDB(logger *logrus.Logger, config *config.Config) *sql.DB {
	// Build the PostgreSQL connection URL from config values
	dbURL := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		config.DBUser, config.DBPassword, config.DBHost, config.DBPort, config.DBName)

	// Open a new database connection
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		logger.Fatal("Cannot open DB: ", err)
	}

	// Ping the database to ensure the connection is valid
	if err := conn.Ping(); err != nil {
		logger.Fatal("Cannot ping DB: ", err)
	}

	// Return the database connection for use by repositories and services
	return conn
}
