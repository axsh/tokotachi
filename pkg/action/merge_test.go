package action_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/axsh/tokotachi/pkg/action"
	"github.com/axsh/tokotachi/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerge_Success_FFOnly(t *testing.T) {
	env := newTestEnv(t)
	branch := "feat-merge"

	setupStateFile(t, env.RepoRoot, branch, nil)
	// Set BaseBranch on the state file
	statePath := state.StatePath(env.RepoRoot, branch)
	sf, err := state.Load(statePath)
	require.NoError(t, err)
	sf.BaseBranch = "main"
	require.NoError(t, state.Save(statePath, sf))

	result, err := env.Runner.Merge(action.MergeOptions{
		Branch:   branch,
		RepoRoot: env.RepoRoot,
		Strategy: action.MergeStrategyFFOnly,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "main", result.BaseBranch)
	assert.Equal(t, action.MergeStrategyFFOnly, result.Strategy)

	recs := env.Recorder.Records()
	assert.True(t, hasRecordContaining(recs, "--ff-only"),
		"git merge --ff-only should be recorded, got: %v", recs)
	assert.True(t, hasRecordContaining(recs, branch),
		"branch name should be in merge command, got: %v", recs)
}

func TestMerge_Success_NoFF(t *testing.T) {
	env := newTestEnv(t)
	branch := "feat-noff"

	setupStateFile(t, env.RepoRoot, branch, nil)
	statePath := state.StatePath(env.RepoRoot, branch)
	sf, err := state.Load(statePath)
	require.NoError(t, err)
	sf.BaseBranch = "main"
	require.NoError(t, state.Save(statePath, sf))

	result, err := env.Runner.Merge(action.MergeOptions{
		Branch:   branch,
		RepoRoot: env.RepoRoot,
		Strategy: action.MergeStrategyNoFF,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	recs := env.Recorder.Records()
	assert.True(t, hasRecordContaining(recs, "--no-ff"),
		"git merge --no-ff should be recorded, got: %v", recs)
}

func TestMerge_Success_FF(t *testing.T) {
	env := newTestEnv(t)
	branch := "feat-ff"

	setupStateFile(t, env.RepoRoot, branch, nil)
	statePath := state.StatePath(env.RepoRoot, branch)
	sf, err := state.Load(statePath)
	require.NoError(t, err)
	sf.BaseBranch = "main"
	require.NoError(t, state.Save(statePath, sf))

	result, err := env.Runner.Merge(action.MergeOptions{
		Branch:   branch,
		RepoRoot: env.RepoRoot,
		Strategy: action.MergeStrategyFF,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	recs := env.Recorder.Records()
	// FF strategy should not have --ff-only or --no-ff flags
	assert.False(t, hasRecordContaining(recs, "--ff-only"),
		"should not have --ff-only for FF strategy, got: %v", recs)
	assert.False(t, hasRecordContaining(recs, "--no-ff"),
		"should not have --no-ff for FF strategy, got: %v", recs)
	assert.True(t, hasRecordContaining(recs, "merge"),
		"git merge should be recorded, got: %v", recs)
}

func TestMerge_BaseBranch_FromState(t *testing.T) {
	env := newTestEnv(t)
	branch := "feat-custom-base"

	setupStateFile(t, env.RepoRoot, branch, nil)
	statePath := state.StatePath(env.RepoRoot, branch)
	sf, err := state.Load(statePath)
	require.NoError(t, err)
	sf.BaseBranch = "develop"
	require.NoError(t, state.Save(statePath, sf))

	result, err := env.Runner.Merge(action.MergeOptions{
		Branch:   branch,
		RepoRoot: env.RepoRoot,
		Strategy: action.MergeStrategyFFOnly,
	})
	require.NoError(t, err)
	assert.Equal(t, "develop", result.BaseBranch,
		"should use BaseBranch from state file")
}

func TestMerge_BaseBranch_Fallback_Main(t *testing.T) {
	env := newTestEnv(t)
	branch := "feat-no-base"

	// Create state file without BaseBranch
	setupStateFile(t, env.RepoRoot, branch, nil)

	result, err := env.Runner.Merge(action.MergeOptions{
		Branch:   branch,
		RepoRoot: env.RepoRoot,
		Strategy: action.MergeStrategyFFOnly,
	})
	require.NoError(t, err)
	assert.Equal(t, "main", result.BaseBranch,
		"should fallback to 'main' when BaseBranch is empty")
}

func TestMerge_NoStateFile_Fallback_Main(t *testing.T) {
	env := newTestEnv(t)
	branch := "feat-no-state"

	// Do NOT create state file
	result, err := env.Runner.Merge(action.MergeOptions{
		Branch:   branch,
		RepoRoot: env.RepoRoot,
		Strategy: action.MergeStrategyFFOnly,
	})
	require.NoError(t, err)
	assert.Equal(t, "main", result.BaseBranch,
		"should fallback to 'main' when state file does not exist")
}

func TestMerge_DryRun(t *testing.T) {
	env := newTestEnv(t)
	branch := "feat-dryrun"

	setupStateFile(t, env.RepoRoot, branch, nil)
	statePath := state.StatePath(env.RepoRoot, branch)
	sf, err := state.Load(statePath)
	require.NoError(t, err)
	sf.BaseBranch = "main"
	require.NoError(t, state.Save(statePath, sf))

	// env is already in DryRun mode
	result, err := env.Runner.Merge(action.MergeOptions{
		Branch:   branch,
		RepoRoot: env.RepoRoot,
		Strategy: action.MergeStrategyFFOnly,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	// In dry-run, commands are recorded but not executed
	recs := env.Recorder.Records()
	assert.True(t, hasRecordContaining(recs, "merge"),
		"merge command should be recorded even in dry-run, got: %v", recs)
}

func TestMerge_BaseBranch_RoundTrip(t *testing.T) {
	// Verify BaseBranch persists correctly through state file operations
	dir := t.TempDir()
	path := filepath.Join(dir, "work", "feat-test.state.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))

	sf := state.StateFile{
		Branch:     "feat-test",
		BaseBranch: "develop",
		CreatedAt:  time.Now(),
	}
	require.NoError(t, state.Save(path, sf))

	loaded, err := state.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "develop", loaded.BaseBranch)
}
