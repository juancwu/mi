package utils

import "os"

func GetServiceURL() string {
	if os.Getenv("APP_ENV") == "development" {
		return "http://127.0.0.1:3000"
	}
	return "https://konbini.juancwu.dev"
}
