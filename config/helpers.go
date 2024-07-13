package config

import "os"

// GetServiceURL returns the right service url based on the running environment.
// For percise control, change the env APP_ENV.
func GetServiceURL() string {
	if os.Getenv("APP_ENV") == "dev" {
		return "http://127.0.0.1:3000"
	}
	return ""
}
