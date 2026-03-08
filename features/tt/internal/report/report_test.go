package report_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/cmdexec"
	"github.com/axsh/tokotachi/features/tt/internal/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleReport() *report.Report {
	return &report.Report{
		StartTime:     time.Date(2026, 3, 7, 14, 25, 0, 0, time.UTC),
		Feature:       "tt",
		Branch:        "test-001",
		OS:            "windows",
		Editor:        "cursor",
		ContainerMode: "docker-local",
		EnvVars: []report.EnvVar{
			{Name: "TT_EDITOR", Value: "", Default: "cursor", WasSet: false},
			{Name: "TT_CMD_CURSOR", Value: "/custom/cursor", Default: "cursor", WasSet: true},
		},
		ShowEnvVars: true,
		Steps: []report.StepEntry{
			{Name: "Container up", Record: &cmdexec.ExecRecord{Command: "docker run ...", Success: true, ExitCode: 0}, Success: true},
			{Name: "Editor open", Record: &cmdexec.ExecRecord{Command: "cursor ./work", Success: true, ExitCode: 0}, Success: true},
		},
		OverallResult: "SUCCESS",
	}
}

func TestReport_Print(t *testing.T) {
	var buf bytes.Buffer
	r := sampleReport()
	r.Print(&buf)
	out := buf.String()
	assert.Contains(t, out, "tt")
	assert.Contains(t, out, "test-001")
	assert.Contains(t, out, "Container up")
	assert.Contains(t, out, "SUCCESS")
}

func TestReport_EnvVars(t *testing.T) {
	var buf bytes.Buffer
	r := sampleReport()
	r.Print(&buf)
	out := buf.String()
	assert.Contains(t, out, "TT_EDITOR")
	assert.Contains(t, out, "not set")
	assert.Contains(t, out, "TT_CMD_CURSOR")
	assert.Contains(t, out, "/custom/cursor")
}

func TestReport_EnvVars_Hidden(t *testing.T) {
	var buf bytes.Buffer
	r := &report.Report{
		StartTime:     time.Date(2026, 3, 7, 14, 25, 0, 0, time.UTC),
		Feature:       "tt",
		Branch:        "test-001",
		EnvVars: []report.EnvVar{
			{Name: "TT_EDITOR", Value: "", Default: "cursor", WasSet: false},
		},
		ShowEnvVars:   false,
		OverallResult: "SUCCESS",
	}
	r.Print(&buf)
	out := buf.String()
	assert.NotContains(t, out, "Environment Variables",
		"env vars section should be hidden when ShowEnvVars is false")
	assert.NotContains(t, out, "TT_EDITOR",
		"individual env var names should not appear")
}

func TestReport_WriteMarkdown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.md")
	r := sampleReport()
	err := r.WriteMarkdown(path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "# tt Execution Report")
	assert.Contains(t, content, "SUCCESS")
}

func TestReport_EmptyRecords(t *testing.T) {
	var buf bytes.Buffer
	r := &report.Report{
		StartTime:     time.Now(),
		Feature:       "empty",
		Branch:        "main",
		OverallResult: "SUCCESS",
	}
	r.Print(&buf)
	assert.Contains(t, buf.String(), "empty")
}
