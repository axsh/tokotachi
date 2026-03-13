package codestatus_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/axsh/tokotachi/pkg/codestatus"
	"github.com/axsh/tokotachi/pkg/state"
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

	// Ensure work/ subdirectory exists
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "work"), 0o755))

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
	lockDir := filepath.Join(dir, "work", ".codestatus.lock")

	// Create lock directory structure manually with stale metadata
	require.NoError(t, os.MkdirAll(lockDir, 0o755))
	staleMeta := map[string]any{
		"pid":        999999999,
		"created_at": "2020-01-01T00:00:00Z",
	}
	data, err := json.Marshal(staleMeta)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(lockDir, "meta.json"), data, 0o644))

	// Should detect stale lock and clean up
	assert.False(t, codestatus.IsRunning(dir))
}

func TestStartBackground_AlreadyRunning(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "work"), 0o755))

	// Acquire lock to simulate running process
	require.NoError(t, codestatus.AcquireLock(dir))
	defer codestatus.ReleaseLock(dir)

	// StartBackground should return nil (already running)
	err := codestatus.StartBackground(dir, "nonexistent-binary", nil)
	assert.NoError(t, err, "StartBackground should return nil when already running")
}

func TestStartBackground_InvalidBinary(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "work"), 0o755))

	// StartBackground with non-existent binary should return error
	err := codestatus.StartBackground(dir, "/nonexistent/binary/path", nil)
	assert.Error(t, err, "StartBackground should error with invalid binary")
}

func TestStartBackground_WithLogFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "work"), 0o755))

	logFile := filepath.Join(dir, "test.log")
	opts := &codestatus.StartBackgroundOptions{
		LogFile: logFile,
	}

	// With a non-existent binary, the command will fail to start
	err := codestatus.StartBackground(dir, "/nonexistent/binary/path", opts)
	assert.Error(t, err, "should fail with invalid binary even with log opts")
}
