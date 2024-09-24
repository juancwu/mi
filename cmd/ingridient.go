package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/juancwu/mi/config"
	"github.com/spf13/cobra"
)

func newIngridientCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingridient",
		Short: "Bento ingridient related commands. Rename, change value, and delete.",
	}
	cmd.AddCommand(newRenameCmd())
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
