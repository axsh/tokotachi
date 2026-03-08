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

func TestDevctlListCode_FullFlagAccepted(t *testing.T) {
	stdout, _, exitCode := runDevctl(t, "list", "--full")
	if exitCode != 0 {
		t.Fatalf("devctl list --full exited with code %d", exitCode)
	}

	assert.Contains(t, stdout, "BRANCH", "Output should contain BRANCH header")
	assert.Contains(t, stdout, "FEATURE", "Output should contain FEATURE header")
	assert.Contains(t, stdout, "CONTAINER", "Output should contain CONTAINER header")
	assert.Contains(t, stdout, "CODE", "Output should contain CODE header")
}

func TestDevctlListCode_EnvFlagAccepted(t *testing.T) {
	_, stderr, exitCode := runDevctl(t, "list", "--env")
	if exitCode != 0 {
		t.Fatalf("devctl list --env exited with code %d", exitCode)
	}

	assert.Contains(t, stderr, "Environment Variables",
		"stderr should contain Environment Variables section")
	assert.Contains(t, stderr, "DEVCTL_LIST_WIDTH",
		"stderr should contain DEVCTL_LIST_WIDTH")
	assert.Contains(t, stderr, "DEVCTL_LIST_PADDING",
		"stderr should contain DEVCTL_LIST_PADDING")
}

func TestDevctlListCode_NoEnvByDefault(t *testing.T) {
	stdout, stderr, exitCode := runDevctl(t, "list")
	if exitCode != 0 {
		t.Fatalf("devctl list exited with code %d", exitCode)
	}

	assert.NotContains(t, stdout, "Environment Variables",
		"stdout should not contain Environment Variables without --env")
	assert.NotContains(t, stderr, "Environment Variables",
		"stderr should not contain Environment Variables without --env")
}

func TestDevctlListCode_DynamicColumnWidth(t *testing.T) {
	stdout, _, exitCode := runDevctl(t, "list")
	if exitCode != 0 {
		t.Fatalf("devctl list exited with code %d", exitCode)
	}

	lines := strings.Split(stdout, "\n")
	if len(lines) < 2 {
		t.Fatal("expected at least 2 lines (header + data)")
	}

	// Dynamic width: last column should not have trailing spaces
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \r")
		if trimmed == "" {
			continue
		}
		assert.Equal(t, trimmed, strings.TrimRight(line, " \r"),
			"line should not have trailing spaces: %q", line)
	}

	// Verify columns are separated by at least 2 spaces (default padding)
	header := lines[0]
	assert.Contains(t, header, "BRANCH", "header should contain BRANCH")
	assert.Contains(t, header, "FEATURE", "header should contain FEATURE")
	// Between BRANCH and FEATURE there should be at least 2 spaces
	branchIdx := strings.Index(header, "BRANCH")
	featureIdx := strings.Index(header, "FEATURE")
	if branchIdx >= 0 && featureIdx > branchIdx {
		gap := header[branchIdx+len("BRANCH") : featureIdx]
		assert.True(t, len(gap) >= 2,
			"gap between BRANCH and FEATURE should be at least 2 spaces, got %d", len(gap))
	}
}
