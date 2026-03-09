//go:build windows

package filelock

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// isProcessAlive checks whether a process with the given PID is still running.
// On Windows, uses tasklist to reliably check process existence.
func isProcessAlive(pid int) bool {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH", "/FO", "CSV")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	output := strings.TrimSpace(string(out))
	return strings.Contains(output, strconv.Itoa(pid))
}
