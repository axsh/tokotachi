package integration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDevctlOpen_NoEnvByDefault verifies that the --env flag is not shown
// in the report output by default when running open with --dry-run.
// Note: open may fail (exit code 1) because the test environment may not have
// the worktree set up, but the report is always printed regardless.
func TestDevctlOpen_NoEnvByDefault(t *testing.T) {
	stdout, _, _ := runDevctl(t, "open", "--dry-run", "fix-env-option")

	// Even if the command fails, it still prints the report.
	// Verify that the report does NOT contain Environment Variables.
	assert.NotContains(t, stdout, "Environment Variables",
		"stdout should not contain Environment Variables section without --env")
}

// TestDevctlOpen_WithEnvFlag verifies that --env shows the Environment Variables
// section in the report output.
func TestDevctlOpen_WithEnvFlag(t *testing.T) {
	stdout, _, _ := runDevctl(t, "open", "--env", "--dry-run", "fix-env-option")

	// Even if the command fails, it still prints the report.
	// Verify that the report DOES contain Environment Variables.
	assert.Contains(t, stdout, "Environment Variables",
		"stdout should contain Environment Variables section with --env")
	assert.Contains(t, stdout, "DEVCTL_EDITOR",
		"stdout should contain DEVCTL_EDITOR env var")
}
