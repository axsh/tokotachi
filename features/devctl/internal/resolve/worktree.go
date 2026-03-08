package resolve

import (
	"fmt"
	"os"
	"path/filepath"
)

// Worktree resolves the worktree path for the given feature and branch.
// New structure:
//   - With feature: work/<branch>/features/<feature>
//   - Without feature: work/<branch>/all/
//
// Backward compatibility fallbacks:
//   - work/<feature>/<branch> (old structure)
//   - work/<feature> (old structure, feature-only)
func Worktree(repoRoot, feature, branch string) (string, error) {
	if feature == "" {
		// No feature: work/<branch>/all/
		allPath := filepath.Join(repoRoot, "work", branch, "all")
		if info, err := os.Stat(allPath); err == nil && info.IsDir() {
			return allPath, nil
		}
		return "", fmt.Errorf("worktree for branch %q (no feature) not found", branch)
	}

	// With feature
	// Priority 1: work/<branch>/features/<feature> (new structure)
	newPath := filepath.Join(repoRoot, "work", branch, "features", feature)
	if info, err := os.Stat(newPath); err == nil && info.IsDir() {
		return newPath, nil
	}

	// Priority 2: work/<feature>/<branch> (old structure - backward compat)
	oldPath := filepath.Join(repoRoot, "work", feature, branch)
	if info, err := os.Stat(oldPath); err == nil && info.IsDir() {
		return oldPath, nil
	}

	// Priority 3: work/<feature> (old structure - backward compat)
	oldFallback := filepath.Join(repoRoot, "work", feature)
	if info, err := os.Stat(oldFallback); err == nil && info.IsDir() {
		return oldFallback, nil
	}

	return "", fmt.Errorf("worktree for feature %q branch %q not found", feature, branch)
}
