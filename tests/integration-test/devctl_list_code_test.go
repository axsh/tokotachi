package integration_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDevctlListCode_ColumnHeaders(t *testing.T) {
	stdout, _, exitCode := runDevctl(t, "list")
	if exitCode != 0 {
		t.Fatalf("devctl list exited with code %d", exitCode)
	}

	// Verify new column headers
	assert.Contains(t, stdout, "CONTAINER", "Output should contain CONTAINER header")
	assert.Contains(t, stdout, "CODE", "Output should contain CODE header")
	assert.Contains(t, stdout, "BRANCH", "Output should contain BRANCH header")
	assert.Contains(t, stdout, "FEATURE", "Output should contain FEATURE header")

	// Verify old STATE header is gone
	lines := strings.Split(stdout, "\n")
	if len(lines) > 0 {
		header := lines[0]
		// STATE should NOT appear as a standalone column header
		// (CONTAINER contains no "STATE" substring)
		assert.NotContains(t, header, "STATE", "Header should not contain STATE")
	}
}

func TestDevctlListCode_NoPathByDefault(t *testing.T) {
	stdout, _, exitCode := runDevctl(t, "list")
	if exitCode != 0 {
		t.Fatalf("devctl list exited with code %d", exitCode)
	}

	lines := strings.Split(stdout, "\n")
	if len(lines) > 0 {
		header := lines[0]
		assert.NotContains(t, header, "PATH", "PATH should not appear without --path flag")
	}
}

func TestDevctlListCode_WithPathFlag(t *testing.T) {
	stdout, _, exitCode := runDevctl(t, "list", "--path")
	if exitCode != 0 {
		t.Fatalf("devctl list --path exited with code %d", exitCode)
	}

	assert.Contains(t, stdout, "PATH", "Output should contain PATH header with --path flag")
}

func TestDevctlListCode_UpdateFlagAccepted(t *testing.T) {
	// Just verify --update flag is accepted (exit code 0)
	_, _, exitCode := runDevctl(t, "list", "--update")
	// This may fail if no git remote is set up, but the flag should be accepted
	// We mainly verify the flag doesn't cause a "unknown flag" error
	assert.Equal(t, 0, exitCode, "devctl list --update should exit successfully")
}

func TestDevctlListCode_JSONOutput(t *testing.T) {
	stdout, _, exitCode := runDevctl(t, "list", "--json")
	if exitCode != 0 {
		t.Fatalf("devctl list --json exited with code %d", exitCode)
	}

	// JSON output should not contain table headers
	assert.NotContains(t, stdout, "CONTAINER")
	assert.NotContains(t, stdout, "CODE")
	// Should be valid JSON (starts with [ or {)
	trimmed := strings.TrimSpace(stdout)
	assert.True(t, strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{"),
		"JSON output should start with [ or {, got: %s", trimmed[:min(50, len(trimmed))])
}
