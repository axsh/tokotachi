package action_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/axsh/tokotachi/pkg/action"
	"github.com/axsh/tokotachi/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDelete_RemovesWorktreeAndBranch(t *testing.T) {
	env := newTestEnv(t)
	branch := "test-branch"

	// Setup: create worktree directory with .git file
	wtDir := filepath.Join(env.RepoRoot, "work", branch)
	require.NoError(t, os.MkdirAll(wtDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(wtDir, ".git"),
		[]byte("gitdir: ../../.git/worktrees/test-branch\n"), 0o644))

	// No active features (no state file = no containers)
	err := env.Runner.Delete(action.DeleteOptions{
		Branch:      branch,
		Force:       false,
		RepoRoot:    env.RepoRoot,
		ProjectName: "test",
		Depth:       10,
		Yes:         true,
	}, env.WM)
	require.NoError(t, err)

	// Verify: worktree remove + branch delete commands recorded
	recs := env.Recorder.Records()
	assert.True(t, hasRecordContaining(recs, "worktree remove"),
		"worktree remove should be called, got records: %v", recs)
	assert.True(t, hasRecordContaining(recs, "branch -d"),
		"branch delete should be called, got records: %v", recs)
}

func TestDelete_BlockedByActiveContainers(t *testing.T) {
	env := newTestEnv(t)
	branch := "test-branch"

	// Setup: active feature in state file
	setupStateFile(t, env.RepoRoot, branch, map[string]state.FeatureState{
		"myfeature": {Status: state.StatusActive, StartedAt: time.Now()},
	})

	// Create worktree directory
	wtDir := filepath.Join(env.RepoRoot, "work", branch)
	require.NoError(t, os.MkdirAll(wtDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(wtDir, ".git"),
		[]byte("gitdir: ../../.git/worktrees/test-branch\n"), 0o644))

	err := env.Runner.Delete(action.DeleteOptions{
		Branch:      branch,
		Force:       false,
		RepoRoot:    env.RepoRoot,
		ProjectName: "test",
		Depth:       10,
		Yes:         true,
	}, env.WM)

	// Should return error due to active containers
	require.Error(t, err)
	assert.Contains(t, err.Error(), "active",
		"error should mention active containers: %v", err)

	// Verify: no worktree or branch commands executed
	recs := env.Recorder.Records()
	assert.False(t, hasRecordContaining(recs, "worktree remove"),
		"worktree remove should NOT be called when active containers exist")
}

func TestDelete_NoStateFile_StillDeletesWorktree(t *testing.T) {
	env := newTestEnv(t)
	branch := "test-branch"

	// No state file exists — this is fine, delete should still work
	wtDir := filepath.Join(env.RepoRoot, "work", branch)
	require.NoError(t, os.MkdirAll(wtDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(wtDir, ".git"),
		[]byte("gitdir: ../../.git/worktrees/test-branch\n"), 0o644))

	err := env.Runner.Delete(action.DeleteOptions{
		Branch:      branch,
		Force:       false,
		RepoRoot:    env.RepoRoot,
		ProjectName: "test",
		Depth:       10,
		Yes:         true,
	}, env.WM)
	require.NoError(t, err)

	recs := env.Recorder.Records()
	assert.True(t, hasRecordContaining(recs, "worktree remove"),
		"worktree remove should be called even without state file")
}

func TestDelete_StoppedFeatures_AllowsDeletion(t *testing.T) {
	env := newTestEnv(t)
	branch := "test-branch"

	// Setup: feature with "stopped" status (not "active")
	setupStateFile(t, env.RepoRoot, branch, map[string]state.FeatureState{
		"myfeature": {Status: state.StatusStopped, StartedAt: time.Now()},
	})

	wtDir := filepath.Join(env.RepoRoot, "work", branch)
	require.NoError(t, os.MkdirAll(wtDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(wtDir, ".git"),
		[]byte("gitdir: ../../.git/worktrees/test-branch\n"), 0o644))

	err := env.Runner.Delete(action.DeleteOptions{
		Branch:      branch,
		Force:       false,
		RepoRoot:    env.RepoRoot,
		ProjectName: "test",
		Depth:       10,
		Yes:         true,
	}, env.WM)
	require.NoError(t, err)

	recs := env.Recorder.Records()
	assert.True(t, hasRecordContaining(recs, "worktree remove"),
		"worktree remove should be called when features are stopped")
}

func TestDelete_Force_RunsPrune(t *testing.T) {
	env := newTestEnv(t)
	branch := "test-branch"

	wtDir := filepath.Join(env.RepoRoot, "work", branch)
	require.NoError(t, os.MkdirAll(wtDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(wtDir, ".git"),
		[]byte("gitdir: ../../.git/worktrees/test-branch\n"), 0o644))

	err := env.Runner.Delete(action.DeleteOptions{
		Branch:      branch,
		Force:       true,
		RepoRoot:    env.RepoRoot,
		ProjectName: "test",
		Depth:       10,
		Yes:         true,
	}, env.WM)
	require.NoError(t, err)

	recs := env.Recorder.Records()
	assert.True(t, hasRecordContaining(recs, "worktree prune"),
		"worktree prune should be called with force, recs: %v", recs)
	assert.True(t, hasRecordContaining(recs, "-f"),
		"force flag should be propagated, recs: %v", recs)
}

func TestDelete_WithNestedWorktrees_DeletesChildrenFirst(t *testing.T) {
	env := newTestEnv(t)
	parentBranch := "parent-branch"
	childBranch := "child-branch"

	// Setup parent worktree
	parentDir := filepath.Join(env.RepoRoot, "work", parentBranch)
	require.NoError(t, os.MkdirAll(parentDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(parentDir, ".git"),
		[]byte("gitdir: ../../.git/worktrees/parent-branch\n"), 0o644))

	// Setup child worktree inside parent's work/ directory
	childDir := filepath.Join(parentDir, "work", childBranch)
	require.NoError(t, os.MkdirAll(childDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(childDir, ".git"),
		[]byte("gitdir: ../../../../.git/worktrees/child-branch\n"), 0o644))

	err := env.Runner.Delete(action.DeleteOptions{
		Branch:      parentBranch,
		Force:       false,
		RepoRoot:    env.RepoRoot,
		ProjectName: "test",
		Depth:       10,
		Yes:         true,
	}, env.WM)
	require.NoError(t, err)

	// Verify command order: child delete before parent delete
	recs := env.Recorder.Records()
	childIdx := -1
	parentIdx := -1
	for i, r := range recs {
		if strings.Contains(r.Command, "worktree remove") && strings.Contains(r.Command, childBranch) {
			childIdx = i
		}
		if strings.Contains(r.Command, "worktree remove") && strings.Contains(r.Command, parentBranch) {
			parentIdx = i
		}
	}
	assert.Greater(t, parentIdx, childIdx,
		"child worktree remove should come before parent worktree remove, recs: %v", recs)

	// Verify child worktree remove uses the correct path under parent's work/ directory
	expectedChildPath := filepath.Join(env.RepoRoot, "work", parentBranch, "work", childBranch)
	if childIdx >= 0 {
		assert.Contains(t, recs[childIdx].Command, expectedChildPath,
			"child worktree remove should use path under parent worktree, recs: %v", recs)
	}
}

func TestDelete_WithNestedWorktrees_UsesCorrectChildPath(t *testing.T) {
	env := newTestEnv(t)
	parentBranch := "parent-branch"
	childBranch := "child-branch"

	// Setup parent worktree
	parentDir := filepath.Join(env.RepoRoot, "work", parentBranch)
	require.NoError(t, os.MkdirAll(parentDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(parentDir, ".git"),
		[]byte("gitdir: ../../.git/worktrees/parent-branch\n"), 0o644))

	// Setup child worktree inside parent's work/ directory
	childDir := filepath.Join(parentDir, "work", childBranch)
	require.NoError(t, os.MkdirAll(childDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(childDir, ".git"),
		[]byte("gitdir: ../../../../.git/worktrees/child-branch\n"), 0o644))

	err := env.Runner.Delete(action.DeleteOptions{
		Branch:      parentBranch,
		Force:       false,
		RepoRoot:    env.RepoRoot,
		ProjectName: "test",
		Depth:       10,
		Yes:         true,
	}, env.WM)
	require.NoError(t, err)

	recs := env.Recorder.Records()

	// Verify: child worktree remove uses path rooted at parent worktree
	// Expected: <RepoRoot>/work/parent-branch/work/child-branch
	// Bug path: <RepoRoot>/work/child-branch (incorrect, before fix)
	expectedChildPath := filepath.Join(env.RepoRoot, "work", parentBranch, "work", childBranch)
	childRemoveFound := false
	for _, r := range recs {
		if strings.Contains(r.Command, "worktree remove") && strings.Contains(r.Command, childBranch) {
			childRemoveFound = true
			assert.Contains(t, r.Command, expectedChildPath,
				"child worktree remove must use path under parent worktree directory")
			// Ensure it does NOT use the root-level path
			incorrectPath := filepath.Join(env.RepoRoot, "work", childBranch)
			if !strings.Contains(expectedChildPath, incorrectPath) {
				assert.NotContains(t, r.Command, incorrectPath,
					"child worktree remove must NOT use root-level path")
			}
		}
	}
	assert.True(t, childRemoveFound,
		"worktree remove for child-branch should have been called, recs: %v", recs)

	// Verify: parent worktree remove uses root-level path
	expectedParentPath := filepath.Join(env.RepoRoot, "work", parentBranch)
	for _, r := range recs {
		if strings.Contains(r.Command, "worktree remove") && strings.Contains(r.Command, parentBranch) {
			assert.Contains(t, r.Command, expectedParentPath,
				"parent worktree remove should use root-level path")
		}
	}
}

func TestDelete_ConfirmNo_Aborts(t *testing.T) {
	env := newTestEnv(t)
	branch := "test-branch"

	wtDir := filepath.Join(env.RepoRoot, "work", branch)
	require.NoError(t, os.MkdirAll(wtDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(wtDir, ".git"),
		[]byte("gitdir: ../../.git/worktrees/test-branch\n"), 0o644))

	err := env.Runner.Delete(action.DeleteOptions{
		Branch:      branch,
		Force:       false,
		RepoRoot:    env.RepoRoot,
		ProjectName: "test",
		Yes:         false,
		Depth:       10,
		Stdin:       strings.NewReader("N\n"),
	}, env.WM)
	require.NoError(t, err) // Abort is not an error

	recs := env.Recorder.Records()
	assert.False(t, hasRecordContaining(recs, "worktree remove"),
		"worktree remove should NOT be called after 'N', recs: %v", recs)
}

func TestDelete_RemovesStateFile(t *testing.T) {
	env := newTestEnv(t)
	branch := "test-branch"

	// Setup: stopped feature in state file
	setupStateFile(t, env.RepoRoot, branch, map[string]state.FeatureState{
		"myfeature": {Status: state.StatusStopped, StartedAt: time.Now()},
	})

	wtDir := filepath.Join(env.RepoRoot, "work", branch)
	require.NoError(t, os.MkdirAll(wtDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(wtDir, ".git"),
		[]byte("gitdir: ../../.git/worktrees/test-branch\n"), 0o644))

	err := env.Runner.Delete(action.DeleteOptions{
		Branch:      branch,
		Force:       false,
		RepoRoot:    env.RepoRoot,
		ProjectName: "test",
		Depth:       10,
		Yes:         true,
	}, env.WM)
	require.NoError(t, err)

	// Verify state file is removed
	statePath := state.StatePath(env.RepoRoot, branch)
	_, statErr := os.Stat(statePath)
	assert.True(t, os.IsNotExist(statErr), "state file should be deleted after delete")
}
