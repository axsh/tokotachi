package resolve

import (
	"fmt"
	"os"
	"path/filepath"
)

// Worktree resolves the worktree path for the given branch.
// Unified structure: work/<branch>
func Worktree(repoRoot, branch string) (string, error) {
	path := filepath.Join(repoRoot, "work", branch)
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return path, nil
	}
	return "", fmt.Errorf("worktree for branch %q not found", branch)
}
