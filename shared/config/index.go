package config

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
)

const (
	CONFIG_DIR  = "konbini"
	CONFIG_NAME = "config.yaml"
)

type AuthConfig struct {
	AccessToken  string `mapstructure:"access_token"`
	RefreshToken string `mapstructure:"refresh_token"`
}

type AppConfig struct {
	Auth AuthConfig `mapstructure:"auth"`
}

func GetAppConfig() (*AppConfig, error) {
	viper.SetConfigName(CONFIG_NAME)
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Errorf("Failed to get user configuration directory: %v\n", err)
		return nil, err
	}
	viper.AddConfigPath(filepath.Join(configDir, CONFIG_DIR))

	err = viper.ReadInConfig()
	if err != nil {
		log.Errorf("Failed to read configuration file: %v\n", err)
		return nil, err
	}

	var appConfig AppConfig
	err = viper.Unmarshal(&appConfig)
	if err != nil {
		log.Errorf("Failed to unmarshal configuration file: %v\n", err)
		return nil, err
	}

	return &appConfig, nil
}

func CreateAppConfig() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Errorf("Failed to get user configuration directory: %v\n", err)
		return err
	}

	appConfigDir := filepath.Join(configDir, CONFIG_DIR)
	_, err = os.Stat(appConfigDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(appConfigDir, 0755)
		if err != nil {
			log.Errorf("Failed to create application configuration directory: %v\n", err)
			return err
		}
		log.Debug("Application configuration directory created.")
	}

	configFilePath := filepath.Join(appConfigDir, CONFIG_NAME)
	_, err = os.Stat(configFilePath)
	if os.IsNotExist(err) {
		configContent := `auth:
    access_token: ""
    refresh_token: ""`
		err = os.WriteFile(configFilePath, []byte(configContent), 0644)
		if err != nil {
			log.Errorf("Failed to create default config file: %v\n", err)
			return err
		}
		log.Debug("Application configuration file created.")
	}

	return nil
}
