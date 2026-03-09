package codestatus

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/filelock"
	"github.com/axsh/tokotachi/features/tt/internal/state"
)

const (
	// RefreshInterval is the minimum time between automatic updates.
	RefreshInterval = 5 * time.Minute

	// ProcessTimeout is the maximum time the background process may run.
	ProcessTimeout = 2 * time.Minute

	// LockDirName is the lock directory name in the work/ directory.
	LockDirName = ".codestatus.lock"
)

// NeedsUpdate checks if any branch needs a code status update.
// Returns true if any branch's LastCheckedAt is nil or older than RefreshInterval.
func NeedsUpdate(states map[string]state.StateFile, now time.Time) bool {
	for _, sf := range states {
		if sf.CodeStatus == nil || sf.CodeStatus.LastCheckedAt == nil {
			return true
		}
		if now.Sub(*sf.CodeStatus.LastCheckedAt) >= RefreshInterval {
			return true
		}
	}
	return false
}

// BranchesNeedingUpdate returns branch names whose code status is stale.
func BranchesNeedingUpdate(states map[string]state.StateFile, now time.Time) []string {
	var branches []string
	for branch, sf := range states {
		if sf.CodeStatus == nil || sf.CodeStatus.LastCheckedAt == nil {
			branches = append(branches, branch)
			continue
		}
		if now.Sub(*sf.CodeStatus.LastCheckedAt) >= RefreshInterval {
			branches = append(branches, branch)
		}
	}
	return branches
}

// lockPath returns the absolute path to the lock directory.
func lockPath(repoRoot string) string {
	return filepath.Join(repoRoot, "work", LockDirName)
}

// newLock creates a filelock.Lock for the code status lock path.
func newLock(repoRoot string) *filelock.Lock {
	return filelock.New(lockPath(repoRoot))
}

// IsRunning checks if the background updater is already running.
// If a stale lock is found (process not running or timed out), it is removed.
func IsRunning(repoRoot string) bool {
	lock := newLock(repoRoot)
	if !lock.IsLocked() {
		return false
	}
	// Try to clean up stale locks
	forced, _ := lock.ForceUnlockIfStale(ProcessTimeout)
	if forced {
		return false
	}
	return true
}

// AcquireLock creates the lock directory with metadata.
// Returns error if the lock is already held by a running process.
func AcquireLock(repoRoot string) error {
	lock := newLock(repoRoot)

	// Try to clean up stale locks first
	_, _ = lock.ForceUnlockIfStale(ProcessTimeout)

	// Ensure work/ directory exists
	if err := os.MkdirAll(filepath.Join(repoRoot, "work"), 0o755); err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}

	acquired, err := lock.TryLock()
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !acquired {
		return fmt.Errorf("lock already held")
	}
	return nil
}

// ReleaseLock removes the lock directory.
func ReleaseLock(repoRoot string) {
	_ = newLock(repoRoot).Unlock()
}

// StartBackgroundOptions configures background process startup.
type StartBackgroundOptions struct {
	// LogFile is the path to write stdout/stderr.
	// Empty string (default) means silent (nil output).
	// Used only for testing.
	LogFile string
}

// StartBackground starts the background updater process.
// Spawns `tt _update-code-status` as a detached child process.
// Returns immediately; does not wait for completion.
// Pass nil for opts to use default (silent) settings.
func StartBackground(repoRoot, ttBinary string, opts *StartBackgroundOptions) error {
	if IsRunning(repoRoot) {
		return nil // Already running
	}

	cmd := exec.Command(ttBinary, "_update-code-status", "--repo-root", repoRoot)
	cmd.Dir = repoRoot

	// Detach child process (platform-specific)
	cmd.SysProcAttr = detachSysProcAttr()

	// Set up output: silent by default, log file for testing
	cmd.Stdin = nil
	var logFileHandle *os.File
	if opts != nil && opts.LogFile != "" {
		f, err := os.OpenFile(opts.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err == nil {
			logFileHandle = f
			cmd.Stdout = f
			cmd.Stderr = f
		}
	}

	if err := cmd.Start(); err != nil {
		if logFileHandle != nil {
			_ = logFileHandle.Close()
		}
		return fmt.Errorf("failed to start background updater: %w", err)
	}

	// Do not wait; let the child run independently.
	// Log file handle is inherited by child process and closed on exit.
	return nil
}
