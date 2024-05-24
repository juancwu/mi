package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/go-playground/validator"
	"github.com/juancwu/konbini-cli/shared/env"
	"github.com/juancwu/konbini-cli/shared/form"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan [email] [password]",
	Long:  "Scan credentials to authenticate user to unblock bento services.",
	Short: "Scan credentials to authenticate user.",
	Args:  cobra.ExactArgs(2),
	RunE:  scanRun,
}

type scanResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func scanRun(cmd *cobra.Command, args []string) error {
	form := form.MembershipScanForm{
		Email:    args[0],
		Password: args[1],
	}

	validate := validator.New()
	if err := validate.Struct(form); err != nil {
		log.Errorf("One or more fields are invalid: %v\n", err)
		return err
	}

	payloadBytes, err := json.Marshal(form)
	if err != nil {
		log.Errorf("Failed to marshal request body: %v\n", err)
		return err
	}

	body := bytes.NewBuffer(payloadBytes)
	resp, err := http.Post(fmt.Sprintf("%s/auth", env.Values().SERVICE_URL), "application/json", body)
	if err != nil {
		log.Errorf("Failed to make http request to Konbini: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var respData scanResponse
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&respData)
		if err != nil {
			log.Errorf("Failed to read response body: %v\n", err)
			return err
		}

		log.Infof("Response Body: %v\n", respData)
	} else {
		log.Errorf("Failed to scan credentials.")
	}

	return nil
}
