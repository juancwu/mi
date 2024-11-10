package cmd

import (
	"bufio"
	"bytes"
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
	"strings"

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
	cmd.AddCommand(newAllowEditCmd())
	cmd.AddCommand(newRevokeEditCmd())

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
				fmt.Println("here")
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

func newAllowEditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "allow-edit <share-to-email> [--permissions=]",
		Short: "Allow edit of an existing bento to another user.",
		Long:  "Allow edit of an existing bento to another user. Permissions is a comma separated string of options: all,write,delete,share,rename_bento,rename_ingridient,write_ingridient,delete_ingridient,revoke_share. One can only grant up to their own permission level.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serviceUrl := config.GetServiceURL()
			email := args[0]
			p, err := cmd.Flags().GetString("permissions")
			if err != nil {
				fmt.Println("Failed to get 'permissions' flag")
				return err
			}
			permissions := strings.Split(p, ",")

			cfg, err := config.LoadConfiguration()
			if err != nil {
				return err
			}

			creds, err := config.LoadCredentials()
			if err != nil {
				return err
			}
			if err := getNewAccessToken(creds); err != nil {
				return err
			}

			// read in private key
			block, err := readPEMKey(cfg.PrivateKeyPath)
			if err != nil {
				return err
			}

			// parse PEM private key
			pk, err := parsePrivateKey(block)
			if err != nil {
				return err
			}

			// create challenge
			challengeBytes, err := createChallenge()

			// sign challenge
			signatureBytes, err := signChallenge(pk, sha256.Sum256(challengeBytes))
			if err != nil {
				return err
			}

			body := map[string]any{
				"bento_id":          cfg.BentoId,
				"share_to_email":    email,
				"challenge":         encodeChallenge(challengeBytes),
				"signature":         encodeSignature(signatureBytes),
				"permission_levels": permissions,
			}
			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return err
			}
			bodyReader := bytes.NewBuffer(bodyBytes)

			req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/bento/edit/allow", serviceUrl), bodyReader)
			if err != nil {
				return err
			}
			req.Header.Add(header_content_type, header_mime_json)
			req.Header.Add(header_content_length, strconv.Itoa(len(bodyBytes)))
			req.Header.Add(header_authorization, fmt.Sprintf("Bearer %s", creds.AccessToken))

			client := http.Client{}
			res, err := client.Do(req)
			if err != nil {
				return err
			}
			defer res.Body.Close()
			resBody, err := readApiResponseBody(res.Body)
			if err != nil {
				return err
			}
			logApiResponseBody(resBody)

			return nil
		},
	}

	cmd.Flags().StringP("permissions", "p", "", "Optional: desired permission levels to give the receiving user.")

	return cmd
}

func newRevokeEditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke-edit <email> [--permissions]",
		Short: "Removes a user from getting edit access to a bento.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			email := args[0]
			perms, err := cmd.Flags().GetString("permissions")
			if err != nil {
				return err
			}
			creds, err := config.LoadCredentials()
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("Please sign-in using 'mi auth signin' first.")
					return nil
				}
				return err
			}
			if err := getNewAccessToken(creds); err != nil {
				return err
			}
			cfg, err := config.LoadConfiguration()
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("Could not find '.miconfig.yaml' in the current working directory.")
					return nil
				}
				return err
			}
			// load up private key
			block, err := readPEMKey(cfg.PrivateKeyPath)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("Could not find private key in the provided config file.")
					return nil
				}
				return err
			}

			// parse private key
			pk, err := parsePrivateKey(block)
			if err != nil {
				return err
			}

			challengeBytes, err := createChallenge()
			if err != nil {
				return err
			}

			signatureBytes, err := signChallenge(pk, sha256.Sum256(challengeBytes))
			if err != nil {
				return err
			}

			revokePermissions := strings.Split(perms, ",")
			body := revokeEditRequest{
				BentoId:                cfg.BentoId,
				Email:                  email,
				Challenge:              encodeChallenge(challengeBytes),
				Signature:              encodeSignature(signatureBytes),
				ToBeRevokedPermissions: revokePermissions,
			}

			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return err
			}
			readBuffer := bytes.NewBuffer(bodyBytes)
			serviceUrl := config.GetServiceURL()
			req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/bento/edit/revoke", serviceUrl), readBuffer)
			if err != nil {
				return err
			}
			req.Header.Add(header_content_type, header_mime_json)
			req.Header.Add(header_content_length, strconv.Itoa(len(bodyBytes)))
			req.Header.Add(header_authorization, fmt.Sprintf("Bearer %s", creds.AccessToken))
			client := http.Client{}
			res, err := client.Do(req)
			if err != nil {
				return err
			}
			defer res.Body.Close()
			resBody, err := readApiResponseBody(res.Body)
			if err != nil {
				return err
			}
			logApiResponseBody(resBody)

			return nil
		},
	}

	cmd.Flags().StringP("permissions", "p", "", "Comma separated permissions to revoke. Refer to https://github.com/juancwu/konbini for more details.")

	return cmd
}
