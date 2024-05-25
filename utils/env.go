package utils

import "os"

func GetEnvContent() ([]byte, error) {
	data, err := os.ReadFile(".env")
	if err != nil {
		return nil, err
	}
	return data, nil
}
