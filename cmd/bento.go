package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/go-playground/validator"
	"github.com/juancwu/konbini-cli/shared/env"
	"github.com/juancwu/konbini-cli/shared/form"
	"github.com/juancwu/konbini-cli/utils"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var createBentoCmd = &cobra.Command{
	Use:   "bento [name]",
	Short: "Commands to manage personal bentos.",
	Args:  cobra.ExactArgs(1),
	Run:   createBentoRun,
}

func createBentoRun(cmd *cobra.Command, args []string) {
	var email, password string
	// get email
	fmt.Print("Enter email: ")
	fmt.Scanln(&email)

	// get password
	fmt.Print("Enter password: ")
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		log.Errorf("Failed to read password: %v\n", err)
		return
	}
	password = string(passwordBytes)

	// validate inputs
	validate := validator.New()
	if err := validate.Struct(form.MembershipScanForm{Email: email, Password: password}); err != nil {
		log.Errorf("One or more fields are invalid/missing: %v\n", err)
		return
	}

	// check if env file is present in the current directory
	envData, err := utils.GetEnvContent()
	if err != nil {
		log.Errorf("Failed to read .env file in current directory: %v\n", err)
		return
	}

	if len(envData) == 0 {
		log.Error("Env file is empty.")
		return
	}

	// authenticate user
	log.Info("Authenticating...")
	accessToken, err := utils.Auth(email, password)
	if err != nil {
		log.Errorf("Failed to authenticate user: %v\n", err)
		return
	}

	// get new keys
	log.Info("Generating keys...")
	keys, err := utils.GenerateKey()
	if err != nil {
		log.Errorf("Failed to generate private key: %v\n", err)
		return
	}

	// encrypt env envData
	log.Info("Encrypting env data...")
	encrypted, err := utils.Encrypt(envData, keys)
	if err != nil {
		log.Errorf("Failed to encrypt env data: %v\n", err)
		return
	}

	log.Info("Encoding keys to PEM...")
	privatePEM, publicPEM, err := utils.Keys2PEM(keys)
	if err != nil {
		log.Errorf("Failed to encode keys to PEM: %v\n", err)
		return
	}

	log.Info("Saving public key in disk as 'public.pem'")
	err = os.WriteFile("public.pem", publicPEM, 0644)
	if err != nil {
		log.Errorf("Failed to save public key to disk: %v\n", err)
		return
	}

	log.Info("Saving private key in disk as 'private.pem'")
	err = os.WriteFile("private.pem", privatePEM, 0644)
	if err != nil {
		log.Errorf("Failed to save private key to disk: %v\n", err)
		return
	}

	log.Info("Gathering bento ingridients...")
	bentoForm := form.BentoForm{
		Name:      args[0],
		PublicKey: string(publicPEM),
		Content:   string(encrypted),
	}
	reqBodyBytes, err := json.Marshal(bentoForm)
	if err != nil {
		log.Errorf("Failed to gather bento ingridients: %v\n", err)
		return
	}

	// make bento request
	log.Info("Preheating pans...")
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/bento/personal/new", env.Values().SERVICE_URL), bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		log.Errorf("Failed to preheat pans: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(reqBodyBytes)))

	// make client to do the request
	log.Info("Cooking bento...")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Something went wrong when cooking bento: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		log.Info("Bento cooked and safely stored in Konbini.")
		bentoIdBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Error reading response body: %v\n", err)
			return
		}
		log.Info("Saving bento recipe...")
		cwd, err := os.Getwd()
		if err != nil {
			log.Warn("Failed to get current working directory, using fallback method.")
			cwd = ""
		}
		bentoConfig := utils.CreateBentoConfig(bentoForm.Name, string(bentoIdBytes), filepath.Join(cwd, "public.pem"), filepath.Join(cwd, "private.pem"))
		err = os.WriteFile(filepath.Join(cwd, DEFAULT_CFG_FILE_PATH), []byte(bentoConfig), 0644)
		if err != nil {
			log.Errorf("Failed to save bento recipe in disk: %v\n", err)
			return
		}
	} else {
		// show error message
		errMsgBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Error reading response body: %v\n", err)
			return
		}
		log.Errorf("Failed to cook bento (code: %d): %s\n", resp.StatusCode, string(errMsgBytes))
	}
}
