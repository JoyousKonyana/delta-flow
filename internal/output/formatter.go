package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/JoyousKonyana/deltaflow/internal/matcher"
	"github.com/fatih/color"
)

// Format constants matching the --format flag values.
const (
	FormatTable     = "table"
	FormatJSON      = "json"
	FormatGHA       = "gha"
	FormatGHAMatrix = "gha-matrix"
	FormatEnv       = "env"
)

// Print routes the result to the correct formatter.
// allServices is the full list of services from config — needed for per-service booleans in GHA output.
func Print(result *matcher.Result, allServices []string, format string, dryRun bool) error {
	switch format {
	case FormatTable:
		printTable(result, dryRun)
	case FormatJSON:
		return printJSON(result)
	case FormatGHA:
		return printGHA(result, allServices)
	case FormatGHAMatrix:
		return printGHAMatrix(result)
	case FormatEnv:
		printEnv(result)
	default:
		return fmt.Errorf("unknown format %q — valid options: table|json|gha|gha-matrix|env", format)
	}
	return nil
}

// --- table ---

func printTable(result *matcher.Result, dryRun bool) {
	affected := color.New(color.FgGreen, color.Bold)
	skipped := color.New(color.FgHiBlack)
	header := color.New(color.FgCyan, color.Bold)

	fmt.Println()
	header.Println("  SERVICE                        STATUS       TRIGGERED BY")
	header.Println("  ─────────────────────────────  ───────────  ────────────────────────────────")

	if result.GlobalTrigger {
		color.Yellow("  ⚡ Global trigger matched — all services will be deployed\n")
	}

	for _, name := range result.Affected {
		files := result.TriggeredBy[name]
		affected.Printf("  %-31s", name)
		fmt.Printf("%-13s", "AFFECTED")
		affected.Printf("%s\n", summariseFiles(files))
	}

	for _, name := range result.Skipped {
		skipped.Printf("  %-31s%-13s%s\n", name, "skipped", "—")
	}

	fmt.Println()

	if dryRun {
		color.Cyan("  [dry-run] No pipelines were triggered.")
	} else {
		color.Green("  ✔  %d service(s) affected, %d skipped.", len(result.Affected), len(result.Skipped))
	}

	fmt.Println()
}

// --- json ---

type jsonOutput struct {
	Affected      []string            `json:"affected"`
	Skipped       []string            `json:"skipped"`
	GlobalTrigger bool                `json:"global_trigger"`
	TriggeredBy   map[string][]string `json:"triggered_by"`
}

func printJSON(result *matcher.Result) error {
	out := jsonOutput{
		Affected:      orEmpty(result.Affected),
		Skipped:       orEmpty(result.Skipped),
		GlobalTrigger: result.GlobalTrigger,
		TriggeredBy:   result.TriggeredBy,
	}
	if out.TriggeredBy == nil {
		out.TriggeredBy = map[string][]string{}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// --- gha ---
// Writes to $GITHUB_OUTPUT (or stdout in local dev).
//
// Outputs the following keys:
//   affected=auth-service,payments-service   CSV of affected services
//   has_changes=true                         whether any service is affected
//   global_trigger=false                     whether a global trigger fired
//   matrix={"include":[{"service":"..."}]}   dynamic matrix for strategy.matrix
//   auth-service=true                        per-service boolean for if: conditions
//   payments-service=false
//   ...one line per service in config

func printGHA(result *matcher.Result, allServices []string) error {
	affectedSet := make(map[string]bool, len(result.Affected))
	for _, s := range result.Affected {
		affectedSet[s] = true
	}

	// Build matrix JSON
	type matrixEntry struct {
		Service string `json:"service"`
	}
	entries := make([]matrixEntry, 0, len(result.Affected))
	for _, s := range result.Affected {
		entries = append(entries, matrixEntry{Service: s})
	}
	matrixBytes, err := json.Marshal(struct {
		Include []matrixEntry `json:"include"`
	}{Include: entries})
	if err != nil {
		return fmt.Errorf("failed to build matrix JSON: %w", err)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("affected=%s\n", strings.Join(result.Affected, ",")))
	b.WriteString(fmt.Sprintf("has_changes=%v\n", len(result.Affected) > 0))
	b.WriteString(fmt.Sprintf("global_trigger=%v\n", result.GlobalTrigger))
	b.WriteString(fmt.Sprintf("matrix=%s\n", string(matrixBytes)))

	// Per-service booleans — written for every service so pipelines can always
	// reference ${{ steps.detect.outputs.auth-service }} without a nil check.
	for _, svc := range allServices {
		b.WriteString(fmt.Sprintf("%s=%v\n", svc, affectedSet[svc]))
	}

	return writeGHAOutput(b.String())
}

// --- gha-matrix ---
// Outputs only the matrix JSON to stdout. Useful when you only need the matrix
// and want to pipe it or capture it separately.

func printGHAMatrix(result *matcher.Result) error {
	type entry struct {
		Service string `json:"service"`
	}
	entries := make([]entry, 0, len(result.Affected))
	for _, s := range result.Affected {
		entries = append(entries, entry{Service: s})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(struct {
		Include []entry `json:"include"`
	}{Include: entries})
}

// writeGHAOutput writes to $GITHUB_OUTPUT when running in CI,
// or prints a preview to stdout during local development.
func writeGHAOutput(content string) error {
	githubOutput := os.Getenv("GITHUB_OUTPUT")
	if githubOutput == "" {
		color.Cyan("# [gha] $GITHUB_OUTPUT not set — local preview:\n")
		fmt.Print(content)
		return nil
	}
	f, err := os.OpenFile(githubOutput, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open $GITHUB_OUTPUT: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to $GITHUB_OUTPUT: %w", err)
	}
	return nil
}

// --- env ---

func printEnv(result *matcher.Result) {
	fmt.Printf("AFFECTED_SERVICES=%s\n", strings.Join(result.Affected, ","))
	fmt.Printf("SKIPPED_SERVICES=%s\n", strings.Join(result.Skipped, ","))
	fmt.Printf("HAS_CHANGES=%v\n", len(result.Affected) > 0)
	fmt.Printf("GLOBAL_TRIGGER=%v\n", result.GlobalTrigger)
}

// --- helpers ---

func summariseFiles(files []string) string {
	if len(files) == 0 {
		return "—"
	}
	const max = 2
	shown := files
	if len(files) > max {
		shown = files[:max]
	}
	out := strings.Join(shown, ", ")
	if len(files) > max {
		out += fmt.Sprintf(" (+%d more)", len(files)-max)
	}
	return out
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
