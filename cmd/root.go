package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "deltaflow",
	Short: "Smart CI/CD change detection",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "./deltaflow.yml", "Path to deltaflow.yml config file")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable detailed output")
}
