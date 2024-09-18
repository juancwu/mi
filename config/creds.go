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
	credsDir, err := getCredentialsPath()
	if err != nil {
		return nil, err
	}
	credsFile := filepath.Join(credsDir, CONFIG_DIR_NAME, CREDS_FILE)
	f, err := os.Open(credsFile)
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
// If APP_ENV is "dev", then it will be in the "$CWD/tmp/.config/mi".
func SaveCredentials(c *Credentials) error {
	credsDir, err := getCredentialsPath()
	if err != nil {
		return err
	}
	credsFile := filepath.Join(credsDir, CONFIG_DIR_NAME, CREDS_FILE)
	_, err = os.Stat(credsFile)
	if err != nil {
		if os.IsNotExist(err) {
			// create dir
			// needs executive perms to work with dir in Unix
			err = os.MkdirAll(filepath.Join(credsDir, CONFIG_DIR_NAME), 0700)
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
	f, err := os.OpenFile(credsFile, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(b)
	return err
}

// getCredentialsPath returns the path where credentials should be stored based on the
// APP_ENV environment variable. If value is "dev" then it will create a new directory
// named "tmp" in the CWD, otherwise it defaults to the user's configuration directory.
func getCredentialsPath() (string, error) {
	if os.Getenv("APP_ENV") == "dev" {
		return "./tmp", nil
	}
	cfgDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return cfgDir, nil
}
