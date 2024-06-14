package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Execute initializes all commands and will run the cli.
// Any additional commands should be added here.
func Execute() error {
	rootCmd := &cobra.Command{
		Version: os.Getenv("VERSION"),
		Use:     "konbini",
		Long:    "Konbini is a CLI that helps you manage your secrets for your awesome projects.",
		Short:   "Manage your project's secrets with ease.",
		Example: "konbini",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			fmt.Println("Root cmd")
			return nil
		},
	}

	rootCmd.AddCommand(getSignupCmd())

	return rootCmd.ExecuteContext(context.Background())
}
