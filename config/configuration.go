package config

import (
	"os"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	CONFIG_FILE = ".miconfig.yaml"
)

// Configuration represents the bento configuration that is used to perform any bento related services, except preparing a new bento.
// When preparing a new bento, a new configuration will be written to the cwd.
type Configuration struct {
	BentoId        string `mapstructure:"bento_id" yaml:"bento_id"`
	PrivateKeyPath string `mapstructure:"private_key_path" yaml:"private_key_path"`
}

// NewConfiguration creates a new Configuration.
func NewConfiguration(bentoId, pkPath string) Configuration {
	return Configuration{
		BentoId:        bentoId,
		PrivateKeyPath: pkPath,
	}
}

// Save the current Configuration to a file.
func (c *Configuration) Save(path string) error {
	out, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(out)
	return err
}

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
