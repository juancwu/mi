package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func executeRootCmd() error {
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

	return rootCmd.ExecuteContext(context.Background())
}
