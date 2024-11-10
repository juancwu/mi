package cmd

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/juancwu/mi/config"
	"github.com/juancwu/mi/text"
	"github.com/juancwu/mi/util"
)

// getNewAccessToken makes a request (only if access token is expired and refresh token is still valid) to get a new access token with a stored refresh token.
func getNewAccessToken(c *config.Credentials) error {
	expired, err := is_jwt_expired(c.AccessToken)
	if err != nil {
		return err
	}
	if expired {
		expired, err = is_jwt_expired(c.RefreshToken)
		if err != nil {
			return err
		}
		if expired {
			if err := c.Remove(); err != nil {
				fmt.Printf("Failed to remove expired credentials: %v\n", err)
			}
			return errors.New("Access and refresh token expired. Please sign-in again using `mi auth signin`.")
		}
		// refresh token is still valid, get new access token
		req, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/auth/refresh", config.GetServiceURL()), nil)
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.RefreshToken))
		client := http.Client{}
		res, err := client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		switch res.StatusCode {
		case http.StatusOK:
			var body map[string]string
			b, err := io.ReadAll(res.Body)
			if err != nil {
				return err
			}
			err = json.Unmarshal(b, &body)
			if err != nil {
				return err
			}
			at, ok := body["access_token"]
			if !ok {
				return errors.New("No access token found in response body.")
			}
			c.AccessToken = at
			err = config.SaveCredentials(c)
			if err != nil {
				return err
			}
		case http.StatusUnauthorized:
			return newErrExpiredCreds()
		default:
			return errors.New("Failed to get new access token.")
		}
	}
	return nil
}

// readApiResponseBody tries to read a response body of type JSON and returns the apiResponse struct.
func readApiResponseBody(body io.ReadCloser) (*apiResponse, error) {
	b, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	var resBody apiResponse
	if err := json.Unmarshal(b, &resBody); err != nil {
		return nil, err
	}
	return &resBody, nil
}

// logApiResponseBody handles formatted logging of an api response body.
func logApiResponseBody(resBody *apiResponse) {
	if resBody.Message != "" {
		fmt.Printf("Message: %s\n", resBody.Message)
	}
	if resBody.RequestId != "" {
		fmt.Printf("Request ID: %s\n", resBody.RequestId)
	}
	if len(resBody.Errs) > 0 {
		util.LogApiResponseErrs(resBody.Errs)
	}
}

// makePreprareBentoRequest makes a request to prepare a new bento.
// This makes it easier to re do a request if it fails due to expired credentials.
func makePreprareBentoRequest(credentials *config.Credentials, contentLength int, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/bento/prepare", config.GetServiceURL()), body)
	if err != nil {
		return nil, err
	}
	req.Header.Add(header_content_type, header_mime_json)
	req.Header.Add(header_content_length, strconv.Itoa(contentLength))
	req.Header.Add(header_authorization, fmt.Sprintf("Bearer %s", credentials.AccessToken))
	client := http.Client{}
	return client.Do(req)
}

// read_json_respone_body will read a response body and will close the body afterwards.
func read_json_respone_body(res *http.Response, i interface{}) error {
	defer res.Body.Close()
	out, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(out, i)
}

// write_ingridients_to_env will write the given ingridients into the current working directory.
//
// IMPORTANT: any existing .env file will be overwritten.
func write_ingridients_to_env(pk *rsa.PrivateKey, ingridients []ingridient, flags int) error {
	f, err := os.OpenFile(".env", flags, 0600)
	if err != nil {
		return err
	}
	var (
		decoded   []byte
		decrypted []byte
		nameBytes []byte
	)
	for _, ingridient := range ingridients {
		decoded, err = hex.DecodeString(ingridient.Value)
		if err != nil {
			fmt.Println(text.Foreground(text.RED, fmt.Sprintf("[ERROR]: Failed to decode the value of ingridient '%s'.", ingridient.Name)))
			continue
		}
		decrypted, err = decryptValue(pk, decoded)
		if err != nil {
			fmt.Println(text.Foreground(text.RED, fmt.Sprintf("[ERROR]: Failed to decrypt the value of ingridient '%s'.", ingridient.Name)))
			continue
		}
		// converting name to bytes because it is usually shorter instead of value to bytes
		nameBytes = []byte(ingridient.Name)
		f.Write(nameBytes)
		f.WriteString("=")
		f.Write(decrypted)
		f.WriteString("\n")
	}
	return nil
}

// file_exists checks if file exists or not. If there is an error (apart from not exists error), it will return falsy and the error.
func file_exists(path string) (bool, error) {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return !stat.IsDir(), nil
}

// is_jwt_expired verifies if the given token is still valid or not.
func is_jwt_expired(tokenStr string) (bool, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, &jwt.RegisteredClaims{})
	if err != nil {
		return true, err
	}
	expirationTime, err := token.Claims.GetExpirationTime()
	if err != nil {
		return true, err
	}
	return time.Now().After(expirationTime.Time), nil
}

// createChallenge generates a challenge that can later be used for signing.
// Most bento related actions required a signed challenge to gain access.
func createChallenge() ([]byte, error) {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}
	return randomBytes, nil
}

// encodes a challenge bytes into hexadecimal
func encodeChallenge(b []byte) string {
	return hex.EncodeToString(b)
}

// encodes a signature bytes into hexadecimal
func encodeSignature(b []byte) string {
	return hex.EncodeToString(b)
}

// signChallenge signs the given challenge with the private key.
func signChallenge(pk *rsa.PrivateKey, challenge [32]byte) ([]byte, error) {
	return rsa.SignPKCS1v15(nil, pk, crypto.SHA256, challenge[:])
}

// parsePrivateKey parses a private key in PEM format.
func parsePrivateKey(block *pem.Block) (*rsa.PrivateKey, error) {
	if block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("Unsupported private key. Expected type: 'PRIVATE KEY', but found: '%s'", block.Type)
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pk, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("Failed to parse private key.")
	}
	return pk, nil
}

func parsePrivateKeyFromPath(keyPath string) (*rsa.PrivateKey, error) {
	block, err := readPEMKey(keyPath)
	if err != nil {
		return nil, err
	}

	if block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("Unsupported private key. Expected type: 'PRIVATE KEY', but found: '%s'", block.Type)
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pk, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("Failed to parse private key.")
	}
	return pk, nil
}

// parsePublicKey parses a public key in PEM format.
func parsePublicKey(block *pem.Block) (*rsa.PublicKey, error) {
	if block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("Unsupported public key. Expected type: 'PUBLIC KEY', but found: '%s'", block.Type)
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pk, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("Failed to parse public key.")
	}
	return pk, nil
}

// readPEMKey will try to read the pem encoded key in the given path and return the block.
func readPEMKey(path string) (*pem.Block, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	fb, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(fb)
	return block, nil
}

// encryptValue encrypts data using the PKCS1v15 schema.
func encryptValue(key *rsa.PublicKey, value []byte) ([]byte, error) {
	n := len(value)
	// encrypt pkcs1v15 msg cannot be longer than public modulus - 11 bytes due to padding
	step := key.Size() - 11
	var encrypted []byte

	for start := 0; start < n; start += step {
		finish := start + step
		if finish > n {
			finish = n
		}
		data, err := rsa.EncryptPKCS1v15(rand.Reader, key, value[start:finish])
		if err != nil {
			return nil, err
		}
		encrypted = append(encrypted, data...)
	}
	return encrypted, nil
}

// decryptValue decrypts encrypted data using the PKCS1v15 schema.
func decryptValue(key *rsa.PrivateKey, value []byte) ([]byte, error) {
	n := len(value)
	step := key.PublicKey.Size()
	var decrypted []byte
	for start := 0; start < n; start += step {
		finish := start + step
		if finish > n {
			finish = n
		}
		data, err := rsa.DecryptPKCS1v15(nil, key, value[start:finish])
		if err != nil {
			return nil, err
		}
		decrypted = append(decrypted, data...)
	}
	return decrypted, nil
}

// ingridient represents a key-value pair environment variable entry.
type ingridient struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// readEnvFile will read the ingridient file from the given path and construct a list of ingridient to return.
func readEnvFile(path string) ([]ingridient, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	values, err := godotenv.Parse(f)
	if err != nil {
		return nil, err
	}
	var envs []ingridient
	for key, value := range values {
		envs = append(envs, ingridient{
			Name:  key,
			Value: value,
		})
	}
	return envs, nil
}

// Creates a new challenger with hex encoded challenge and signature
func newChallenger(pk *rsa.PrivateKey) (*challengerType, error) {
	challengeBytes, err := createChallenge()
	if err != nil {
		return nil, err
	}
	signatureBytes, err := signChallenge(pk, sha256.Sum256(challengeBytes))
	if err != nil {
		return nil, err
	}
	return &challengerType{
		Challenge: encodeChallenge(challengeBytes),
		Signature: encodeSignature(signatureBytes),
	}, nil
}
