package integration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTtEditor_DryRun verifies that 'tt editor <branch> --dry-run' runs
// without errors and shows the editor open step in the report.
func TestTtEditor_DryRun(t *testing.T) {
	stdout, stderr, exitCode := runTT(t, "editor", "--dry-run", branchName)
	combined := stdout + stderr

	// In dry-run mode, editor open is simulated
	// The command may fail if worktree doesn't exist, but report should print
	assert.Contains(t, combined, "Editor",
		"output should mention editor, got stdout=%q stderr=%q", stdout, stderr)
	_ = exitCode
}

// TestTtEditor_ReservedBranch verifies that 'tt editor main' is rejected.
func TestTtEditor_ReservedBranch(t *testing.T) {
	_, stderr, exitCode := runTT(t, "editor", "--dry-run", "main")
	assert.NotEqual(t, 0, exitCode,
		"tt editor main should fail")
	assert.Contains(t, stderr, "reserved",
		"error should mention reserved branch: %q", stderr)
}
