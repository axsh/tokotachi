// Package delegate runs registry *_cmd shell command lines.
package delegate

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Run executes cmdLine via `bash -lc` with the working directory set to workspace.
// It returns captured stdout. A non-zero exit status yields an error that includes stderr.
func Run(ctx context.Context, workspace string, cmdLine string) (stdout string, err error) {
	bash, lookErr := exec.LookPath("bash")
	if lookErr != nil {
		return "", fmt.Errorf("bash not found in PATH (on Windows install Git Bash): %w", lookErr)
	}
	cmd := exec.CommandContext(ctx, bash, "-lc", cmdLine)
	cmd.Dir = workspace
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	stdout = outBuf.String()
	if runErr != nil {
		msg := strings.TrimSpace(errBuf.String())
		if msg != "" {
			return stdout, fmt.Errorf("command failed: %s: %w: %s", cmdLine, runErr, msg)
		}
		return stdout, fmt.Errorf("command failed: %s: %w", cmdLine, runErr)
	}
	return stdout, nil
}
