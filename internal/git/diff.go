package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetChangedFiles returns a list of files changed between baseBranch and headCommit
// by running: git diff --name-only baseBranch..headCommit
func GetChangedFiles(baseBranch string, headCommit string) ([]string, error) {
	ref := fmt.Sprintf("%s..%s", baseBranch, headCommit)

	cmd := exec.Command("git", "diff", "--name-only", ref)
	output, err := cmd.Output()
	if err != nil {
		// exec.ExitError carries stderr, surface it for a descriptive message
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git diff failed (ref: %s): %s", ref, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git diff failed (ref: %s): %w", ref, err)
	}

	raw := strings.Split(string(output), "\n")

	files := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Normalise backslashes to forward slashes for Windows compatibility
		line = strings.ReplaceAll(line, "\\", "/")
		files = append(files, line)
	}

	return files, nil
}
