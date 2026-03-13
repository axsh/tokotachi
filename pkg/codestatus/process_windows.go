//go:build windows

package codestatus

import (
	"syscall"
)

// detachSysProcAttr returns platform-specific SysProcAttr for detaching child process.
// On Windows, uses CREATE_NEW_PROCESS_GROUP | CREATE_NO_WINDOW to detach the child
// and prevent a console window from appearing.
func detachSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: 0x00000200 | 0x08000000, // CREATE_NEW_PROCESS_GROUP | CREATE_NO_WINDOW
	}
}
