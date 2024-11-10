package cmd

import "github.com/spf13/cobra"

// newRootCmd creates a root command and all its subcommands.
func newRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "mi",
		Short:   "Mi is a secret management cli for projects.",
		Long:    "Mi is a secret management cli for projects that connects to the Konbini API.",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return nil
		},
	}

	cmd.AddCommand(newAuthCmd())
	cmd.AddCommand(newBentoCmd())
	cmd.AddCommand(newUpdateCmd())

	return cmd
}

// Execute starts the cli.
func Execute(version string) error {
	cmd := newRootCmd(version)
	if err := cmd.Execute(); err != nil {
		return err
	}
	return nil
}
