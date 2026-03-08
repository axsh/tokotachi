package integration_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTtStatusWhenRunning verifies that 'tt status' indicates a running container.
func TestTtStatusWhenRunning(t *testing.T) {
	requireDockerAvailable(t)

	// Setup: start a container
	ensureWorktree(t)
	upOut, upErr, upCode := runTT(t, "up", branchName, featureName)
	assert.Equal(t, 0, upCode, "tt up failed during setup.\nSTDOUT:\n%s\nSTDERR:\n%s", upOut, upErr)

	// Execute: tt status
	stdout, stderr, code := runTT(t, "status", branchName, featureName)
	assert.Equal(t, 0, code, "tt status failed.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)

	// Verify: output contains "running"
	combined := strings.ToLower(stdout + stderr)
	assert.Contains(t, combined, "running",
		"Expected 'running' in tt status output.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)

	// Cleanup
	cleanupTTDown(t)
}

// TestTtStatusWhenStopped verifies that 'tt status' works when no container is running.
func TestTtStatusWhenStopped(t *testing.T) {
	requireDockerAvailable(t)

	// Ensure no container is running
	cleanupTTDown(t)

	// Execute: tt status
	stdout, stderr, code := runTT(t, "status", branchName, featureName)
	assert.Equal(t, 0, code, "tt status should not error when no container is running.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)
}
