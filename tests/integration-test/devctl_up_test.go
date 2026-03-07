package integration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDevctlUpStartsContainer verifies that 'devctl up' starts a Docker container.
func TestDevctlUpStartsContainer(t *testing.T) {
	requireDockerAvailable(t)

	// Ensure clean state
	cleanupDevctlDown(t)

	// Execute: devctl up
	stdout, stderr, code := runDevctl(t, "up", featureName, "--verbose")
	assert.Equal(t, 0, code, "devctl up failed.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)

	// Verify: container is running
	psOut, err := dockerRun("ps", "--filter", "name="+featureName, "--format", "{{.Names}}")
	assert.NoError(t, err, "docker ps failed")
	assert.Contains(t, psOut, featureName, "Container not found in running containers.\ndocker ps output: %s", psOut)

	// Cleanup
	cleanupDevctlDown(t)
}

// TestDevctlUpIdempotent verifies that running 'devctl up' twice does not cause errors.
func TestDevctlUpIdempotent(t *testing.T) {
	requireDockerAvailable(t)

	// Ensure clean state
	cleanupDevctlDown(t)

	// First up
	stdout1, stderr1, code1 := runDevctl(t, "up", featureName)
	assert.Equal(t, 0, code1, "First devctl up failed.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout1, stderr1)

	// Second up (should be idempotent)
	stdout2, stderr2, code2 := runDevctl(t, "up", featureName)
	assert.Equal(t, 0, code2, "Second devctl up should be idempotent.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout2, stderr2)

	// Cleanup
	cleanupDevctlDown(t)
}
