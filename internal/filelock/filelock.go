package filelock

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Lock provides a cross-platform file lock using directory creation atomicity.
// os.Mkdir is atomic on all major operating systems, making it safe for
// inter-process locking without platform-specific APIs.
type Lock struct {
	dir string // absolute path to the lock directory
}

// Meta holds lock metadata stored inside the lock directory.
type Meta struct {
	PID       int       `json:"pid"`
	CreatedAt time.Time `json:"created_at"`
}

const metaFileName = "meta.json"

// New creates a new Lock targeting the given directory path.
// The directory will be created on TryLock and removed on Unlock.
func New(dir string) *Lock {
	return &Lock{dir: dir}
}

// TryLock attempts to acquire the lock.
// Returns (true, nil) if the lock was successfully acquired.
// Returns (false, nil) if the lock is already held by another process.
// Returns (false, err) on unexpected errors.
func (l *Lock) TryLock() (bool, error) {
	err := os.Mkdir(l.dir, 0o755)
	if err != nil {
		if os.IsExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to create lock directory: %w", err)
	}

	// Write metadata file inside the lock directory
	meta := Meta{
		PID:       os.Getpid(),
		CreatedAt: time.Now(),
	}
	data, err := json.Marshal(meta)
	if err != nil {
		// Clean up on failure
		_ = os.RemoveAll(l.dir)
		return false, fmt.Errorf("failed to marshal lock metadata: %w", err)
	}

	metaPath := filepath.Join(l.dir, metaFileName)
	if err := os.WriteFile(metaPath, data, 0o644); err != nil {
		_ = os.RemoveAll(l.dir)
		return false, fmt.Errorf("failed to write lock metadata: %w", err)
	}

	return true, nil
}

// Unlock releases the lock by removing the lock directory and its contents.
// Returns nil if the lock directory does not exist (idempotent).
func (l *Lock) Unlock() error {
	err := os.RemoveAll(l.dir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock directory: %w", err)
	}
	return nil
}

// IsLocked reports whether the lock directory exists.
func (l *Lock) IsLocked() bool {
	info, err := os.Stat(l.dir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ForceUnlockIfStale checks whether the lock is stale and forces an unlock if so.
// A lock is considered stale if:
//   - The metadata file is missing or corrupted.
//   - The CreatedAt + timeout has elapsed.
//   - The PID in metadata no longer refers to a running process.
//
// Returns (true, nil) if the lock was force-unlocked, (false, nil) if still valid.
func (l *Lock) ForceUnlockIfStale(timeout time.Duration) (bool, error) {
	if !l.IsLocked() {
		return false, nil
	}

	metaPath := filepath.Join(l.dir, metaFileName)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		// No metadata → stale, force unlock
		if unlockErr := l.Unlock(); unlockErr != nil {
			return false, unlockErr
		}
		return true, nil
	}

	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		// Corrupted metadata → stale, force unlock
		if unlockErr := l.Unlock(); unlockErr != nil {
			return false, unlockErr
		}
		return true, nil
	}

	// Check timeout
	if time.Since(meta.CreatedAt) >= timeout {
		if unlockErr := l.Unlock(); unlockErr != nil {
			return false, unlockErr
		}
		return true, nil
	}

	// Check if process is still alive
	if !isProcessAlive(meta.PID) {
		if unlockErr := l.Unlock(); unlockErr != nil {
			return false, unlockErr
		}
		return true, nil
	}

	return false, nil
}
