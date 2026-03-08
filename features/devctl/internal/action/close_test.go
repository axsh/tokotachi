package action_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/axsh/tokotachi/features/devctl/internal/action"
	"github.com/axsh/tokotachi/features/devctl/internal/cmdexec"
	"github.com/axsh/tokotachi/features/devctl/internal/log"
	"github.com/axsh/tokotachi/features/devctl/internal/state"
	"github.com/axsh/tokotachi/features/devctl/internal/worktree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEnv holds shared test objects with a single cmdexec.Runner and Recorder.
type testEnv struct {
	Runner   *action.Runner
	WM       *worktree.Manager
	Recorder *cmdexec.Recorder
	RepoRoot string
}

// newTestEnv creates a shared test environment where Runner and worktree.Manager
// use the same cmdexec.Runner (and thus the same Recorder).
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	repoRoot := t.TempDir()
	logger := log.New(os.Stderr, false)
	rec := cmdexec.NewRecorder()
	cmdRunner := &cmdexec.Runner{Logger: logger, DryRun: true, Recorder: rec}
	runner := &action.Runner{Logger: logger, DryRun: true, CmdRunner: cmdRunner}
	wm := &worktree.Manager{CmdRunner: cmdRunner, RepoRoot: repoRoot}
	return &testEnv{Runner: runner, WM: wm, Recorder: rec, RepoRoot: repoRoot}
}

// setupStateFile creates a state file with the given features for testing.
func setupStateFile(t *testing.T, repoRoot, branch string, features map[string]state.FeatureState) {
	t.Helper()
	sf := state.StateFile{
		Branch:    branch,
		CreatedAt: time.Now(),
		Features:  features,
	}
	statePath := state.StatePath(repoRoot, branch)
	require.NoError(t, os.MkdirAll(filepath.Dir(statePath), 0o755))
	require.NoError(t, state.Save(statePath, sf))
}

// hasRecordContaining checks if any recorder entry contains the given substring.
func hasRecordContaining(recs []cmdexec.ExecRecord, substr string) bool {
	for _, r := range recs {
		if strings.Contains(r.Command, substr) {
			return true
		}
	}
	return false
}

func TestClose_WithFeature_LastFeature_CleansUpWorktree(t *testing.T) {
	env := newTestEnv(t)
	branch := "test-branch"

	// Setup: one feature in state
	setupStateFile(t, env.RepoRoot, branch, map[string]state.FeatureState{
		"myfeature": {Status: state.StatusActive, StartedAt: time.Now()},
	})

	// Create worktree directory so wm.Exists() returns true
	wtDir := filepath.Join(env.RepoRoot, "work", branch)
	require.NoError(t, os.MkdirAll(wtDir, 0o755))
	// Add .git file so wm.Exists() recognizes this as a valid worktree
	require.NoError(t, os.WriteFile(filepath.Join(wtDir, ".git"), []byte("gitdir: ../../.git/worktrees/test-branch\n"), 0o644))

	err := env.Runner.Close(action.CloseOptions{
		Feature:     "myfeature",
		Branch:      branch,
		Force:       false,
		RepoRoot:    env.RepoRoot,
		ProjectName: "test",
		Yes:         true,
		Depth:       10,
	}, env.WM)
	require.NoError(t, err)

	// Verify: state file should be removed (auto cleanup)
	statePath := state.StatePath(env.RepoRoot, branch)
	_, statErr := os.Stat(statePath)
	assert.True(t, os.IsNotExist(statErr), "state file should be deleted after last feature close")

	// Verify: worktree remove command should be recorded
	recs := env.Recorder.Records()
	assert.True(t, hasRecordContaining(recs, "worktree remove"),
		"worktree remove should be called, got records: %v", recs)
}

func TestClose_WithFeature_OtherFeaturesRemain_KeepsWorktree(t *testing.T) {
	env := newTestEnv(t)
	branch := "test-branch"

	// Setup: two features in state
	setupStateFile(t, env.RepoRoot, branch, map[string]state.FeatureState{
		"feature-a": {Status: state.StatusActive, StartedAt: time.Now()},
		"feature-b": {Status: state.StatusActive, StartedAt: time.Now()},
	})

	// Create worktree directory
	wtDir := filepath.Join(env.RepoRoot, "work", branch)
	require.NoError(t, os.MkdirAll(wtDir, 0o755))
	// Add .git file so wm.Exists() recognizes this as a valid worktree
	require.NoError(t, os.WriteFile(filepath.Join(wtDir, ".git"), []byte("gitdir: ../../.git/worktrees/test-branch\n"), 0o644))

	err := env.Runner.Close(action.CloseOptions{
		Feature:     "feature-a",
		Branch:      branch,
		Force:       false,
		RepoRoot:    env.RepoRoot,
		ProjectName: "test",
		Yes:         true,
		Depth:       10,
	}, env.WM)
	require.NoError(t, err)

	// Verify: state file should still exist
	statePath := state.StatePath(env.RepoRoot, branch)
	_, statErr := os.Stat(statePath)
	assert.False(t, os.IsNotExist(statErr), "state file should still exist")

	// Verify: remaining feature should be in state
	sf, loadErr := state.Load(statePath)
	require.NoError(t, loadErr)
	assert.Contains(t, sf.Features, "feature-b", "feature-b should remain in state")
	assert.NotContains(t, sf.Features, "feature-a", "feature-a should be removed from state")

	// Verify: worktree remove should NOT be called
	recs := env.Recorder.Records()
	assert.False(t, hasRecordContaining(recs, "worktree remove"),
		"worktree remove should not be called when other features remain")
}

func TestClose_WithFeature_Force_PropagatedToCleanup(t *testing.T) {
	env := newTestEnv(t)
	branch := "test-branch"

	// Setup: one feature in state
	setupStateFile(t, env.RepoRoot, branch, map[string]state.FeatureState{
		"myfeature": {Status: state.StatusActive, StartedAt: time.Now()},
	})

	// Create worktree directory
	wtDir := filepath.Join(env.RepoRoot, "work", branch)
	require.NoError(t, os.MkdirAll(wtDir, 0o755))
	// Add .git file so wm.Exists() recognizes this as a valid worktree
	require.NoError(t, os.WriteFile(filepath.Join(wtDir, ".git"), []byte("gitdir: ../../.git/worktrees/test-branch\n"), 0o644))

	err := env.Runner.Close(action.CloseOptions{
		Feature:     "myfeature",
		Branch:      branch,
		Force:       true, // Force=true
		RepoRoot:    env.RepoRoot,
		ProjectName: "test",
		Yes:         true,
		Depth:       10,
	}, env.WM)
	require.NoError(t, err)

	// Verify: worktree remove with -f flag
	recs := env.Recorder.Records()
	assert.True(t, hasRecordContaining(recs, "-f"),
		"force flag should be propagated to worktree remove, got records: %v", recs)

	// Verify: branch delete with -D flag
	assert.True(t, hasRecordContaining(recs, "-D"),
		"force flag should be propagated to branch delete, got records: %v", recs)
}

func TestClose_WithoutFeature_Unchanged(t *testing.T) {
	env := newTestEnv(t)
	branch := "test-branch"

	// Setup: one feature in state
	setupStateFile(t, env.RepoRoot, branch, map[string]state.FeatureState{
		"myfeature": {Status: state.StatusActive, StartedAt: time.Now()},
	})

	// Create worktree directory
	wtDir := filepath.Join(env.RepoRoot, "work", branch)
	require.NoError(t, os.MkdirAll(wtDir, 0o755))
	// Add .git file so wm.Exists() recognizes this as a valid worktree
	require.NoError(t, os.WriteFile(filepath.Join(wtDir, ".git"), []byte("gitdir: ../../.git/worktrees/test-branch\n"), 0o644))

	err := env.Runner.Close(action.CloseOptions{
		Feature:     "", // No feature = close all
		Branch:      branch,
		Force:       false,
		RepoRoot:    env.RepoRoot,
		ProjectName: "test",
		Yes:         true,
		Depth:       10,
	}, env.WM)
	require.NoError(t, err)

	// Verify: state file should be removed
	statePath := state.StatePath(env.RepoRoot, branch)
	_, statErr := os.Stat(statePath)
	assert.True(t, os.IsNotExist(statErr), "state file should be deleted")

	// Verify: worktree remove + branch delete commands recorded
	recs := env.Recorder.Records()
	assert.True(t, hasRecordContaining(recs, "worktree remove"),
		"worktree remove should be called, got records: %v", recs)
	assert.True(t, hasRecordContaining(recs, "branch -d"),
		"branch delete should be called, got records: %v", recs)
}

func TestClose_WithNestedWorktrees_ClosesChildrenFirst(t *testing.T) {
	env := newTestEnv(t)
	parentBranch := "parent-branch"
	childBranch := "child-branch"

	// Setup parent worktree directory
	parentDir := filepath.Join(env.RepoRoot, "work", parentBranch)
	require.NoError(t, os.MkdirAll(parentDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(parentDir, ".git"),
		[]byte("gitdir: ../../.git/worktrees/parent-branch\n"), 0o644,
	))

	// Setup child worktree inside parent's work/ directory
	childDir := filepath.Join(parentDir, "work", childBranch)
	require.NoError(t, os.MkdirAll(childDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(childDir, ".git"),
		[]byte("gitdir: ../../../../.git/worktrees/child-branch\n"), 0o644,
	))

	// Setup state files for both
	setupStateFile(t, env.RepoRoot, parentBranch, map[string]state.FeatureState{
		"feat": {Status: state.StatusActive, StartedAt: time.Now()},
	})
	setupStateFile(t, env.RepoRoot, childBranch, map[string]state.FeatureState{
		"feat": {Status: state.StatusActive, StartedAt: time.Now()},
	})

	err := env.Runner.Close(action.CloseOptions{
		Branch:      parentBranch,
		Force:       false,
		RepoRoot:    env.RepoRoot,
		ProjectName: "test",
		Yes:         true,
		Depth:       10,
	}, env.WM)
	require.NoError(t, err)

	// Verify command order: child close before parent close
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

	// Verify both state files are removed
	_, err1 := os.Stat(state.StatePath(env.RepoRoot, parentBranch))
	_, err2 := os.Stat(state.StatePath(env.RepoRoot, childBranch))
	assert.True(t, os.IsNotExist(err1), "parent state file should be deleted")
	assert.True(t, os.IsNotExist(err2), "child state file should be deleted")
}

func TestClose_DepthLimitStopsRecursion(t *testing.T) {
	env := newTestEnv(t)

	// Create 3-level nesting: a -> b -> c
	dirA := filepath.Join(env.RepoRoot, "work", "a")
	require.NoError(t, os.MkdirAll(dirA, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dirA, ".git"), []byte("gitdir: ../../.git/worktrees/a\n"), 0o644))

	dirB := filepath.Join(dirA, "work", "b")
	require.NoError(t, os.MkdirAll(dirB, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dirB, ".git"), []byte("gitdir: ../../../../.git/worktrees/b\n"), 0o644))

	dirC := filepath.Join(dirB, "work", "c")
	require.NoError(t, os.MkdirAll(dirC, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dirC, ".git"), []byte("gitdir: ../../../../../../.git/worktrees/c\n"), 0o644))

	err := env.Runner.Close(action.CloseOptions{
		Branch:      "a",
		Force:       false,
		RepoRoot:    env.RepoRoot,
		ProjectName: "test",
		Yes:         true,
		Depth:       1, // Only 1 level deep
	}, env.WM)
	require.NoError(t, err)

	recs := env.Recorder.Records()
	// b should be closed (depth=1 allows descending to b)
	assert.True(t, hasRecordContaining(recs, "branch") && hasRecordContaining(recs, "b"),
		"branch b should be closed at depth=1")
	// c should NOT be closed (depth=0 for b's children)
	hasCRemove := false
	for _, r := range recs {
		if strings.Contains(r.Command, "worktree remove") && strings.Contains(r.Command, "work"+string(filepath.Separator)+"c") {
			hasCRemove = true
		}
	}
	assert.False(t, hasCRemove, "worktree c should NOT be removed at depth limit, recs: %v", recs)
}

func TestClose_Force_RunsPrune(t *testing.T) {
	env := newTestEnv(t)
	branch := "test-branch"

	wtDir := filepath.Join(env.RepoRoot, "work", branch)
	require.NoError(t, os.MkdirAll(wtDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(wtDir, ".git"), []byte("gitdir: ../../.git/worktrees/test-branch\n"), 0o644))

	err := env.Runner.Close(action.CloseOptions{
		Branch:      branch,
		Force:       true,
		RepoRoot:    env.RepoRoot,
		ProjectName: "test",
		Yes:         true,
		Depth:       10,
	}, env.WM)
	require.NoError(t, err)

	recs := env.Recorder.Records()
	assert.True(t, hasRecordContaining(recs, "worktree prune"),
		"worktree prune should be called with --force, recs: %v", recs)
}

func TestClose_NoForce_SkipsPrune(t *testing.T) {
	env := newTestEnv(t)
	branch := "test-branch"

	wtDir := filepath.Join(env.RepoRoot, "work", branch)
	require.NoError(t, os.MkdirAll(wtDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(wtDir, ".git"), []byte("gitdir: ../../.git/worktrees/test-branch\n"), 0o644))

	err := env.Runner.Close(action.CloseOptions{
		Branch:      branch,
		Force:       false,
		RepoRoot:    env.RepoRoot,
		ProjectName: "test",
		Yes:         true,
		Depth:       10,
	}, env.WM)
	require.NoError(t, err)

	recs := env.Recorder.Records()
	assert.False(t, hasRecordContaining(recs, "worktree prune"),
		"worktree prune should NOT be called without --force, recs: %v", recs)
}

func TestClose_ConfirmYes_Executes(t *testing.T) {
	env := newTestEnv(t)
	branch := "test-branch"

	wtDir := filepath.Join(env.RepoRoot, "work", branch)
	require.NoError(t, os.MkdirAll(wtDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(wtDir, ".git"), []byte("gitdir: ../../.git/worktrees/test-branch\n"), 0o644))

	err := env.Runner.Close(action.CloseOptions{
		Branch:      branch,
		Force:       false,
		RepoRoot:    env.RepoRoot,
		ProjectName: "test",
		Yes:         false,
		Depth:       10,
		Stdin:       strings.NewReader("y\n"),
	}, env.WM)
	require.NoError(t, err)

	recs := env.Recorder.Records()
	assert.True(t, hasRecordContaining(recs, "worktree remove"),
		"worktree remove should be called after 'y' confirmation, recs: %v", recs)
}

func TestClose_ConfirmNo_Aborts(t *testing.T) {
	env := newTestEnv(t)
	branch := "test-branch"

	wtDir := filepath.Join(env.RepoRoot, "work", branch)
	require.NoError(t, os.MkdirAll(wtDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(wtDir, ".git"), []byte("gitdir: ../../.git/worktrees/test-branch\n"), 0o644))

	err := env.Runner.Close(action.CloseOptions{
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
		"worktree remove should NOT be called after 'N' confirmation, recs: %v", recs)
}

func TestClose_YesFlag_SkipsConfirmation(t *testing.T) {
	env := newTestEnv(t)
	branch := "test-branch"

	wtDir := filepath.Join(env.RepoRoot, "work", branch)
	require.NoError(t, os.MkdirAll(wtDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(wtDir, ".git"), []byte("gitdir: ../../.git/worktrees/test-branch\n"), 0o644))

	err := env.Runner.Close(action.CloseOptions{
		Branch:      branch,
		Force:       false,
		RepoRoot:    env.RepoRoot,
		ProjectName: "test",
		Yes:         true, // Skip confirmation
		Depth:       10,
		Stdin:       nil, // No stdin needed
	}, env.WM)
	require.NoError(t, err)

	recs := env.Recorder.Records()
	assert.True(t, hasRecordContaining(recs, "worktree remove"),
		"worktree remove should be called with --yes, recs: %v", recs)
}
