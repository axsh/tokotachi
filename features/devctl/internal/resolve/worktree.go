package resolve

import (
	"fmt"
	"os"
	"path/filepath"
)

// Worktree resolves the worktree path for the given feature.
// Returns an error if the directory does not exist.
func Worktree(repoRoot, feature string) (string, error) {
	path := filepath.Join(repoRoot, "work", feature)
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("worktree for feature %q not found: %w", feature, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("worktree path is not a directory: %s", path)
	}
	return path, nil
}
