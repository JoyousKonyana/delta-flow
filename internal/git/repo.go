package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// IsGitRepo returns true if the current working directory is inside a git repository.
func IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	err := cmd.Run()
	return err == nil
}

// IsShallowClone returns true if the repository is a shallow clone
// (i.e. the .git/shallow file exists).
func IsShallowClone() bool {
	root, err := GetRepoRoot()
	if err != nil {
		return false
	}
	shallowPath := root + "/.git/shallow"
	_, err = os.Stat(shallowPath)
	return err == nil
}

// GetRepoRoot returns the absolute path of the repository's top-level directory.
func GetRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("could not determine repo root: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("could not determine repo root: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// BranchExists returns true if the given branch (or ref) exists in the repository.
func BranchExists(branch string) (bool, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	_, err := cmd.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// Non-zero exit means the ref does not exist — not an error condition.
			return false, nil
		}
		return false, fmt.Errorf("failed to verify branch %q: %w", branch, err)
	}
	return true, nil
}
