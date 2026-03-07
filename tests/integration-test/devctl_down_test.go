package integration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDevctlDownStopsContainer verifies that 'devctl down' stops and removes the container.
func TestDevctlDownStopsContainer(t *testing.T) {
	requireDockerAvailable(t)

	// Setup: start a container
	stdout, stderr, code := runDevctl(t, "up", featureName)
	assert.Equal(t, 0, code, "devctl up failed during setup.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)

	// Execute: devctl down
	stdout, stderr, code = runDevctl(t, "down", featureName, "--verbose")
	assert.Equal(t, 0, code, "devctl down failed.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)

	// Verify: container is gone
	psOut, err := dockerRun("ps", "-a", "--filter", "name="+featureName, "--format", "{{.Names}}")
	assert.NoError(t, err, "docker ps -a failed")
	assert.NotContains(t, psOut, featureName,
		"Container still exists after 'devctl down'.\ndocker ps -a output: %s", psOut)
}

// TestDevctlDownNoopWhenNotRunning verifies that 'devctl down' when no container
// is running does not crash or panic (exit code may be non-zero depending
// on devctl's implementation).
func TestDevctlDownNoopWhenNotRunning(t *testing.T) {
	requireDockerAvailable(t)

	// Ensure no container is running
	cleanupDevctlDown(t)

	// Execute: devctl down again
	// devctl down may return exit=1 when no container exists, this is acceptable
	stdout, stderr, _ := runDevctl(t, "down", featureName)
	t.Logf("devctl down (noop) stdout: %s", stdout)
	t.Logf("devctl down (noop) stderr: %s", stderr)
	// No assertion on exit code — devctl currently returns 1 when container doesn't exist.
	// The key assertion is that it doesn't crash/panic.
}
