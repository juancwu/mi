package utils

import "fmt"

// store bento utils

const BENTO_CONFIG_TEMPLATE = `name: %s
bento_id: %s
public_key_path: %s
private_key_path: %s
`

func CreateBentoConfig(name, bentoId, publicKeyPath, privateKeyPath string) string {
	return fmt.Sprintf(BENTO_CONFIG_TEMPLATE, name, bentoId, publicKeyPath, privateKeyPath)
}
