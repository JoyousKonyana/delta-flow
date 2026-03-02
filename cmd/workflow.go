package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/JoyousKonyana/deltaflow/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	workflowMatrix bool
	workflowOut    string
)

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Generate GitHub Actions workflow files powered by Deltaflow",
}

var workflowInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a GitHub Actions workflow wired to deltaflow detect",
	RunE:  runWorkflowInit,
}

func runWorkflowInit(cmd *cobra.Command, args []string) error {
	// Load config so we can tailor the workflow to actual services
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return err
	}

	var workflow string
	if workflowMatrix {
		workflow = buildMatrixWorkflow()
	} else {
		workflow = buildStaticWorkflow(cfg)
	}

	// Determine output path
	outPath := workflowOut
	if outPath == "" {
		if err := os.MkdirAll(".github/workflows", 0755); err != nil {
			return fmt.Errorf("failed to create .github/workflows directory: %w", err)
		}
		outPath = ".github/workflows/deltaflow.yml"
	}

	// Don't overwrite without warning
	if _, err := os.Stat(outPath); err == nil {
		color.Yellow("⚠  %s already exists. Delete it first or use --out to specify a different path.", outPath)
		return nil
	}

	if err := os.WriteFile(outPath, []byte(workflow), 0644); err != nil {
		return fmt.Errorf("failed to write workflow to %s: %w", outPath, err)
	}

	color.Green("✔  Workflow written to %s", outPath)
	fmt.Println()

	if workflowMatrix {
		fmt.Println("Using dynamic matrix mode — any service added to deltaflow.yml is")
		fmt.Println("automatically included in the pipeline with no workflow changes needed.")
	} else {
		fmt.Println("Using per-service job mode — each service has an explicit job with")
		fmt.Println("an if: condition driven by deltaflow detect output.")
	}

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Review %s and adjust deploy steps for each service\n", outPath)
	fmt.Println("  2. Commit both deltaflow.yml and the workflow file")
	fmt.Println("  3. Push a branch and open a PR — Deltaflow will gate deployments automatically")
	fmt.Println()

	return nil
}

// buildStaticWorkflow generates a per-service job workflow tailored to the config.
// Each service gets its own job with an if: condition referencing deltaflow output.
func buildStaticWorkflow(cfg *config.Config) string {
	var b strings.Builder

	b.WriteString(`name: Deltaflow CI

on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]

jobs:
  # ── Step 1: detect which services changed ──────────────────────────────────
  detect:
    name: Detect changed services
    runs-on: ubuntu-latest
    outputs:
      affected:       ${{ steps.delta.outputs.affected }}
      has_changes:    ${{ steps.delta.outputs.has_changes }}
      global_trigger: ${{ steps.delta.outputs.global_trigger }}
`)

	// Emit one output declaration per service
	for _, svc := range cfg.ServiceNames() {
		safe := sanitiseID(svc)
		b.WriteString(fmt.Sprintf("      %s: ${{ steps.delta.outputs.%s }}\n", safe, svc))
	}

	b.WriteString(`
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # required — deltaflow needs full git history

      - name: Install Deltaflow
        run: |
          curl -sSL https://github.com/JoyousKonyana/delta-flow/releases/latest/download/deltaflow-linux-amd64 \
            -o /usr/local/bin/deltaflow
          chmod +x /usr/local/bin/deltaflow

      - name: Run Deltaflow
        id: delta
        run: deltaflow detect --base=origin/main --format=gha

  # ── Step 2: deploy only affected services ──────────────────────────────────
`)

	for _, svc := range cfg.ServiceNames() {
		safe := sanitiseID(svc)
		b.WriteString(fmt.Sprintf(`  deploy-%s:
    name: Deploy %s
    needs: detect
    if: needs.detect.outputs.%s == 'true'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Deploy %s
        run: |
          echo "Deploying %s..."
          # TODO: replace with your actual deploy command
          # e.g. kubectl apply -f services/%s/
          #      helm upgrade --install %s ./charts/%s
          #      ./scripts/deploy.sh %s

`, safe, svc, safe, svc, svc, svc, svc, svc, svc))
	}

	return b.String()
}

// buildMatrixWorkflow generates a dynamic matrix workflow that auto-scales
// to any number of services — no workflow changes needed when services are added.
func buildMatrixWorkflow() string {
	return `name: Deltaflow CI (Matrix)

on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]

jobs:
  # ── Step 1: detect which services changed ──────────────────────────────────
  detect:
    name: Detect changed services
    runs-on: ubuntu-latest
    outputs:
      matrix:      ${{ steps.delta.outputs.matrix }}
      has_changes: ${{ steps.delta.outputs.has_changes }}

    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # required — deltaflow needs full git history

      - name: Install Deltaflow
        run: |
          curl -sSL https://github.com/JoyousKonyana/delta-flow/releases/latest/download/deltaflow-linux-amd64 \
            -o /usr/local/bin/deltaflow
          chmod +x /usr/local/bin/deltaflow

      - name: Run Deltaflow
        id: delta
        run: deltaflow detect --base=origin/main --format=gha

  # ── Step 2: deploy each affected service in parallel ───────────────────────
  deploy:
    name: Deploy ${{ matrix.service }}
    needs: detect
    if: needs.detect.outputs.has_changes == 'true'
    runs-on: ubuntu-latest

    strategy:
      fail-fast: false   # don't cancel other services if one fails
      matrix: ${{ fromJson(needs.detect.outputs.matrix) }}

    steps:
      - uses: actions/checkout@v4

      - name: Deploy ${{ matrix.service }}
        run: |
          echo "Deploying ${{ matrix.service }}..."
          # TODO: replace with your actual deploy command
          # e.g. kubectl apply -f services/${{ matrix.service }}/
          #      helm upgrade --install ${{ matrix.service }} ./charts/${{ matrix.service }}
          #      ./scripts/deploy.sh ${{ matrix.service }}
`
}

// sanitiseID converts a service name to a valid GitHub Actions job ID / output key.
// e.g. "auth-service" → "auth-service" (already valid), "my.service" → "my-service"
func sanitiseID(name string) string {
	return strings.NewReplacer(".", "-", "_", "-", " ", "-").Replace(name)
}

func init() {
	workflowInitCmd.Flags().BoolVar(&workflowMatrix, "matrix", false, "Generate a dynamic matrix workflow instead of per-service jobs")
	workflowInitCmd.Flags().StringVar(&workflowOut, "out", "", "Output path (default: .github/workflows/deltaflow.yml)")

	workflowCmd.AddCommand(workflowInitCmd)
	rootCmd.AddCommand(workflowCmd)
}
