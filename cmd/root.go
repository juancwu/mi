package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

const (
	DEFAULT_CFG_FILE_PATH = ".konbini.yaml"
)

var (
	cfgFilePath string
	verbose     bool
	prompt      bool
	debug       bool
)

var rootCmd = &cobra.Command{
	Use:              "konbini",
	Short:            "Konbini is a (convenient store) to store secrets (bentos) for your projects securely but easily accessible.",
	Version:          "0.0.1",
	Run:              rootRun,
	PersistentPreRun: globalFlagSetup,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFilePath, "config", DEFAULT_CFG_FILE_PATH, "config file (default root directory .konbini.yml)")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&prompt, "prompt", false, "open interactive prompt")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logs")
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(scanCmd)
}

func globalFlagSetup(cmd *cobra.Command, args []string) {
	if debug {
		log.SetLevel(log.DebugLevel)
		log.Debug("Debug Enabled")
	}
}

func rootRun(cmd *cobra.Command, args []string) {
	if verbose {
		log.Info("Arguments", "verbose", verbose, "config", cfgFilePath)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute root command: %v\n", err)
	}
}
