package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const starterConfig = `version: 1

# Global triggers — changes to these files mark ALL services as affected.
# Useful for shared CI config, root-level dependencies, or infrastructure.
global_triggers:
  - .github/workflows/**
  - go.mod
  - go.sum

# Define your services and the paths that belong to each one.
# Supports doublestar glob patterns (e.g. services/api/**)
services:
  api:
    paths:
      - services/api/**
      - packages/shared/**

  worker:
    paths:
      - services/worker/**
      - packages/shared/**

  frontend:
    paths:
      - services/frontend/**
      - packages/ui/**
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a starter deltaflow.yml in the current directory",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	target := cfgFile // respects --config flag, defaults to ./deltaflow.yml

	if _, err := os.Stat(target); err == nil {
		color.Yellow("⚠  %s already exists. Delete it first or use --config to specify a different path.", target)
		return nil
	}

	if err := os.WriteFile(target, []byte(starterConfig), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", target, err)
	}

	color.Green("✔  Created %s", target)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit deltaflow.yml to match your repo structure")
	fmt.Println("  2. Run:  deltaflow detect --base=main")
	fmt.Println("  3. Run:  deltaflow workflow init          (per-service jobs)")
	fmt.Println("           deltaflow workflow init --matrix (dynamic matrix)")
	fmt.Println("  4. Commit both files to your repository")
	fmt.Println()

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
}
