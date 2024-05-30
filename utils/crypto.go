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

	"github.com/charmbracelet/log"
)

type Keys struct {
	Private *rsa.PrivateKey
	Public  *rsa.PublicKey
}

const (
	LINEFEED byte = 10
	EQUAL_CH byte = 61
)

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

func splitKeyValLine(line []byte) ([]byte, []byte, error) {
	for i := 0; i < len(line); i++ {
		if line[i] == EQUAL_CH {
			return line[0:i], line[i+1:], nil
		}
	}

	return nil, nil, errors.New("Invalid key-value entry")
}

func encryptValue(data []byte, keys *Keys) ([]byte, error) {
	label := []byte("")
	hash := sha256.New()
	size := len(data)
	step := keys.Public.Size() - 2*hash.Size() - 2
	var encrypted []byte
	for start := 0; start < size; start += step {
		finish := start + step
		if finish > size {
			finish = size
		}

		block, err := rsa.EncryptOAEP(hash, rand.Reader, keys.Public, data[start:finish], label)
		if err != nil {
			return nil, err
		}

		encrypted = append(encrypted, block...)
	}
	return encrypted, nil
}

// returns keys, values, error
func Encrypt(data []byte, keys *Keys) ([]string, [][]byte, error) {
	k := []string{}
	v := [][]byte{}

	// split data using linefeed and grab the env value
	s := 0
	for i := 0; i < len(data); i++ {
		if data[i] == LINEFEED {
			line := data[s:i]
			key, val, err := splitKeyValLine(line)
			if err != nil {
				return nil, nil, err
			}
			k = append(k, string(key))
			encrypted, err := encryptValue(val, keys)
			if err != nil {
				return nil, nil, err
			}
			v = append(v, encrypted)
			s = i + 1
			log.Info("encrypted", "key", string(key))
		}
	}

	return k, v, nil
}

// data is a base64 encoded data
func decryptValue(data string, keys *Keys) ([]byte, error) {
	// decode base64
	decodedBytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	label := []byte("")
	hash := sha256.New()
	size := len(decodedBytes)
	step := keys.Private.Size()
	var decrypted []byte
	for start := 0; start < size; start += step {
		finish := start + step
		if finish > size {
			finish = size
		}

		block, err := rsa.DecryptOAEP(hash, rand.Reader, keys.Private, decodedBytes[start:finish], label)
		if err != nil {
			return nil, err
		}

		decrypted = append(decrypted, block...)
	}

	return decrypted, nil
}

// returns keys, values, error
func Decrypt(data []string, keys *Keys) ([]string, [][]byte, error) {
	k := []string{}
	v := [][]byte{}

	for i := 0; i < len(data)-1; i += 2 {
		decrypted, err := decryptValue(data[i+1], keys)
		if err != nil {
			return nil, nil, err
		}
		k = append(k, data[i])
		v = append(v, decrypted)
	}

	return k, v, nil
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
