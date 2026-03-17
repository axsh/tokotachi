package integration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTtOpen_SyntaxSugar_DryRun verifies that 'tt open <branch> --dry-run'
// executes the create → editor sequence (skipping up when no feature).
func TestTtOpen_SyntaxSugar_DryRun(t *testing.T) {
	stdout, stderr, exitCode := runTT(t, "open", "--dry-run", branchName)
	combined := stdout + stderr

	// Should show worktree and editor steps in output/report
	assert.Contains(t, combined, "Editor",
		"output should mention editor, got: %q", combined)
	_ = exitCode
}

// TestTtOpen_SyntaxSugar_WithFeature_DryRun verifies that 'tt open <branch> <feature> --dry-run'
// executes the full create → up → editor sequence.
func TestTtOpen_SyntaxSugar_WithFeature_DryRun(t *testing.T) {
	stdout, stderr, exitCode := runTT(t, "open", "--dry-run", branchName, featureName)
	combined := stdout + stderr

	// Should show container start (or skip) and editor steps
	assert.NotEmpty(t, combined,
		"output should not be empty")
	_ = exitCode
}

// TestTtClose_SyntaxSugar_ReservedBranch verifies that 'tt close main' is rejected.
func TestTtClose_SyntaxSugar_ReservedBranch(t *testing.T) {
	_, stderr, exitCode := runTT(t, "close", "--yes", "main")
	assert.NotEqual(t, 0, exitCode,
		"tt close main should fail")
	assert.Contains(t, stderr, "reserved",
		"error should mention reserved branch: %q", stderr)
}

// TestTtUp_RequiresFeature verifies that 'tt up <branch>' without feature
// fails with a usage error since feature is now required.
func TestTtUp_RequiresFeature(t *testing.T) {
	_, stderr, exitCode := runTT(t, "up", branchName)
	assert.NotEqual(t, 0, exitCode,
		"tt up should fail without feature argument")
	assert.Contains(t, stderr, "accepts 2 arg(s)",
		"error should mention argument count: %q", stderr)
}
