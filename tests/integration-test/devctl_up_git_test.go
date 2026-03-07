package integration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDevctlUpGitWorktree verifies that git commands work inside the container
// when the worktree is mounted (git worktree setup should configure paths).
func TestDevctlUpGitWorktree(t *testing.T) {
	requireDockerAvailable(t)

	// Ensure clean state
	cleanupDevctlDown(t)

	// Start container
	stdout, stderr, code := runDevctl(t, "up", featureName, "--verbose")
	assert.Equal(t, 0, code, "devctl up failed.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)

	// Verify git status works inside the container
	containerName := "devctl-" + featureName
	gitOut, err := dockerRun("exec", containerName, "git", "status")
	assert.NoError(t, err, "git status failed inside container.\nOutput: %s", gitOut)
	assert.NotContains(t, gitOut, "fatal:", "git status returned error: %s", gitOut)

	// Verify git log works (can read commit history from main .git)
	logOut, err := dockerRun("exec", containerName, "git", "log", "--oneline", "-1")
	assert.NoError(t, err, "git log failed inside container.\nOutput: %s", logOut)
	assert.NotEmpty(t, logOut, "git log returned empty output")

	// Cleanup
	cleanupDevctlDown(t)
}
