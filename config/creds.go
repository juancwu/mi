package config

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

const (
	CONFIG_DIR_NAME = "mi"
	CREDS_FILE      = "creds.json"
)

// Credentials represent the json that is saved in the config folder for tokens.
type Credentials struct {
	Email        string `json:"email,omitempty"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// LoadCredentials loads up the Credentials file in the config folder if exists, otherwise it returns an error.
func LoadCredentials() (*Credentials, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	configFile := filepath.Join(configDir, CONFIG_DIR_NAME, CREDS_FILE)
	f, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var c Credentials
	err = json.Unmarshal(b, &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// SaveCredentials saves the given token Credentials in the config folder "$HOME/.config/mi"
func SaveCredentials(c *Credentials) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	configFile := filepath.Join(configDir, CONFIG_DIR_NAME, CREDS_FILE)
	_, err = os.Stat(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// create dir
			// needs executive perms to work with dir in Unix
			err = os.MkdirAll(filepath.Join(configDir, CONFIG_DIR_NAME), 0700)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	// only let the owner read/write the Credentials file
	f, err := os.OpenFile(configFile, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(b)
	return err
}
