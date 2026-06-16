package record

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSchemasDir(t *testing.T) string {
	t.Helper()
	// Relative path from this test file to the schemas directory
	return "../../../../../prompts/memory/schemas"
}

func TestValidator_Validate(t *testing.T) {
	schemasDir := testSchemasDir(t)
	v, err := NewValidator(schemasDir)
	require.NoError(t, err, "NewValidator should succeed")

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "valid payload with all fields",
			input: `{
				"version": 1,
				"source_type": "coding_agent",
				"agent": "antigravity",
				"task_summary": "Implement auth middleware",
				"raw_notes": ["Added JWT validation", "Updated config schema"],
				"changed_paths": ["internal/auth/middleware.go"],
				"flags": {"architecture_impact": true},
				"client_request_id": "req-001"
			}`,
			wantErr: false,
		},
		{
			name: "valid payload with minimum required fields",
			input: `{
				"version": 1,
				"source_type": "coding_agent",
				"agent": "cursor",
				"task_summary": "Fix bug",
				"raw_notes": ["Fixed it"]
			}`,
			wantErr: false,
		},
		{
			name: "missing required field raw_notes",
			input: `{
				"version": 1,
				"source_type": "coding_agent",
				"agent": "antigravity",
				"task_summary": "Some task"
			}`,
			wantErr: true,
		},
		{
			name: "missing required field task_summary",
			input: `{
				"version": 1,
				"source_type": "coding_agent",
				"agent": "antigravity",
				"raw_notes": ["note1"]
			}`,
			wantErr: true,
		},
		{
			name: "missing required field agent",
			input: `{
				"version": 1,
				"source_type": "coding_agent",
				"task_summary": "Some task",
				"raw_notes": ["note1"]
			}`,
			wantErr: true,
		},
		{
			name: "type mismatch: version as string instead of integer",
			input: `{
				"version": "1",
				"source_type": "coding_agent",
				"agent": "antigravity",
				"task_summary": "Some task",
				"raw_notes": ["note1"]
			}`,
			wantErr: true,
		},
		{
			name: "wrong version value",
			input: `{
				"version": 2,
				"source_type": "coding_agent",
				"agent": "antigravity",
				"task_summary": "Some task",
				"raw_notes": ["note1"]
			}`,
			wantErr: true,
		},
		{
			name: "agent not in enum",
			input: `{
				"version": 1,
				"source_type": "coding_agent",
				"agent": "invalid-agent",
				"task_summary": "Some task",
				"raw_notes": ["note1"]
			}`,
			wantErr: true,
		},
		{
			name: "raw_notes empty array (minItems 1)",
			input: `{
				"version": 1,
				"source_type": "coding_agent",
				"agent": "antigravity",
				"task_summary": "Some task",
				"raw_notes": []
			}`,
			wantErr: true,
		},
		{
			name:    "task_summary exceeds 500 characters",
			input:   buildPayloadWithSummaryLength(501),
			wantErr: true,
		},
		{
			name:    "task_summary at 500 characters is valid",
			input:   buildPayloadWithSummaryLength(500),
			wantErr: false,
		},
		{
			name:    "raw_notes with 33 items (maxItems 32)",
			input:   buildPayloadWithNoteCount(33),
			wantErr: true,
		},
		{
			name:    "raw_notes with 32 items is valid",
			input:   buildPayloadWithNoteCount(32),
			wantErr: false,
		},
		{
			name: "task_summary empty string (minLength 1)",
			input: `{
				"version": 1,
				"source_type": "coding_agent",
				"agent": "antigravity",
				"task_summary": "",
				"raw_notes": ["note1"]
			}`,
			wantErr: true,
		},
		{
			name: "additional property not allowed",
			input: `{
				"version": 1,
				"source_type": "coding_agent",
				"agent": "antigravity",
				"task_summary": "task",
				"raw_notes": ["note"],
				"unknown_field": "value"
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate([]byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err, "expected validation error")
			} else {
				assert.NoError(t, err, "expected no validation error")
			}
		})
	}
}

func buildPayloadWithSummaryLength(length int) string {
	summary := make([]byte, length)
	for i := range summary {
		summary[i] = 'a'
	}
	return `{"version":1,"source_type":"coding_agent","agent":"antigravity","task_summary":"` +
		string(summary) + `","raw_notes":["note"]}`
}

func buildPayloadWithNoteCount(count int) string {
	notes := "["
	for i := range count {
		if i > 0 {
			notes += ","
		}
		notes += `"note` + string(rune('A'+i%26)) + `"`
	}
	notes += "]"
	return `{"version":1,"source_type":"coding_agent","agent":"antigravity","task_summary":"task","raw_notes":` +
		notes + `}`
}
