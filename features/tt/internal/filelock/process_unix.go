//go:build !windows

package filelock

import (
	"os"
	"syscall"
)

// isProcessAlive checks whether a process with the given PID is still running.
// On Unix, sends signal 0 to check process existence without killing it.
func isProcessAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	return err == nil
}
