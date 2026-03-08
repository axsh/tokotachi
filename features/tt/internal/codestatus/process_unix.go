//go:build !windows

package codestatus

import (
	"os"
	"syscall"
)

// isProcessAlive checks whether a process with the given PID is still running.
// On Unix, sends signal 0 to check process existence.
func isProcessAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 does not kill but checks if process exists.
	// Returns nil if process exists and caller has permission to signal it.
	err = p.Signal(syscall.Signal(0))
	return err == nil
}

// detachSysProcAttr returns platform-specific SysProcAttr for detaching child process.
// On Unix, creates a new session to detach from parent.
func detachSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true,
	}
}
