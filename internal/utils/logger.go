package utils 


import (
	"github.com/Gkemhcs/kavach-backend/internal/config"
	"github.com/sirupsen/logrus"
	"os"
)

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
