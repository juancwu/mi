package env

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
)

type Env struct {
	SERVICE_URL string
}

var values Env

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading env: %v\n", err)
	}

	values = Env{}

	// required env
	values.SERVICE_URL = getEnv("SERVICE_URL", true)
}

// checks if env exists or not
func getEnv(key string, required bool) string {
	v := os.Getenv(key)
	if v == "" {
		if required {
			log.Fatalf("Missing required env: %s\n", key)
		} else {
			log.Warnf("Missing optional env: %s\n", key)
		}
	}
	return v
}

func Values() *Env {
	return &values
}
