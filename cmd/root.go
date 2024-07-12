package cmd

import "github.com/spf13/cobra"

// newRootCmd creates a root command and all its subcommands.
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mi",
		Short: "Mi is a secret management cli for projects.",
		Long:  "Mi is a secret management cli for projects that connects to the Konbini API.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return nil
		},
	}

	cmd.AddCommand(newAuthCmd())

	return cmd
}

// Execute starts the cli.
func Execute() error {
	cmd := newRootCmd()
	if err := cmd.Execute(); err != nil {
		return err
	}
	return nil
}
