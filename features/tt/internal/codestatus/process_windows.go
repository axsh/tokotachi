//go:build windows

package codestatus

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// isProcessAlive checks whether a process with the given PID is still running.
// On Windows, uses `tasklist /FI "PID eq <pid>"` to reliably check.
func isProcessAlive(pid int) bool {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH", "/FO", "CSV")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	output := strings.TrimSpace(string(out))
	// tasklist outputs CSV lines; if PID is found, a line with the PID appears.
	return strings.Contains(output, strconv.Itoa(pid))
}

// detachSysProcAttr returns platform-specific SysProcAttr for detaching child process.
// On Windows, uses CREATE_NEW_PROCESS_GROUP to detach the child.
func detachSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: 0x00000200, // CREATE_NEW_PROCESS_GROUP
	}
}
