package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	"github.com/go-playground/validator"
	"github.com/juancwu/konbini-cli/shared/env"
	"github.com/juancwu/konbini-cli/shared/form"
	"golang.org/x/term"
)

// returns the access token
func Auth(email, password string) (string, error) {
	form := form.MembershipScanForm{
		Email:    email,
		Password: password,
	}

	validate := validator.New()
	if err := validate.Struct(form); err != nil {
		log.Errorf("One or more fields are invalid: %v\n", err)
		return "", err
	}

	payloadBytes, err := json.Marshal(form)
	if err != nil {
		log.Errorf("Failed to marshal request body: %v\n", err)
		return "", err
	}

	body := bytes.NewBuffer(payloadBytes)
	resp, err := http.Post(fmt.Sprintf("%s/auth", env.Values().SERVICE_URL), "application/json", body)
	if err != nil {
		log.Errorf("Failed to make http request to Konbini: %v\n", err)
		return "", err
	}
	defer resp.Body.Close()

	token := ""
	if resp.StatusCode == http.StatusOK {
		respBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Failed to read response body: %v\n", err)
			return "", err
		}
		token = string(respBytes)
	} else {
		log.Errorf("Failed authentication.")
		return "", errors.New("Failed authentication.")
	}

	return token, nil
}

func PromptCredentials() (*form.MembershipScanForm, error) {
	credsFrom := new(form.MembershipScanForm)
	fmt.Print("Enter email: ")
	fmt.Scanln(&credsFrom.Email)

	fmt.Print("Enter password: ")
	pwdBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return nil, err
	}
	credsFrom.Password = string(pwdBytes)

	validate := validator.New()
	err = validate.Struct(credsFrom)
	if err != nil {
		return nil, err
	}

	return credsFrom, nil
}
