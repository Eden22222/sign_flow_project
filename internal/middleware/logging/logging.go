package logging

import (
	"os"

	log "github.com/sirupsen/logrus"
)

// Setup configures global logrus behavior for local development and production.
func Setup() {
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})
}
