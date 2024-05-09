package cmd

import "github.com/spf13/cobra"

var getCmd = &cobra.Command{
	Use:   "get [command]",
	Long:  "Provides a series of commands to get things from Konbini.",
	Short: "Get things from Konbini",
}

func init() {
	getCmd.AddCommand(membershipCmd)
}
