package utils

import (
	"os"

	"github.com/Gkemhcs/kavach-backend/internal/config"
	"github.com/sirupsen/logrus"
)

// New creates and configures a new logrus.Logger instance based on the environment.
// Sets output to stdout and uses JSON formatting. Adjusts log level for development vs production.
func New(cfg *config.Config) *logrus.Logger {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.JSONFormatter{})

	if cfg.Env == "development" {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}

	return log
}
