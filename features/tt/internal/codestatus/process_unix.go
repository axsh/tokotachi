//go:build !windows

package codestatus

import (
	"syscall"
)

// detachSysProcAttr returns platform-specific SysProcAttr for detaching child process.
// On Unix, creates a new session to detach from parent.
func detachSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true,
	}
}
