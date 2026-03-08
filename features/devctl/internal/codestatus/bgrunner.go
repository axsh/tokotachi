package codestatus

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/axsh/tokotachi/features/devctl/internal/state"
)

const (
	// RefreshInterval is the minimum time between automatic updates.
	RefreshInterval = 5 * time.Minute

	// ProcessTimeout is the maximum time the background process may run.
	ProcessTimeout = 2 * time.Minute

	// LockFileName is the lock file name in the work/ directory.
	LockFileName = ".codestatus.lock"
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

// lockPath returns the absolute path to the lock file.
func lockPath(repoRoot string) string {
	return filepath.Join(repoRoot, "work", LockFileName)
}

// IsRunning checks if the background updater is already running.
// If a stale lock file is found (process not running), it is removed.
func IsRunning(repoRoot string) bool {
	path := lockPath(repoRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		// Corrupted lock file, remove it
		_ = os.Remove(path)
		return false
	}

	if !isProcessAlive(pid) {
		_ = os.Remove(path)
		return false
	}

	return true
}

// AcquireLock creates the lock file with the current PID.
// Returns error if the lock is already held by a running process.
func AcquireLock(repoRoot string) error {
	if IsRunning(repoRoot) {
		return fmt.Errorf("lock already held")
	}

	path := lockPath(repoRoot)
	// Ensure work/ directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create lock directory: %w", err)
	}

	pid := os.Getpid()
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0o644)
}

// ReleaseLock removes the lock file.
func ReleaseLock(repoRoot string) {
	_ = os.Remove(lockPath(repoRoot))
}

// StartBackground starts the background updater process.
// Spawns `devctl _update-code-status` as a detached child process.
// Returns immediately; does not wait for completion.
func StartBackground(repoRoot, devctlBinary string) error {
	if IsRunning(repoRoot) {
		return nil // Already running
	}

	cmd := exec.Command(devctlBinary, "_update-code-status", "--repo-root", repoRoot)
	cmd.Dir = repoRoot

	// Detach from parent process
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// On Windows, CREATE_NEW_PROCESS_GROUP detaches the child.
		CreationFlags: 0x00000200, // CREATE_NEW_PROCESS_GROUP
	}

	// Redirect to null to avoid holding parent's handles
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start background updater: %w", err)
	}

	// Do not wait; let the child run independently
	return nil
}
