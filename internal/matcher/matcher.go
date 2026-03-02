package matcher

import (
	"fmt"

	"github.com/JoyousKonyana/deltaflow/internal/config"
	"github.com/bmatcuk/doublestar/v4"
)

// Result holds the outcome of a match operation against changed files.
type Result struct {
	Affected      []string            // Service names that are affected
	Skipped       []string            // Service names with no matching changes
	GlobalTrigger bool                // True if a global trigger pattern matched
	TriggeredBy   map[string][]string // Service name → matched file paths
}

// Match compares changedFiles against the config and returns which services are affected.
func Match(changedFiles []string, cfg *config.Config) (*Result, error) {
	result := &Result{
		TriggeredBy: make(map[string][]string),
	}

	// --- Step 1: check global triggers ---
	for _, pattern := range cfg.GlobalTriggers {
		for _, file := range changedFiles {
			matched, err := doublestar.Match(pattern, file)
			if err != nil {
				return nil, fmt.Errorf("invalid global trigger pattern %q: %w", pattern, err)
			}
			if matched {
				result.GlobalTrigger = true
				break
			}
		}
		if result.GlobalTrigger {
			break
		}
	}

	// If a global trigger fired, all services are affected.
	if result.GlobalTrigger {
		result.Affected = cfg.ServiceNames()
		return result, nil
	}

	// --- Step 2: per-service pattern matching ---
	for _, name := range cfg.ServiceNames() {
		svc := cfg.Services[name]
		var hits []string

		for _, file := range changedFiles {
			for _, pattern := range svc.Paths {
				matched, err := doublestar.Match(pattern, file)
				if err != nil {
					return nil, fmt.Errorf("invalid path pattern %q for service %q: %w", pattern, name, err)
				}
				if matched {
					hits = append(hits, file)
					break // one pattern match per file is enough
				}
			}
		}

		if len(hits) > 0 {
			result.Affected = append(result.Affected, name)
			result.TriggeredBy[name] = hits
		} else {
			result.Skipped = append(result.Skipped, name)
		}
	}

	return result, nil
}
