package action_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/action"
	"github.com/axsh/tokotachi/features/tt/internal/state"
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
