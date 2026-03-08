package codestatus_test

import (
	"testing"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/codestatus"
	"github.com/axsh/tokotachi/features/tt/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNeedsUpdate_Fresh(t *testing.T) {
	now := time.Now()
	checked := now.Add(-1 * time.Minute)
	states := map[string]state.StateFile{
		"feat-a": {
			Branch: "feat-a",
			CodeStatus: &state.CodeStatus{
				Status:        state.CodeStatusHosted,
				LastCheckedAt: &checked,
			},
		},
	}
	assert.False(t, codestatus.NeedsUpdate(states, now))
}

func TestNeedsUpdate_Stale(t *testing.T) {
	now := time.Now()
	checked := now.Add(-10 * time.Minute)
	states := map[string]state.StateFile{
		"feat-a": {
			Branch: "feat-a",
			CodeStatus: &state.CodeStatus{
				Status:        state.CodeStatusHosted,
				LastCheckedAt: &checked,
			},
		},
	}
	assert.True(t, codestatus.NeedsUpdate(states, now))
}

func TestNeedsUpdate_Nil(t *testing.T) {
	now := time.Now()
	states := map[string]state.StateFile{
		"feat-a": {
			Branch: "feat-a",
			// CodeStatus is nil
		},
	}
	assert.True(t, codestatus.NeedsUpdate(states, now))
}

func TestBranchesNeedingUpdate(t *testing.T) {
	now := time.Now()
	fresh := now.Add(-1 * time.Minute)
	stale := now.Add(-10 * time.Minute)
	states := map[string]state.StateFile{
		"feat-fresh": {
			Branch: "feat-fresh",
			CodeStatus: &state.CodeStatus{
				Status:        state.CodeStatusHosted,
				LastCheckedAt: &fresh,
			},
		},
		"feat-stale": {
			Branch: "feat-stale",
			CodeStatus: &state.CodeStatus{
				Status:        state.CodeStatusHosted,
				LastCheckedAt: &stale,
			},
		},
		"feat-nil": {
			Branch: "feat-nil",
		},
	}
	branches := codestatus.BranchesNeedingUpdate(states, now)
	assert.Len(t, branches, 2)
	assert.Contains(t, branches, "feat-stale")
	assert.Contains(t, branches, "feat-nil")
	assert.NotContains(t, branches, "feat-fresh")
}

func TestIsRunning_NoLock(t *testing.T) {
	dir := t.TempDir()
	assert.False(t, codestatus.IsRunning(dir))
}

func TestAcquireLock_ReleaseLock(t *testing.T) {
	dir := t.TempDir()

	err := codestatus.AcquireLock(dir)
	require.NoError(t, err)

	// Lock should be held
	assert.True(t, codestatus.IsRunning(dir))

	// Release lock
	codestatus.ReleaseLock(dir)
	assert.False(t, codestatus.IsRunning(dir))
}

func TestIsRunning_StaleLock(t *testing.T) {
	dir := t.TempDir()

	// Write a lock file with a non-existent PID
	lockPath := dir + "/work/.codestatus.lock"
	require.NoError(t, writeTestFile(lockPath, "999999999"))

	// Should detect stale lock and clean up
	assert.False(t, codestatus.IsRunning(dir))
}
