package resolve

import (
	"fmt"
	"os"
	"path/filepath"
)

// Worktree resolves the worktree path for the given branch.
// Returns error if directory exists but is not a valid git worktree (ghost directory).
func Worktree(repoRoot, branch string) (string, error) {
	path := filepath.Join(repoRoot, "work", branch)
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		// Validate: must have .git file (worktree) or .git directory
		gitPath := filepath.Join(path, ".git")
		if _, gitErr := os.Stat(gitPath); gitErr == nil {
			return path, nil
		}
		return "", fmt.Errorf("worktree for branch %q exists but is not a valid git worktree (ghost directory)", branch)
	}
	return "", fmt.Errorf("worktree for branch %q not found", branch)
}
