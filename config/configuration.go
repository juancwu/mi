package config

import "github.com/spf13/viper"

// Configuration represents the bento configuration that is used to perform any bento related services, except preparing a new bento.
// When preparing a new bento, a new configuration will be written to the cwd.
type Configuration struct {
	BentoId        string `mapstructure:"bento_id"`
	PrivateKeyPath string `mapstructure:"private_key_path"`
}

const (
	CONFIG_FILE = ".miconfig.yaml"
)

// LoadConfiguration loads the app configuration in the cwd.
func LoadConfiguration() (*Configuration, error) {
	viper.SetConfigFile(CONFIG_FILE)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	var config Configuration
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	return &config, nil
}
