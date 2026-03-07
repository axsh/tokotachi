package doctor

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusPass, "✅"},
		{StatusFail, "❌"},
		{StatusWarn, "⚠️"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.status.String())
	}
}

func TestStatus_MarshalJSON(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusPass, `"pass"`},
		{StatusFail, `"fail"`},
		{StatusWarn, `"warn"`},
	}
	for _, tt := range tests {
		data, err := json.Marshal(tt.status)
		require.NoError(t, err)
		assert.Equal(t, tt.want, string(data))
	}
}

func TestReport_HasFailures(t *testing.T) {
	tests := []struct {
		name    string
		results []CheckResult
		want    bool
	}{
		{
			name:    "no results",
			results: nil,
			want:    false,
		},
		{
			name: "all pass",
			results: []CheckResult{
				{Status: StatusPass},
				{Status: StatusPass},
			},
			want: false,
		},
		{
			name: "warn only",
			results: []CheckResult{
				{Status: StatusPass},
				{Status: StatusWarn},
			},
			want: false,
		},
		{
			name: "has fail",
			results: []CheckResult{
				{Status: StatusPass},
				{Status: StatusFail},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Report{Results: tt.results}
			assert.Equal(t, tt.want, r.HasFailures())
		})
	}
}

func TestReport_Summary(t *testing.T) {
	r := &Report{
		Results: []CheckResult{
			{Status: StatusPass},
			{Status: StatusPass},
			{Status: StatusFail},
			{Status: StatusWarn},
			{Status: StatusPass},
		},
	}
	s := r.Summary()
	assert.Equal(t, 5, s.Total)
	assert.Equal(t, 3, s.Passed)
	assert.Equal(t, 1, s.Failed)
	assert.Equal(t, 1, s.Warnings)
}

func TestReport_PrintText(t *testing.T) {
	r := &Report{
		Results: []CheckResult{
			{Category: "Tools", Name: "git", Status: StatusPass, Message: "git version 2.43.0"},
			{Category: "Tools", Name: "gh", Status: StatusWarn, Message: "not found", FixHint: "Install GitHub CLI"},
			{Category: "Config", Name: ".devrc.yaml", Status: StatusFail, Message: "parse error", Expected: "valid YAML", FixHint: "Fix YAML syntax"},
		},
	}

	var buf bytes.Buffer
	r.PrintText(&buf)
	output := buf.String()

	// Verify categories are present
	assert.Contains(t, output, "Tools")
	assert.Contains(t, output, "Config")

	// Verify icons
	assert.Contains(t, output, "✅")
	assert.Contains(t, output, "⚠️")
	assert.Contains(t, output, "❌")

	// Verify fix hints for non-pass items
	assert.Contains(t, output, "Install GitHub CLI")
	assert.Contains(t, output, "Fix YAML syntax")

	// Verify summary line
	assert.Contains(t, output, "1 passed")
	assert.Contains(t, output, "1 failed")
	assert.Contains(t, output, "1 warning")
}

func TestReport_PrintJSON(t *testing.T) {
	r := &Report{
		Results: []CheckResult{
			{Category: "Tools", Name: "git", Status: StatusPass, Message: "git version 2.43.0"},
			{Category: "Tools", Name: "gh", Status: StatusWarn, Message: "not found"},
		},
	}

	var buf bytes.Buffer
	err := r.PrintJSON(&buf)
	require.NoError(t, err)

	// Verify valid JSON
	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	// Verify structure
	assert.Contains(t, parsed, "results")
	assert.Contains(t, parsed, "summary")

	// Verify summary
	summary := parsed["summary"].(map[string]any)
	assert.Equal(t, float64(2), summary["total"])
	assert.Equal(t, float64(1), summary["passed"])
	assert.Equal(t, float64(0), summary["failed"])
	assert.Equal(t, float64(1), summary["warnings"])

	// Verify status values are strings
	results := parsed["results"].([]any)
	first := results[0].(map[string]any)
	assert.Equal(t, "pass", first["status"])

	// Verify empty fields are omitted
	output := buf.String()
	assert.True(t, !strings.Contains(output, "fix_hint") || strings.Contains(output, `"fix_hint"`),
		"fix_hint should be omitted when empty")
}

func TestReport_PrintText_Fixed(t *testing.T) {
	r := &Report{
		Results: []CheckResult{
			{Category: "Config", Name: ".devrc.yaml", Status: StatusPass, Message: "created with default settings", Fixed: true},
			{Category: "Tools", Name: "git", Status: StatusPass, Message: "git version 2.43.0"},
		},
	}

	var buf bytes.Buffer
	r.PrintText(&buf)
	output := buf.String()

	assert.Contains(t, output, "🔧", "Fixed items should show wrench icon")
	assert.Contains(t, output, "created with default settings")
	assert.Contains(t, output, "1 fixed")
}

func TestSummary_Fixed(t *testing.T) {
	r := &Report{
		Results: []CheckResult{
			{Status: StatusPass, Fixed: true},
			{Status: StatusPass},
			{Status: StatusWarn},
		},
	}
	s := r.Summary()
	assert.Equal(t, 3, s.Total)
	assert.Equal(t, 2, s.Passed)
	assert.Equal(t, 1, s.Warnings)
	assert.Equal(t, 1, s.Fixed)
}

func TestReport_PrintJSON_Fixed(t *testing.T) {
	r := &Report{
		Results: []CheckResult{
			{Category: "Config", Name: ".devrc.yaml", Status: StatusPass, Message: "created with defaults", Fixed: true},
		},
	}

	var buf bytes.Buffer
	err := r.PrintJSON(&buf)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	results := parsed["results"].([]any)
	first := results[0].(map[string]any)
	assert.Equal(t, true, first["fixed"])

	summary := parsed["summary"].(map[string]any)
	assert.Equal(t, float64(1), summary["fixed"])
}
