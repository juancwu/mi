package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

type Keys struct {
	Private *rsa.PrivateKey
	Public  *rsa.PublicKey
}

func GenerateKey() (*Keys, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	keys := &Keys{
		Private: privateKey,
		Public:  &privateKey.PublicKey,
	}
	return keys, nil
}

// returns private, public, error
func Keys2PEM(keys *Keys) ([]byte, []byte, error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(keys.Public)
	if err != nil {
		return nil, nil, err
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(keys.Private)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	return privateKeyPEM, publicKeyPEM, nil
}

func Encrypt(data []byte, keys *Keys) ([]byte, error) {
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, keys.Public, data)
	if err != nil {
		return nil, err
	}
	return encrypted, nil
}
