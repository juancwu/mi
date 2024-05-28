package utils

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"os"
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

func LoadKeys(BentoConfig *BentoConfig) (*Keys, error) {
	pubPemBytes, err := os.ReadFile(BentoConfig.PublicKeyPath)
	if err != nil {
		return nil, err
	}
	pubBlock, _ := pem.Decode(pubPemBytes)
	if pubBlock == nil || pubBlock.Type != "RSA PUBLIC KEY" {
		return nil, errors.New("Invalid RSA public key")
	}
	midPubKey, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		return nil, err
	}
	pubKey, ok := midPubKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("Failed to parse public key")
	}

	prvPemBytes, err := os.ReadFile(BentoConfig.PrivateKeyPath)
	if err != nil {
		return nil, err
	}
	prvBlock, _ := pem.Decode(prvPemBytes)
	if prvBlock == nil || prvBlock.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("Invalid RSA private key")
	}
	prvKey, err := x509.ParsePKCS1PrivateKey(prvBlock.Bytes)
	if err != nil {
		return nil, err
	}

	return &Keys{
		Private: prvKey,
		Public:  pubKey,
	}, nil
}

func Encrypt(data []byte, keys *Keys) ([]byte, error) {
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, keys.Public, data)
	if err != nil {
		return nil, err
	}
	return encrypted, nil
}

func Decrypt(data string, keys *Keys) ([]byte, error) {
	// decode base64
	decodedBytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	plainContext, err := rsa.DecryptPKCS1v15(rand.Reader, keys.Private, decodedBytes)
	if err != nil {
		return nil, err
	}
	return plainContext, nil
}

func GetSignature(keys *Keys) ([]byte, []byte, error) {
	challenge, err := GetChallenge(32)
	if err != nil {
		return nil, nil, err
	}

	hash := sha256.New()
	_, err = hash.Write(challenge)
	if err != nil {
		return nil, nil, err
	}
	hashed := hash.Sum(nil)

	signature, err := rsa.SignPKCS1v15(rand.Reader, keys.Private, crypto.SHA256, hashed)
	if err != nil {
		return nil, nil, err
	}

	return hashed, signature, nil
}

func GetChallenge(length int) ([]byte, error) {
	challenge := make([]byte, length)
	_, err := rand.Read(challenge)
	if err != nil {
		return nil, err
	}
	return challenge, nil
}
