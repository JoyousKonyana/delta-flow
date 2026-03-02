package cmd

import (
	"fmt"

	"github.com/JoyousKonyana/deltaflow/internal/config"
	"github.com/JoyousKonyana/deltaflow/internal/git"
	"github.com/JoyousKonyana/deltaflow/internal/matcher"
	"github.com/JoyousKonyana/deltaflow/internal/output"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	baseBranch string
	headCommit string
	format     string
	dryRun     bool
)

var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect changed services between two commits",
	RunE:  runDetect,
}

func runDetect(cmd *cobra.Command, args []string) error {
	if !git.IsGitRepo() {
		return fmt.Errorf("not a git repository — run deltaflow from within your project root")
	}

	if git.IsShallowClone() {
		color.Yellow("⚠  Warning: shallow clone detected — diff results may be incomplete")
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Printf("Config loaded: %d service(s) defined\n", len(cfg.Services))
	}

	changedFiles, err := git.GetChangedFiles(baseBranch, headCommit)
	if err != nil {
		return fmt.Errorf("failed to get changed files: %w", err)
	}

	if verbose {
		fmt.Printf("Changed files (%d):\n", len(changedFiles))
		for _, f := range changedFiles {
			fmt.Printf("  %s\n", f)
		}
		fmt.Println()
	}

	result, err := matcher.Match(changedFiles, cfg)
	if err != nil {
		return fmt.Errorf("matching failed: %w", err)
	}

	// Pass full service list so GHA output can emit per-service booleans
	return output.Print(result, cfg.ServiceNames(), format, dryRun)
}

func init() {
	detectCmd.Flags().StringVar(&baseBranch, "base", "main", "Base branch to compare against")
	detectCmd.Flags().StringVar(&headCommit, "head", "HEAD", "Head commit to compare")
	detectCmd.Flags().StringVar(&format, "format", "table", "Output format: table|json|gha|gha-matrix|env")
	detectCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print detected changes without triggering pipelines")

	rootCmd.AddCommand(detectCmd)
}
