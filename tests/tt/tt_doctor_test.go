package integration_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTtDoctorBasic verifies that 'tt doctor' runs successfully
// and outputs the expected category headers.
func TestTtDoctorBasic(t *testing.T) {
	stdout, stderr, code := runTT(t, "doctor")
	combined := stdout + stderr

	assert.Equal(t, 0, code,
		"tt doctor should exit 0.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)

	assert.Contains(t, combined, "External Tools",
		"Output should contain External Tools category.\nOutput:\n%s", combined)
	assert.Contains(t, combined, "Repository Structure",
		"Output should contain Repository Structure category.\nOutput:\n%s", combined)
	assert.Contains(t, combined, "Feature: tt",
		"Output should contain Feature: tt.\nOutput:\n%s", combined)
}

// TestTtDoctorJSON verifies that 'tt doctor --json' produces valid JSON
// with the expected structure.
// Uses --feature tt to limit the scope.
func TestTtDoctorJSON(t *testing.T) {
	stdout, stderr, code := runTT(t, "doctor", "--feature", "tt", "--json")

	assert.Equal(t, 0, code,
		"tt doctor --json should exit 0.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)

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

// TestTtDoctorFeatureFilter verifies that '--feature tt' only checks
// the tt feature and not others.
func TestTtDoctorFeatureFilter(t *testing.T) {
	stdout, stderr, code := runTT(t, "doctor", "--feature", "tt")
	combined := stdout + stderr

	assert.Equal(t, 0, code,
		"tt doctor --feature tt should exit 0.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)

	assert.Contains(t, combined, "Feature: tt",
		"Output should contain Feature: tt.\nOutput:\n%s", combined)

	// Verify that other known features are NOT in the output
	assert.NotContains(t, combined, "Feature: integration-test",
		"Output should NOT contain integration-test feature.\nOutput:\n%s", combined)
}
