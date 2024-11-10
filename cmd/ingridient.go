package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"syscall"

	"github.com/juancwu/mi/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newIngridientCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ing",
		Short: "Bento ingridient related commands. Rename, change value, and delete.",
	}
	cmd.AddCommand(newRenameCmd())
	cmd.AddCommand(newReseasonCmd())
	return cmd
}

func newRenameCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rename <old-name> <new-name>",
		Short: "Renames an existing ingridient from an existing bento.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			serviceUrl := config.GetServiceURL()
			oldName := args[0]
			newName := args[1]

			bentoCfg, err := config.LoadConfiguration()
			if err != nil {
				return err
			}

			creds, err := config.LoadCredentials()
			if err != nil {
				return err
			}
			// renew creds
			if err := getNewAccessToken(creds); err != nil {
				return err
			}

			// read PEM encoded private key
			block, err := readPEMKey(bentoCfg.PrivateKeyPath)
			if err != nil {
				return err
			}

			// parse private key
			pk, err := parsePrivateKey(block)
			if err != nil {
				return err
			}

			challenge, err := createChallenge()
			if err != nil {
				return err
			}

			hashed := sha256.Sum256(challenge)
			signature, err := signChallenge(pk, hashed)
			if err != nil {
				return err
			}

			requestBody := map[string]any{
				"bento_id": bentoCfg.BentoId,
				"challenger": map[string]string{
					"challenge": encodeChallenge(challenge),
					"signature": encodeSignature(signature),
				},
				"old_name": oldName,
				"new_name": newName,
			}
			reqBodyBytes, err := json.Marshal(requestBody)
			if err != nil {
				return err
			}
			reqBodyBuf := bytes.NewBuffer(reqBodyBytes)
			req, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/bento/ingridient/rename", serviceUrl), reqBodyBuf)
			if err != nil {
				return err
			}
			req.Header.Add(header_content_type, header_mime_json)
			req.Header.Add(header_content_length, strconv.Itoa(len(reqBodyBytes)))
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
	return cmd
}

func newReseasonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reseason <name>",
		Short: "Changes the value of an ingridient.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			creds, err := config.LoadCredentials()
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("Please sign-in using 'mi auth signin'.")
					return nil
				}
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

			// parse private key
			pk, err := parsePrivateKeyFromPath(cfg.PrivateKeyPath)
			if err != nil {
				return err
			}

			if err := getNewAccessToken(creds); err != nil {
				return err
			}

			challenger, err := newChallenger(pk)
			if err != nil {
				return err
			}

			// read in secret value
			fmt.Print("Enter secret seasoning: ")
			secret, err := term.ReadPassword(syscall.Stdin)
			if err != nil {
				return err
			}
			fmt.Print("\n")

			// encrypt secret
			encryptedSecret, err := encryptValue(&pk.PublicKey, secret)
			if err != nil {
				return err
			}

			body := reseasonIngridientRequest{
				BentoId:    cfg.BentoId,
				Challenger: *challenger,
				Name:       name,
				Value:      hex.EncodeToString(encryptedSecret),
			}
			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return err
			}
			readBuffer := bytes.NewBuffer(bodyBytes)
			serviceUrl := config.GetServiceURL()
			req, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/bento/ingridient/reseason", serviceUrl), readBuffer)
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

	return cmd
}
