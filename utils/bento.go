package utils

import (
	"fmt"

	"github.com/spf13/viper"
)

// store bento utils

const BENTO_CONFIG_TEMPLATE = `name: %s
bento_id: %s
public_key_path: %s
private_key_path: %s
`

type BentoConfig struct {
	Name           string `mapstructure:"name"`
	BentoId        string `mapstructure:"bento_id"`
	PublicKeyPath  string `mapstructure:"public_key_path"`
	PrivateKeyPath string `mapstructure:"private_key_path"`
}

func CreateBentoConfig(name, bentoId, publicKeyPath, privateKeyPath string) string {
	return fmt.Sprintf(BENTO_CONFIG_TEMPLATE, name, bentoId, publicKeyPath, privateKeyPath)
}

func GetBentoConfig(configPath string) (*BentoConfig, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	err := v.ReadInConfig()
	if err != nil {
		return nil, err
	}

	var bentoConfig BentoConfig
	err = v.Unmarshal(&bentoConfig)
	if err != nil {
		return nil, err
	}

	return &bentoConfig, nil
}
