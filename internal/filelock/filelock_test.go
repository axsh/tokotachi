package filelock_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/axsh/tokotachi/internal/filelock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLock_TryLock_Success(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "test.lock")
	lock := filelock.New(dir)

	acquired, err := lock.TryLock()
	require.NoError(t, err)
	assert.True(t, acquired, "first TryLock should succeed")

	// Lock directory should exist
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir(), "lock path should be a directory")

	// meta.json should exist inside
	_, err = os.Stat(filepath.Join(dir, "meta.json"))
	assert.NoError(t, err, "meta.json should exist in lock directory")

	// Cleanup
	require.NoError(t, lock.Unlock())
}

func TestLock_TryLock_AlreadyLocked(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "test.lock")
	lock := filelock.New(dir)

	acquired, err := lock.TryLock()
	require.NoError(t, err)
	assert.True(t, acquired)

	// Second TryLock should fail
	acquired2, err := lock.TryLock()
	require.NoError(t, err)
	assert.False(t, acquired2, "second TryLock should fail when already locked")

	require.NoError(t, lock.Unlock())
}

func TestLock_Unlock(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "test.lock")
	lock := filelock.New(dir)

	acquired, err := lock.TryLock()
	require.NoError(t, err)
	require.True(t, acquired)

	err = lock.Unlock()
	require.NoError(t, err)

	// Lock directory should be removed
	_, err = os.Stat(dir)
	assert.True(t, os.IsNotExist(err), "lock directory should be removed after Unlock")
}

func TestLock_Unlock_ThenRelock(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "test.lock")
	lock := filelock.New(dir)

	// Lock, unlock, then re-lock
	acquired, err := lock.TryLock()
	require.NoError(t, err)
	require.True(t, acquired)

	require.NoError(t, lock.Unlock())

	// Re-lock should succeed
	acquired2, err := lock.TryLock()
	require.NoError(t, err)
	assert.True(t, acquired2, "TryLock should succeed after Unlock")

	require.NoError(t, lock.Unlock())
}

func TestLock_IsLocked(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "test.lock")
	lock := filelock.New(dir)

	assert.False(t, lock.IsLocked(), "should not be locked initially")

	acquired, err := lock.TryLock()
	require.NoError(t, err)
	require.True(t, acquired)
	assert.True(t, lock.IsLocked(), "should be locked after TryLock")

	require.NoError(t, lock.Unlock())
	assert.False(t, lock.IsLocked(), "should not be locked after Unlock")
}

func TestLock_ForceUnlockIfStale_Timeout(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "test.lock")
	lock := filelock.New(dir)

	// Acquire lock
	acquired, err := lock.TryLock()
	require.NoError(t, err)
	require.True(t, acquired)

	// Overwrite meta.json with old timestamp to simulate stale lock
	oldMeta := `{"pid": 999999999, "created_at": "2020-01-01T00:00:00Z"}`
	err = os.WriteFile(filepath.Join(dir, "meta.json"), []byte(oldMeta), 0o644)
	require.NoError(t, err)

	// ForceUnlockIfStale with short timeout should detect stale lock
	forced, err := lock.ForceUnlockIfStale(1 * time.Second)
	require.NoError(t, err)
	assert.True(t, forced, "stale lock should be force-unlocked")

	// Lock directory should be gone
	assert.False(t, lock.IsLocked())
}

func TestLock_ForceUnlockIfStale_InvalidMeta(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "test.lock")

	// Manually create lock directory with invalid meta
	require.NoError(t, os.Mkdir(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "meta.json"), []byte("invalid json"), 0o644))

	lock := filelock.New(dir)
	forced, err := lock.ForceUnlockIfStale(1 * time.Second)
	require.NoError(t, err)
	assert.True(t, forced, "lock with invalid meta should be force-unlocked")

	assert.False(t, lock.IsLocked())
}

func TestLock_ForceUnlockIfStale_NoMeta(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "test.lock")

	// Manually create empty lock directory (no meta.json)
	require.NoError(t, os.Mkdir(dir, 0o755))

	lock := filelock.New(dir)
	forced, err := lock.ForceUnlockIfStale(1 * time.Second)
	require.NoError(t, err)
	assert.True(t, forced, "lock without meta should be force-unlocked")

	assert.False(t, lock.IsLocked())
}

func TestLock_ForceUnlockIfStale_NotStale(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "test.lock")
	lock := filelock.New(dir)

	// Acquire lock normally (PID is current process, timestamp is now)
	acquired, err := lock.TryLock()
	require.NoError(t, err)
	require.True(t, acquired)

	// Should not force-unlock since lock is fresh and PID is alive
	forced, err := lock.ForceUnlockIfStale(5 * time.Minute)
	require.NoError(t, err)
	assert.False(t, forced, "fresh lock should not be force-unlocked")

	assert.True(t, lock.IsLocked(), "lock should still be held")
	require.NoError(t, lock.Unlock())
}

func TestLock_ConcurrentAccess(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "test.lock")

	const numGoroutines = 20
	var mu sync.Mutex
	successCount := 0
	var wg sync.WaitGroup

	wg.Add(numGoroutines)
	for range numGoroutines {
		go func() {
			defer wg.Done()
			lock := filelock.New(dir)
			acquired, err := lock.TryLock()
			if err != nil {
				return
			}
			if acquired {
				mu.Lock()
				successCount++
				mu.Unlock()
				// Hold lock briefly
				time.Sleep(10 * time.Millisecond)
				_ = lock.Unlock()
			}
		}()
	}
	wg.Wait()

	// At least one goroutine should have acquired the lock,
	// but due to sequential acquire-release, multiple may succeed.
	assert.True(t, successCount >= 1, "at least one goroutine should acquire the lock")
}

func TestLock_Unlock_NotLocked(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "test.lock")
	lock := filelock.New(dir)

	// Unlocking when not locked should not error
	err := lock.Unlock()
	assert.NoError(t, err, "Unlock on non-existent lock should not error")
}
