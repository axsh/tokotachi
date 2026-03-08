package integration_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDevctlDoctorBasic verifies that 'devctl doctor' runs successfully
// and outputs the expected category headers.
func TestDevctlDoctorBasic(t *testing.T) {
	stdout, stderr, code := runDevctl(t, "doctor")
	combined := stdout + stderr

	assert.Equal(t, 0, code,
		"devctl doctor should exit 0.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)

	assert.Contains(t, combined, "External Tools",
		"Output should contain External Tools category.\nOutput:\n%s", combined)
	assert.Contains(t, combined, "Repository Structure",
		"Output should contain Repository Structure category.\nOutput:\n%s", combined)
	assert.Contains(t, combined, "Global Config",
		"Output should contain Global Config category.\nOutput:\n%s", combined)
	assert.Contains(t, combined, "Feature: devctl",
		"Output should contain Feature: devctl.\nOutput:\n%s", combined)
}

// TestDevctlDoctorJSON verifies that 'devctl doctor --json' produces valid JSON
// with the expected structure.
// Uses --feature devctl to limit the scope.
func TestDevctlDoctorJSON(t *testing.T) {
	stdout, stderr, code := runDevctl(t, "doctor", "--feature", "devctl", "--json")

	assert.Equal(t, 0, code,
		"devctl doctor --json should exit 0.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)

	// Parse the JSON output (may be in stdout or stderr depending on output routing)
	jsonOutput := stdout
	if jsonOutput == "" {
		jsonOutput = stderr
	}

	var parsed map[string]any
	err := json.Unmarshal([]byte(strings.TrimSpace(jsonOutput)), &parsed)
	require.NoError(t, err,
		"Output should be valid JSON.\nRaw output:\n%s", jsonOutput)

	assert.Contains(t, parsed, "results",
		"JSON output should contain 'results' key")
	assert.Contains(t, parsed, "summary",
		"JSON output should contain 'summary' key")

	// Verify summary structure
	summary, ok := parsed["summary"].(map[string]any)
	require.True(t, ok, "summary should be an object")
	assert.Contains(t, summary, "total")
	assert.Contains(t, summary, "passed")
	assert.Contains(t, summary, "failed")
	assert.Contains(t, summary, "warnings")
}

// TestDevctlDoctorFeatureFilter verifies that '--feature devctl' only checks
// the devctl feature and not others.
func TestDevctlDoctorFeatureFilter(t *testing.T) {
	stdout, stderr, code := runDevctl(t, "doctor", "--feature", "devctl")
	combined := stdout + stderr

	assert.Equal(t, 0, code,
		"devctl doctor --feature devctl should exit 0.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)

	assert.Contains(t, combined, "Feature: devctl",
		"Output should contain Feature: devctl.\nOutput:\n%s", combined)

	// Verify that other known features are NOT in the output
	assert.NotContains(t, combined, "Feature: integration-test",
		"Output should NOT contain integration-test feature.\nOutput:\n%s", combined)
}
