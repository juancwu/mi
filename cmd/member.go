package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/go-playground/validator"
	"github.com/spf13/cobra"

	"github.com/juancwu/konbini-cli/shared/env"
	"github.com/juancwu/konbini-cli/shared/form"
)

var membershipCmd = &cobra.Command{
	Use:   "membership",
	Long:  "Become a member of Konbini to gain access to all the awesome services.",
	Short: "Become a member of Konbini.",
	RunE:  membershipRun,
}

var membershipForm *form.MembershipForm

func init() {
	membershipForm = new(form.MembershipForm)
	membershipCmd.PersistentFlags().StringVar(&membershipForm.Email, "email", "", "Email to link to membership")
	membershipCmd.PersistentFlags().StringVar(&membershipForm.Password, "password", "", "Passowrd for membership")
	membershipCmd.PersistentFlags().StringVar(&membershipForm.FirstName, "firstname", "", "Your first name")
	membershipCmd.PersistentFlags().StringVar(&membershipForm.LastName, "lastname", "", "Your last name")
}

func membershipRun(cmd *cobra.Command, args []string) error {
	if !prompt {
		log.Debug("No prompt. Getting values from flags", "cmd", "konbini get membership")
		validate := validator.New()
		if err := validate.Struct(membershipForm); err != nil {
			log.Errorf("One or more fields are invalid/missing: %v\n", err)
			return err
		}

		payloadBytes, err := json.Marshal(membershipForm)
		if err != nil {
			log.Errorf("Failed to marshal request body: %v\n", err)
			return err
		}

		body := bytes.NewBuffer(payloadBytes)
		resp, err := http.Post(fmt.Sprintf("%s/auth/register", env.Values().SERVICE_URL), "application/json", body)
		if err != nil {
			log.Errorf("Failed to make http request to Konbini: %v\n", err)
			return err
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Failed to read response body: %v\n", err)
			return err
		}

		if resp.StatusCode == http.StatusCreated {
			log.Info(string(respBody))
		} else {
			log.Error(string(respBody))
		}
	} else {
		log.Debug("Building prompt...", "cmd", "konbini get membership")
		log.Warn("Prompt not implemented.")
	}

	return nil
}
