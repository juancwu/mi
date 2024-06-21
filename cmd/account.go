package cmd

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/juancwu/konbini-cli/models"
	"github.com/spf13/cobra"
)

// getSignupCmd will initialize the cobra command for signing up
// which then can be added to a higher up command or executed directly.
func getSignupCmd() *cobra.Command {
	signupCmd := &cobra.Command{
		Use: "signup",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := tea.NewProgram(models.InitSignupModel())
			_, err := p.Run()
			return err
		},
	}

	return signupCmd
}

// getLoginCmd  will initialize the cobra command for logging in
// which then can be added to a higher up command or executed directly.
func getLoginCmd() (*cobra.Command, error) {
	loginCmd := &cobra.Command{
		Use: "login",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return loginCmd, nil
}

// getEmailVerificationCmd will initialize the cobra command for getting a new email verification.
// The same email can only get a new email verification every 60 seconds.
func getEmailVerificationCmd() *cobra.Command {
	emailVerificationCmd := &cobra.Command{
		Use: "send-verification-email",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return emailVerificationCmd
}

// getLogoutCmd will initialize the cobra command for logging out
// which then can be added to a higher up command or executed directly.
func getLogoutCmd() (*cobra.Command, error) {
	logoutCmd := &cobra.Command{
		Use: "logout",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return logoutCmd, nil
}

// getVerifyEmailCmd will initialize the cobra command for verifying the email of an account
// which then can be added to a higher up command or executed directly.
func getVerifyEmailCmd() (*cobra.Command, error) {
	verifyEmailCmd := &cobra.Command{
		Use:  "verify-email <verification-code>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return verifyEmailCmd, nil
}

// getResetPwdCmd will initialize the cobra command for resetting the password of an account
// which then can be added to a higher up command or executed directly.
func getResetPwdCmd() (*cobra.Command, error) {
	resetPwdCmd := &cobra.Command{
		Use: "reset-password",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return resetPwdCmd, nil
}

// getUpdateAccountCmd will initialize the cobra command for updating an account
// which then can be added to a higher up command or executed directly.
func getUpdateAccountCmd() (*cobra.Command, error) {
	updateAccountCmd := &cobra.Command{
		Use: "update-account",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return updateAccountCmd, nil
}

// getViewAccountCmd will initialize the cobra command for viewing an account
// which then can be added to a higher up command or executed directly.
func getViewAccountCmd() (*cobra.Command, error) {
	viewAccountCmd := &cobra.Command{
		Use: "update-account",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return viewAccountCmd, nil
}

// getDeleteAccountCmd will initialize the cobra command for deleting an account
// which then can be added to a higher up command or executed directly.
func getDeleteAccountCmd() (*cobra.Command, error) {
	deleteAccountCmd := &cobra.Command{
		Use: "update-account",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return deleteAccountCmd, nil
}
