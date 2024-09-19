package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"syscall"

	"github.com/juancwu/mi/config"
	"github.com/juancwu/mi/text"
	"github.com/juancwu/mi/util"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// apiResponse represents a general response body
type apiResponse struct {
	Message   string   `json:"message"`
	RequestId string   `json:"request_id"`
	Errs      []string `json:"errors,omitempty"`
	BentoId   string   `json:"bento_id,omitempty"`
}

// newAuthCmd creates a new auth command and all its subcommands.
func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication related actions.",
	}
	cmd.AddCommand(newSignupCmd())
	cmd.AddCommand(newSigninCmd())
	cmd.AddCommand(newResendVerificationEmailCmd())
	cmd.AddCommand(newVerifyEmailCmd())
	cmd.AddCommand(newResetPasswordCmd())
	cmd.AddCommand(newDeleteAccountCmd())
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

func newVerifyEmailCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify-email <code>",
		Short: "Verifies email with the given code.",
		Long:  "Verifies email with the given code.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code := args[0]
			serviceUrl := config.GetServiceURL()
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/auth/email/verify?code=%s", serviceUrl, url.QueryEscape(code)), nil)
			if err != nil {
				return err
			}
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
			if err := json.Unmarshal(resBodyBytes, &resBody); err != nil {
				return err
			}
			fmt.Printf("Message: %s\nRequest ID: %s\n", resBody.Message, resBody.RequestId)
			if len(resBody.Errs) > 0 {
				util.LogApiResponseErrs(resBody.Errs)
			}
			return nil
		},
	}
	return cmd
}

func newResendVerificationEmailCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resend-verification <email>",
		Short: "Resends verification email.",
		Long:  "Resends verification email.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			email := args[0]
			serviceUrl := config.GetServiceURL()
			body := map[string]string{
				"email": email,
			}
			marshalled, err := json.Marshal(body)
			if err != nil {
				return err
			}
			buf := bytes.NewBuffer(marshalled)
			req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/auth/email/resend", serviceUrl), buf)
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
			if err := json.Unmarshal(resBodyBytes, &resBody); err != nil {
				return err
			}
			fmt.Printf("Message: %s\nRequest ID: %s\n", resBody.Message, resBody.RequestId)
			if len(resBody.Errs) > 0 {
				util.LogApiResponseErrs(resBody.Errs)
			}
			return nil
		},
	}
	return cmd
}

func newResetPasswordCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset-password <email>",
		Short: "Start reset password process.",
		Long:  "Start reset password process. You will need access to the email that is linked to the account.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			email := args[0]
			serviceUrl := config.GetServiceURL()
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/auth/forgot/password?email=%s", serviceUrl, url.QueryEscape(email)), nil)
			if err != nil {
				return err
			}
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

func newDeleteAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete-account",
		Short: "Delete your Konbini account. PERMANENTLY.",
		Long:  "Delete your Konbini account. PERMANENTLY. You will have to be logged in, or have a valid access token to perform this action.",
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Are you sure you want to permanently delete your account? [y/n]: ")
			confirmation, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			if confirmation != "y\n" {
				return nil
			}
			fmt.Print("Please confirm again, are you really sure? [y/n]: ")
			confirmation, err = reader.ReadString('\n')
			if err != nil {
				return err
			}
			if confirmation != "y\n" {
				return nil
			}
			fmt.Println(text.Foreground(text.YELLOW, "WARNING: PROCEEDING TO DELETE ACCOUNT"))
			creds, err := config.LoadCredentials()
			if err != nil {
				return err
			}
			serviceUrl := config.GetServiceURL()
			if err := getNewAccessToken(creds); err != nil {
				return err
			}
			req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/auth/account", serviceUrl), nil)
			if err != nil {
				return err
			}
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
			if res.StatusCode == http.StatusOK {
				if err := creds.Remove(); err != nil {
					fmt.Printf("Failed to remove old saved credentials: %s\n", creds.LocalFilePath)
					fmt.Printf(text.Foreground(text.RED, "ERROR: %v\n"), err)
					fmt.Println("The credentials are no longer valid since the account was deleted.")
				}
			}
			return nil
		},
	}
	return cmd
}
