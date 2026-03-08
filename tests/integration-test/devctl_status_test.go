package integration_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDevctlStatusWhenRunning verifies that 'devctl status' indicates a running container.
func TestDevctlStatusWhenRunning(t *testing.T) {
	requireDockerAvailable(t)

	// Setup: start a container
	upOut, upErr, upCode := runDevctl(t, "up", branchName, featureName)
	assert.Equal(t, 0, upCode, "devctl up failed during setup.\nSTDOUT:\n%s\nSTDERR:\n%s", upOut, upErr)

	// Execute: devctl status
	stdout, stderr, code := runDevctl(t, "status", branchName, featureName)
	assert.Equal(t, 0, code, "devctl status failed.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)

	// Verify: output contains "running"
	combined := strings.ToLower(stdout + stderr)
	assert.Contains(t, combined, "running",
		"Expected 'running' in devctl status output.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)

	// Cleanup
	cleanupDevctlDown(t)
}

// TestDevctlStatusWhenStopped verifies that 'devctl status' works when no container is running.
func TestDevctlStatusWhenStopped(t *testing.T) {
	requireDockerAvailable(t)

	// Ensure no container is running
	cleanupDevctlDown(t)

	// Execute: devctl status
	stdout, stderr, code := runDevctl(t, "status", branchName, featureName)
	assert.Equal(t, 0, code, "devctl status should not error when no container is running.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)
}
