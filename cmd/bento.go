package cmd

import (
	"bufio"
	"bytes"
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
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/juancwu/mi/config"
	"github.com/juancwu/mi/text"
	"github.com/spf13/cobra"
)

// newBentoCmd creates a new bento cmd and all of its subcommands.
func newBentoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bento",
		Short: "Bento related commands. Get, update, delete, all in here.",
	}
	cmd.AddCommand(newOrderBentoCmd())
	cmd.AddCommand(newPrepareBentoCmd())
	cmd.AddCommand(newFillBentoCmd())
	cmd.AddCommand(newThrowBentoCmd())
	return cmd
}

// newPrepareBentoCmd creates a new command to prepare a new bento.
func newPrepareBentoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prepare <bento-name>",
		Short: "Prepare a new bento.",
		Long:  "Prepare a new bento with the given name. Optionally include a path to an .env file to fill the bento with using --env. If no path to a RSA private key PEM encoded was provided to the --key flag, one will be generate.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bentoName := args[0]
			envPath, err := cmd.Flags().GetString("env")
			if err != nil {
				return err
			}
			keyPath, err := cmd.Flags().GetString("key")
			if err != nil {
				return err
			}

			// load up credentials
			credentials, err := config.LoadCredentials()
			if err != nil {
				if os.IsNotExist(err) {
					// need to sign in first
					fmt.Println("Please sign in before preparing a new bento. Use: 'mi auth signin' or 'mi auth signup' to create a new account.")
					return nil
				}
				return err
			}

			var pk *rsa.PrivateKey
			if keyPath == "" {
				// generate new key
				pk, err = rsa.GenerateKey(rand.Reader, 2048)
				if err != nil {
					return err
				}
				pkBytes, err := x509.MarshalPKCS8PrivateKey(pk)
				if err != nil {
					return err
				}
				// save to file now
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				keyPath = filepath.Join(cwd, "private.pem")
				pkFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE, 0600)
				if err != nil {
					return err
				}
				defer pkFile.Close()
				block := pem.Block{
					Type:  "PRIVATE KEY",
					Bytes: pkBytes,
				}
				err = pem.Encode(pkFile, &block)
				if err != nil {
					return err
				}
			} else {
				block, err := readPEMKey(keyPath)
				if err != nil {
					return err
				}
				pk, err = parsePrivateKey(block)
				if err != nil {
					return err
				}
			}

			pubBytes, err := x509.MarshalPKIXPublicKey(&pk.PublicKey)
			if err != nil {
				return err
			}

			block := pem.Block{
				Type:  "PUBLIC KEY",
				Bytes: pubBytes,
			}
			pemBytes := pem.EncodeToMemory(&block)
			if pemBytes == nil {
				return errors.New("Error encoding PEM block for public key.")
			}

			var ingridients []ingridient
			if envPath != "" {
				ingridients, err = readEnvFile(envPath)
				if err != nil {
					return err
				}
				for i := 0; i < len(ingridients); i++ {
					encrypted, err := encryptValue(&pk.PublicKey, []byte(ingridients[i].Value))
					if err != nil {
						return err
					}
					// encode to hex to make it easier to send and store
					ingridients[i].Value = hex.EncodeToString(encrypted)
				}
			}

			body := map[string]any{
				"name":    bentoName,
				"pub_key": string(pemBytes),
			}
			if len(ingridients) > 0 {
				body["ingridients"] = ingridients
			}

			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return err
			}
			bodyBuffer := bytes.NewBuffer(bodyBytes)

			res, err := makePreprareBentoRequest(credentials, len(bodyBytes), bodyBuffer)
			if err != nil {
				return err
			}

			if res.StatusCode == http.StatusUnauthorized {
				// request for new access token
				err = getNewAccessToken(credentials)
				if err != nil {
					return err
				}
				bodyBuffer.Reset()
				_, err := bodyBuffer.Write(bodyBytes)
				if err != nil {
					return err
				}
				res, err = makePreprareBentoRequest(credentials, len(bodyBytes), bodyBuffer)
				if err != nil {
					return err
				}
			}

			var resBody apiResponse
			err = read_json_respone_body(res, &resBody)
			if err != nil {
				return err
			}

			fmt.Printf("Message: %s\nRequest ID: %s\n", resBody.Message, resBody.RequestId)
			if len(resBody.Errs) > 0 {
				for _, e := range resBody.Errs {
					fmt.Printf("%s %s\n", text.Foreground(text.RED, "Error:"), e)
				}
			}

			if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusCreated {
				configuration := config.NewConfiguration(resBody.BentoId, keyPath)
				configuration.Save(config.CONFIG_FILE)
			}

			return nil
		},
	}
	cmd.Flags().String("env", "", "Include an .env file to fill the newly prepared bento with.")
	cmd.Flags().StringP("key", "k", "", "Provide the path to the RSA private key PEM encoded that you wish to use for the bento.")
	return cmd
}

// newFillBentoCmd creates a new command to add more "ingridients" to a prepared bento.
func newFillBentoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fill <path-to-env-file>",
		Args:  cobra.ExactArgs(1),
		Short: "Fills a prepared bento with the given ingridient file.",
		Long:  "Fills a prepared bento with the key value pairs from the given ingridient file. The command will use the bento id in the .miconfig.yaml file by default. Use -b for custom bento id.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfiguration()
			if err != nil {
				return err
			}

			credentials, err := config.LoadCredentials()
			if err != nil {
				if os.IsNotExist(err) {
					// need to sign in first
					fmt.Println("Please sign in before filling a bento. Use: 'mi auth signin' or 'mi auth signup' to create a new account.")
					return nil
				}
				return err
			}

			bentoId, err := cmd.Flags().GetString("bento")
			if err != nil {
				return err
			}
			if bentoId == "" {
				bentoId = cfg.BentoId
			}

			keyPath, err := cmd.Flags().GetString("key")
			if err != nil {
				return err
			}
			if keyPath == "" {
				if cfg.PrivateKeyPath == "" {
					return errors.New("No private key path found in .miconfig.yaml or given through --key.")
				}
				keyPath = cfg.PrivateKeyPath
			}

			block, err := readPEMKey(keyPath)
			if err != nil {
				return err
			}

			pk, err := parsePrivateKey(block)
			if err != nil {
				return err
			}

			ingridients, err := readEnvFile(args[0])
			if err != nil {
				return err
			}

			for i := 0; i < len(ingridients); i++ {
				b, err := encryptValue(&pk.PublicKey, []byte(ingridients[i].Value))
				if err != nil {
					return err
				}
				ingridients[i].Value = hex.EncodeToString(b)
			}

			challenge, err := createChallenge()
			if err != nil {
				return err
			}
			signature, err := signChallenge(pk, sha256.Sum256(challenge))
			if err != nil {
				return err
			}

			reqBody := map[string]any{
				"bento_id":    cfg.BentoId,
				"ingridients": ingridients,
				"challenge":   hex.EncodeToString(challenge),
				"signature":   hex.EncodeToString(signature),
			}
			reqBodyBytes, err := json.Marshal(reqBody)
			if err != nil {
				return err
			}
			reqBodybuf := bytes.NewBuffer(reqBodyBytes)
			req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/bento/add/ingridients", config.GetServiceURL()), reqBodybuf)
			if err != nil {
				return err
			}
			req.Header.Add(header_content_type, header_mime_json)
			req.Header.Add(header_content_length, strconv.Itoa(len(reqBodyBytes)))
			req.Header.Add(header_authorization, fmt.Sprintf("Bearer %s", credentials.AccessToken))
			client := http.Client{}
			res, err := client.Do(req)
			if err != nil {
				return err
			}
			var resBody apiResponse
			err = read_json_respone_body(res, &resBody)
			if err != nil {
				return err
			}
			if res.StatusCode == http.StatusOK {
				fmt.Println("Bento filled with the provided ingridients.")
			} else {
				fmt.Println(text.Foreground(text.RED, resBody.Message))
				if len(resBody.Errs) > 0 {
					for _, errMsg := range resBody.Errs {
						fmt.Println(text.Foreground(text.RED, fmt.Sprintf("[ERROR]: %s", errMsg)))
					}
				}
				fmt.Println(text.Foreground(text.RED, fmt.Sprintf("Request ID: %s", resBody.RequestId)))
			}

			return nil
		},
	}
	cmd.Flags().StringP("key", "k", "private.pem", "Optional: The path of the PEM encoded private key. Use this flag if you would like to use a different private key aside from the one described in the configuration file.")
	cmd.Flags().StringP("bento", "b", "", "Optional: The bento id that to fill. Use this flag if you would like to use a different bento id aside from the one described in the configuration file.")
	return cmd
}

// newOrderBentoCmd creates a new command to order a prepared bento.
func newOrderBentoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "order",
		Short: "Ordered a bento that was previously prepared.",
		Long:  "Use this command when you need to get the contents of a bento that was previously prepared. You will need the private key that was used for the prepared bento.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// before doing anything, need to confirm overwrite of .env if exists or append
			exists, err := file_exists(".env")
			if err != nil {
				return err
			}

			writeEnvFileFlags := os.O_CREATE | os.O_WRONLY
			if exists {
				reader := bufio.NewReader(os.Stdin)
				fmt.Print("An existing .env file found. Do you want to overwrite (w), append (a) or abort (q)? [w/a/q] ")
				answer, err := reader.ReadString(10)
				if err != nil {
					return err
				}
				answer = answer[:len(answer)-1]
				switch answer {
				case "w":
					writeEnvFileFlags |= os.O_TRUNC
				case "a":
					writeEnvFileFlags |= os.O_APPEND
				default:
					fmt.Println("Exit")
					return nil
				}
			}
			cfg, err := config.LoadConfiguration()
			if err != nil {
				return err
			}
			// create challenge
			challenge, err := createChallenge()
			if err != nil {
				return err
			}
			// hash the challenge with sha256
			hashed := sha256.Sum256(challenge)
			// read in the private key
			block, err := readPEMKey(cfg.PrivateKeyPath)
			if err != nil {
				return err
			}
			// parse the private key so that the challenge can be signed
			pk, err := parsePrivateKey(block)
			if err != nil {
				return err
			}
			// sign the challenge
			signature, err := signChallenge(pk, hashed)
			if err != nil {
				return err
			}
			// encode both challenge and signature to send to Konbini API
			encodedSign := hex.EncodeToString(signature)
			encodedChallenge := hex.EncodeToString(challenge)
			// preprare request query params
			values := url.Values{}
			values.Add("signature", encodedSign)
			values.Add("challenge", encodedChallenge)
			q := values.Encode()
			// prepare http request
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/bento/order/%s?%s", config.GetServiceURL(), cfg.BentoId, q), nil)
			if err != nil {
				return err
			}
			// prepare client
			client := http.Client{}
			res, err := client.Do(req)
			if err != nil {
				return err
			}
			defer res.Body.Close()
			switch res.StatusCode {
			case http.StatusOK:
				var resBody orderBentoResponseBody
				resBodyBytes, err := io.ReadAll(res.Body)
				if err != nil {
					return err
				}
				err = json.Unmarshal(resBodyBytes, &resBody)
				if err != nil {
					return err
				}
				if len(resBody.Ingridients) < 1 {
					return errors.New("Status 200 but no bento received.")
				}
				fmt.Println("Bento arrived!")
				fmt.Println("Unpacking bento...")
				err = write_ingridients_to_env(pk, resBody.Ingridients, writeEnvFileFlags)
				if err != nil {
					return err
				}
				fmt.Println("All done!")
			default:
				var resBody apiResponse
				resBodyBytes, err := io.ReadAll(res.Body)
				if err != nil {
					return err
				}
				err = json.Unmarshal(resBodyBytes, &resBody)
				if resBody.Message != "" {
					fmt.Println(text.Foreground(text.RED, resBody.Message))
				}
				if len(resBody.Errs) > 0 {
					for _, errMsg := range resBody.Errs {
						fmt.Println(text.Foreground(text.RED, fmt.Sprintf("[ERROR]: %s", errMsg)))
					}
				}
				fmt.Println(text.Foreground(text.RED, fmt.Sprintf("Request ID: %s", resBody.RequestId)))
			}
			return nil
		},
	}

	cmd.Flags().StringP("key", "k", "private.pem", "Optional: The path of the PEM encoded private key. Use this flag if you would like to use a different private key aside from the one described in the configuration file.")
	cmd.Flags().StringP("bento", "b", "", "Optional: The prepared bento id that to order. Use this flag if you would like to use a different bento id aside from the one described in the configuration file.")

	return cmd
}

func newThrowBentoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "throw",
		Short: "Throws a bento.",
		Long:  "Throws an existing bento. All contents will be deleted and cannot be recovered. You must be signed in with the account that owns the bento.",
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Please confirm you want to delete the bento defined in .miconfig.yaml? [y/n] ")
			confirmation, err := reader.ReadString(ascii_linefeed)
			if err != nil {
				return err
			}
			fmt.Println()
			if confirmation == "y\n" {
				// reconfirm
				fmt.Print("ARE YOU SURE SURE YOU WANT TO PROCEED? [y/n] ")
				confirmation, err = reader.ReadString(ascii_linefeed)
				if err != nil {
					return err
				}
				fmt.Println()
				if confirmation != "y\n" {
					fmt.Println("Exit")
					return nil
				}
			} else {
				fmt.Println("Exit")
				return nil
			}
			cfg, err := config.LoadConfiguration()
			if err != nil {
				return err
			}
			creds, err := config.LoadCredentials()
			if err != nil {
				return err
			}

			expired, err := is_jwt_expired(creds.AccessToken)
			if err != nil {
				return err
			}
			if expired {
				expired, err = is_jwt_expired(creds.RefreshToken)
				if err != nil {
					return err
				}
				if expired {
					return errors.New("Signed out. Please sign in again.")
				}
				err = getNewAccessToken(creds)
				if err != nil {
					return err
				}
			}

			// make request to delete bento
			req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/bento/throw/%s", config.GetServiceURL(), cfg.BentoId), nil)
			if err != nil {
				return err
			}
			req.Header.Add(header_authorization, fmt.Sprintf("Bearer %s", creds.AccessToken))

			client := http.Client{}
			res, err := client.Do(req)
			if err != nil {
				return err
			}

			if res.StatusCode == http.StatusOK {
				fmt.Println("Bento deleted.")
				res.Body.Close()
				if exists, _ := file_exists(".miconfig.yaml"); exists {
					fmt.Print("Do you want to delete '.miconfig.yaml'? [y/n] ")
					confirmation, err = reader.ReadString(ascii_linefeed)
					if err != nil {
						return err
					}
					fmt.Println()
					if confirmation == "y\n" {
						fmt.Println("Deleting '.miconfig.yaml'")
						err = os.Remove(".miconfig.yaml")
						if err != nil {
							fmt.Println(text.Foreground(text.RED, "Failed to remove .miconfig.yaml"))
						}
					}
				}
				if exists, _ := file_exists(".env"); exists {
					fmt.Print("Do you want to delete '.env'? [y/n] ")
					confirmation, err = reader.ReadString(ascii_linefeed)
					if err != nil {
						return err
					}
					fmt.Println()
					if confirmation == "y\n" {
						fmt.Println("Deleting '.env'")
						err = os.Remove(".env")
						if err != nil {
							fmt.Println(text.Foreground(text.RED, "Failed to remove .env"))
						}
					}
				}
				if exists, _ := file_exists(cfg.PrivateKeyPath); exists {
					fmt.Printf("Do you want to delete the private key (path: %s)? [y/n] ", cfg.PrivateKeyPath)
					confirmation, err = reader.ReadString(ascii_linefeed)
					if err != nil {
						return err
					}
					fmt.Println()
					if confirmation == "y\n" {
						fmt.Printf("Deleting '%s'\n", cfg.PrivateKeyPath)
						err = os.Remove(cfg.PrivateKeyPath)
						if err != nil {
							fmt.Println(text.Foreground(text.RED, fmt.Sprintf("Failed to remove '%s'", cfg.PrivateKeyPath)))
						}
					}
				}
			} else {
				var resBody map[string]string
				resBodyBytes, err := io.ReadAll(res.Body)
				if err != nil {
					return err
				}
				err = json.Unmarshal(resBodyBytes, &resBody)
				if err != nil {
					return err
				}
				fmt.Println(text.Foreground(text.RED, fmt.Sprintf("[ERROR]: %s", resBody["message"])))
				res.Body.Close()
			}

			return nil
		},
	}
	return cmd
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
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var envs []ingridient
	var c byte
	eqIdx := -1
	for start, end := 0, 0; end < len(b); end++ {
		c = b[end]
		switch c {
		case ascii_equal:
			if eqIdx == -1 {
				eqIdx = end
			}
		case ascii_linefeed:
			env := ingridient{
				// grab the key of the ingridient
				Name: string(b[start:eqIdx]),
				// grab the string after the first "=" to the end
				Value: string(b[eqIdx+1 : end]),
			}
			// reset the equal idx
			eqIdx = -1
			// move the start point
			start = end + 1
			envs = append(envs, env)
		}
	}
	return envs, nil
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
