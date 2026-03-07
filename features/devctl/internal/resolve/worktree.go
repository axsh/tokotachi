package resolve

import (
	"fmt"
	"os"
	"path/filepath"
)

// Worktree resolves the worktree path for the given feature and branch.
// Search order: work/<feature>/<branch> → work/<feature> (backward compat).
// Returns an error if no directory exists.
func Worktree(repoRoot, feature, branch string) (string, error) {
	// Primary: work/<feature>/<branch>
	primary := filepath.Join(repoRoot, "work", feature, branch)
	if info, err := os.Stat(primary); err == nil && info.IsDir() {
		return primary, nil
	}

	// Fallback: work/<feature> (backward compatibility)
	fallback := filepath.Join(repoRoot, "work", feature)
	if info, err := os.Stat(fallback); err == nil && info.IsDir() {
		return fallback, nil
	}

	return "", fmt.Errorf("worktree for feature %q branch %q not found", feature, branch)
}
