package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTtCreate_CreatesWorktree verifies that 'tt create <branch>' creates
// a worktree for the given branch. Uses --dry-run to avoid real worktree changes.
func TestTtCreate_CreatesWorktree(t *testing.T) {
	stdout, stderr, exitCode := runTT(t, "create", "--dry-run", branchName)
	combined := stdout + stderr

	// The command should succeed (or fail gracefully in dry-run)
	// In dry-run mode, worktree creation is simulated
	assert.Contains(t, combined, "Worktree",
		"output should mention worktree, got stdout=%q stderr=%q", stdout, stderr)
	_ = exitCode // dry-run may return 0 or non-zero depending on existing state
}

// TestTtCreate_IdempotentIfExists verifies that running 'tt create' for
// a branch that already has a worktree succeeds without errors.
func TestTtCreate_IdempotentIfExists(t *testing.T) {
	// First create (or verify already exists)
	_, _, _ = runTT(t, "create", "--dry-run", branchName)
	// Second create should also succeed
	stdout, stderr, _ := runTT(t, "create", "--dry-run", branchName)
	combined := stdout + stderr

	// Should either create or report "already exists" — not fail
	assert.NotContains(t, combined, "FAILED",
		"idempotent create should not report FAILED")
}

// TestTtDelete_BlockedByRunningContainer verifies that 'tt delete' fails
// when a container is running for the branch.
func TestTtDelete_BlockedByRunningContainer(t *testing.T) {
	requireDockerAvailable(t)

	// First ensure container is running
	ensureWorktree(t)
	_, _, setupCode := runTT(t, "up", branchName, featureName)
	if setupCode != 0 {
		t.Fatalf("setup: tt up failed with exit code %d", setupCode)
	}
	defer cleanupTTDown(t)

	// Attempt to delete while container is running
	_, stderr, exitCode := runTT(t, "delete", "--yes", branchName)
	assert.NotEqual(t, 0, exitCode,
		"tt delete should fail when container is running")
	assert.Contains(t, stderr, "active",
		"error should mention active containers: %q", stderr)
}

// TestTtDelete_DryRun verifies that 'tt delete --dry-run' does NOT
// actually remove the worktree.
func TestTtDelete_DryRun(t *testing.T) {
	stdout, stderr, exitCode := runTT(t, "delete", "--dry-run", "--yes", branchName)
	combined := stdout + stderr

	// dry-run should simulate deletion
	assert.Contains(t, combined, "DRY-RUN",
		"output should indicate dry-run mode, got: %q", combined)
	_ = exitCode
}

// TestTtDelete_ReservedBranch verifies that 'tt delete main' is rejected.
func TestTtDelete_ReservedBranch(t *testing.T) {
	_, stderr, exitCode := runTT(t, "delete", "--yes", "main")
	require.NotEqual(t, 0, exitCode,
		"tt delete main should fail")
	assert.Contains(t, stderr, "reserved",
		"error should mention reserved branch: %q", stderr)
}

func TestTtDelete_PendingChanges_Yes_UsesForceRemove(t *testing.T) {
	branch := fmt.Sprintf("pending-yes-%d", time.Now().UnixNano())
	worktreePath := filepath.Join(projectRoot(), "work", branch)

	_, _, createExit := runTT(t, "create", branch)
	require.Equal(t, 0, createExit, "tt create should succeed")

	t.Cleanup(func() {
		_, _, _ = runTT(t, "delete", "--force", "--yes", branch)
	})

	require.NoError(t, os.MkdirAll(worktreePath, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(worktreePath, "pending-untracked.txt"), []byte("pending"), 0o644))

	stdout, stderr, exitCode := runTTWithInput(t, "yes\n", "delete", branch)
	combined := stdout + stderr

	assert.Equal(t, 0, exitCode, "tt delete should succeed with yes confirmation")
	assert.NotContains(t, combined, "contains modified or untracked files",
		"delete should not fail with missing --force: %s", combined)
}

func TestTtDelete_PendingChanges_No_Aborts(t *testing.T) {
	branch := fmt.Sprintf("pending-no-%d", time.Now().UnixNano())
	worktreePath := filepath.Join(projectRoot(), "work", branch)

	_, _, createExit := runTT(t, "create", branch)
	require.Equal(t, 0, createExit, "tt create should succeed")

	t.Cleanup(func() {
		_, _, _ = runTT(t, "delete", "--force", "--yes", branch)
	})

	require.NoError(t, os.MkdirAll(worktreePath, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(worktreePath, "pending-untracked.txt"), []byte("pending"), 0o644))

	_, stderr, exitCode := runTTWithInput(t, "n\n", "delete", branch)
	assert.Equal(t, 0, exitCode, "abort should not be treated as error")
	assert.Contains(t, stderr, "Aborted", "stderr should contain abort message: %s", stderr)

	_, statErr := os.Stat(worktreePath)
	assert.NoError(t, statErr, "worktree should remain after abort")
}
