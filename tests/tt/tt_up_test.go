package integration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTtUpStartsContainer verifies that 'tt up' starts a Docker container.
func TestTtUpStartsContainer(t *testing.T) {
	requireDockerAvailable(t)

	// Ensure clean state
	cleanupTTDown(t)
	ensureWorktree(t)

	// Execute: tt up
	stdout, stderr, code := runTT(t, "up", branchName, featureName, "--verbose")
	assert.Equal(t, 0, code, "tt up failed.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)

	// Verify: container is running
	psOut, err := dockerRun("ps", "--filter", "name="+featureName, "--format", "{{.Names}}")
	assert.NoError(t, err, "docker ps failed")
	assert.Contains(t, psOut, featureName, "Container not found in running containers.\ndocker ps output: %s", psOut)

	// Cleanup
	cleanupTTDown(t)
}

// TestTtUpIdempotent verifies that running 'tt up' twice does not cause errors.
func TestTtUpIdempotent(t *testing.T) {
	requireDockerAvailable(t)

	// Ensure clean state
	cleanupTTDown(t)
	ensureWorktree(t)

	// First up
	stdout1, stderr1, code1 := runTT(t, "up", branchName, featureName)
	assert.Equal(t, 0, code1, "First tt up failed.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout1, stderr1)

	// Second up (should be idempotent)
	stdout2, stderr2, code2 := runTT(t, "up", branchName, featureName)
	assert.Equal(t, 0, code2, "Second tt up should be idempotent.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout2, stderr2)

	// Cleanup
	cleanupTTDown(t)
}
