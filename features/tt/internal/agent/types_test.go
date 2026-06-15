package agent

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExitCodeConstants(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected int
	}{
		{"ExitOK", ExitOK, 0},
		{"ExitJSONParseError", ExitJSONParseError, 10},
		{"ExitSchemaValidationError", ExitSchemaValidationError, 11},
		{"ExitUnsupportedVersion", ExitUnsupportedVersion, 12},
		{"ExitAgentIDInvalid", ExitAgentIDInvalid, 20},
		{"ExitStorageLockTimeout", ExitStorageLockTimeout, 30},
		{"ExitStorageWriteFailed", ExitStorageWriteFailed, 31},
		{"ExitPermissionDenied", ExitPermissionDenied, 40},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.code)
		})
	}
}

func TestResultCodeConstants(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{"CodeOK", CodeOK, "OK"},
		{"CodeIndexDegraded", CodeIndexDegraded, "INDEX_DEGRADED"},
		{"CodeNoGitRepository", CodeNoGitRepository, "NO_GIT_REPOSITORY"},
		{"CodeJSONParseError", CodeJSONParseError, "JSON_PARSE_ERROR"},
		{"CodeSchemaValidationError", CodeSchemaValidationError, "SCHEMA_VALIDATION_ERROR"},
		{"CodeUnsupportedVersion", CodeUnsupportedVersion, "UNSUPPORTED_SCHEMA_VERSION"},
		{"CodeAgentIDInvalid", CodeAgentIDInvalid, "AGENT_ID_INVALID"},
		{"CodeStorageLockTimeout", CodeStorageLockTimeout, "STORAGE_LOCK_TIMEOUT"},
		{"CodeStorageWriteFailed", CodeStorageWriteFailed, "STORAGE_WRITE_FAILED"},
		{"CodePermissionDenied", CodePermissionDenied, "PERMISSION_DENIED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.code)
		})
	}
}

func TestNotifyPayloadJSONRoundTrip(t *testing.T) {
	original := NotifyPayload{
		Version:    1,
		SourceType: "coding_agent",
		Agent:      "antigravity",
		TaskSummary: "Implement auth middleware",
		RawNotes:   []string{"Added JWT validation", "Updated config schema"},
		ChangedPaths: []string{"internal/auth/middleware.go"},
		Flags: &Flags{
			ArchitectureImpact: true,
			MemoryRelated:      false,
			DesignPattern:      true,
			Convention:         false,
			LessonLearned:      true,
			Preference:         false,
		},
		ClientRequestID: "req-001",
		Context: &PayloadContext{
			SessionID:      "sess-abc",
			WrapperVersion: "1.0.0",
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded NotifyPayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Version, decoded.Version)
	assert.Equal(t, original.Agent, decoded.Agent)
	assert.Equal(t, original.TaskSummary, decoded.TaskSummary)
	assert.Equal(t, original.RawNotes, decoded.RawNotes)
	assert.Equal(t, original.ChangedPaths, decoded.ChangedPaths)
	assert.Equal(t, original.Flags.ArchitectureImpact, decoded.Flags.ArchitectureImpact)
	assert.Equal(t, original.Flags.DesignPattern, decoded.Flags.DesignPattern)
	assert.Equal(t, original.Flags.LessonLearned, decoded.Flags.LessonLearned)
	assert.Equal(t, original.ClientRequestID, decoded.ClientRequestID)
	assert.Equal(t, original.Context.SessionID, decoded.Context.SessionID)
}

func TestIntakeEventJSONRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	original := IntakeEvent{
		NotifyPayload: NotifyPayload{
			Version:     1,
			SourceType:  "coding_agent",
			Agent:       "cursor",
			TaskSummary: "Fix bug in parser",
			RawNotes:    []string{"Fixed off-by-one error"},
		},
		EventID:     "E-01JZABC123XYZ7890ABCDEFGH",
		InstanceID:  "E-01JZABC123XYZ7890ABCDEFGH",
		ContentHash: "sha256:5be7c3abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234",
		ContentID:   "RAWC-3d8f1cabcdef1234567890abcdef1234567890abcdef1234567890abcdef12",
		Scope:       "branch",
		Timestamps: Timestamps{
			CreatedAt: now,
			StoredAt:  now,
		},
		Provenance: Provenance{
			Hostname: "devbox",
			User:     "developer",
			Cwd:      "/home/developer/project",
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded IntakeEvent
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.EventID, decoded.EventID)
	assert.Equal(t, original.ContentHash, decoded.ContentHash)
	assert.Equal(t, original.ContentID, decoded.ContentID)
	assert.Equal(t, original.Scope, decoded.Scope)
	assert.Equal(t, original.Provenance.Hostname, decoded.Provenance.Hostname)
	assert.Equal(t, original.Timestamps.CreatedAt, decoded.Timestamps.CreatedAt)
}

func TestNotifyResultJSONRoundTrip(t *testing.T) {
	original := NotifyResult{
		Status:      "accepted",
		Code:        CodeOK,
		Mode:        "deferred",
		EventID:     "E-01JZABC123XYZ7890ABCDEFGH",
		ContentHash: "sha256:abcdef",
		IndexState:  "ok",
		Warnings:    []string{},
		NextAction:  "none",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded NotifyResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Status, decoded.Status)
	assert.Equal(t, original.Code, decoded.Code)
	assert.Equal(t, original.EventID, decoded.EventID)
}

func TestValidAgents(t *testing.T) {
	expected := []string{"codex", "claude-code", "antigravity", "cursor", "unknown"}
	for _, agent := range expected {
		assert.True(t, ValidAgents[agent], "Expected %q to be a valid agent", agent)
	}
	assert.False(t, ValidAgents["invalid-agent"])
	assert.False(t, ValidAgents[""])
}
