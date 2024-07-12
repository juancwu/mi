package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
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

// newAuthCmd creates a new auth command and all its subcommands.
func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication related actions.",
	}
	cmd.AddCommand(newSignupCmd())
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

			type apiResponse struct {
				Message   string   `json:"message"`
				RequestId string   `json:"request_id"`
				Errs      []string `json:"errors,omitempty"`
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
