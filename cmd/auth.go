package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"syscall"

	"github.com/juancwu/konbini-cli/config"
	"github.com/juancwu/konbini-cli/text"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// apiResponse represents a general response body
type apiResponse struct {
	Message   string   `json:"message"`
	RequestId string   `json:"request_id"`
	Errs      []string `json:"errors,omitempty"`
}

// newAuthCmd creates a new auth command and all its subcommands.
func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication related actions.",
	}
	cmd.AddCommand(newSignupCmd())
	cmd.AddCommand(newSigninCmd())
	return cmd
}

// newSignupCmd creates a new signup command and all its subcommands.
func newSignupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signup",
		Short: "Signup for an account to use the Konbini API.",
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter your email: ")
			email, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			// remove the inclusive delimeter
			email = email[:len(email)-1]
			fmt.Print("Enter your password:")
			bytePassword, err := term.ReadPassword(syscall.Stdin)
			if err != nil {
				return err
			}
			fmt.Print("\n")
			password := string(bytePassword)
			fmt.Print("Enter your name: ")
			name, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			// remove the inclusive delimeter
			name = name[:len(name)-1]
			// make the request
			body := map[string]string{
				"email":    email,
				"password": password,
				"name":     name,
			}
			marshalled, err := json.Marshal(body)
			if err != nil {
				return err
			}
			buf := bytes.NewBuffer(marshalled)
			req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/auth/signup", config.GetServiceURL()), buf)
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("Content-Length", strconv.Itoa(buf.Len()))

			client := http.Client{}
			res, err := client.Do(req)
			if err != nil {
				return err
			}
			defer res.Body.Close()

			resBodyBytes, err := io.ReadAll(res.Body)
			if err != nil {
				return err
			}

			var resBody apiResponse
			err = json.Unmarshal(resBodyBytes, &resBody)
			if err != nil {
				return err
			}

			fmt.Printf("Message: %s\nRequest ID: %s\n", resBody.Message, resBody.RequestId)
			if len(resBody.Errs) > 0 {
				for _, e := range resBody.Errs {
					fmt.Printf("%s %s\n", text.Foreground(text.RED, "Error:"), e)
				}
			}

			return nil
		},
	}
	return cmd
}

// newSigninCmd creates a new command to signin.
// When signin is successful, the access and refresh tokens will be saved in the user's config path "$HOME/.config/mi".
// A warning will be logged when it is done.
func newSigninCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signin",
		Short: "Signin to a Konbini account.",
		RunE: func(cmd *cobra.Command, args []string) error {
			email, err := readEmail()
			if err != nil {
				return err
			}
			password, err := readPassword()
			if err != nil {
				return err
			}
			body := map[string]string{
				"email":    email,
				"password": password,
			}
			b, err := json.Marshal(body)
			if err != nil {
				return err
			}
			buf := bytes.NewBuffer(b)
			req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/auth/signin", config.GetServiceURL()), buf)
			if err != nil {
				return err
			}
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("Content-Length", strconv.Itoa(len(b)))
			client := http.Client{}
			res, err := client.Do(req)
			if err != nil {
				return err
			}
			defer res.Body.Close()
			if res.StatusCode == http.StatusOK {
				var c config.Credentials
				b, err = io.ReadAll(res.Body)
				if err != nil {
					return err
				}
				err = json.Unmarshal(b, &c)
				if err != nil {
					return err
				}
				c.Email = email
				err = config.SaveCredentials(&c)
				if err != nil {
					return err
				}
				fmt.Printf("%s credentials were saved in $HOME/.config/%s/%s. If you do not wish them to be there save them somewhere else.\n", text.Foreground(text.YELLOW, "WARN:"), config.CONFIG_DIR_NAME, config.CREDS_FILE)
				fmt.Println(text.Foreground(text.GREEN, fmt.Sprintf("Successfully signed in as: %s", email)))
			} else {
				var resBody apiResponse
				b, err = io.ReadAll(res.Body)
				if err != nil {
					return err
				}
				err = json.Unmarshal(b, &resBody)
				if err != nil {
					return err
				}
				fmt.Printf("Message: %s\n", resBody.Message)
				fmt.Printf("Request ID: %s\n", resBody.RequestId)
				if len(resBody.Errs) > 0 {
					for _, e := range resBody.Errs {
						fmt.Printf("%s %s\n", text.Foreground(text.RED, "Error:"), e)
					}
				}
			}
			return nil
		},
	}
	return cmd
}

// getNewAccessToken makes a request to get a new access token with a stored refresh token.
func getNewAccessToken(c *config.Credentials) error {
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
	return nil
}
