package storage

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditLog_Append(t *testing.T) {
	t.Run("success entry goes to agent-notify.ndjson", func(t *testing.T) {
		tmpDir := t.TempDir()
		al := NewAuditLog(tmpDir)

		event := &agent.IntakeEvent{
			NotifyPayload: agent.NotifyPayload{
				Agent:       "antigravity",
				TaskSummary: "Test task",
			},
		}
		result := &agent.NotifyResult{
			Status:  "accepted",
			Code:    agent.CodeOK,
			EventID: "E-01JZTEST0000000000000001",
		}

		err := al.Append(event, result)
		require.NoError(t, err)

		// Verify file exists and has one line
		logPath := filepath.Join(tmpDir, "agent-notify.ndjson")
		entries := readNDJSON(t, logPath)
		require.Len(t, entries, 1)
		assert.Equal(t, "accepted", entries[0].Status)
		assert.Equal(t, "OK", entries[0].Code)
		assert.Equal(t, "E-01JZTEST0000000000000001", entries[0].EventID)
		assert.Equal(t, "antigravity", entries[0].Agent)

		// Error file should not exist
		errorPath := filepath.Join(tmpDir, "agent-notify-error.ndjson")
		_, err = os.Stat(errorPath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("rejected entry goes to agent-notify-error.ndjson", func(t *testing.T) {
		tmpDir := t.TempDir()
		al := NewAuditLog(tmpDir)

		event := &agent.IntakeEvent{
			NotifyPayload: agent.NotifyPayload{
				Agent:       "cursor",
				TaskSummary: "Bad task",
			},
		}
		result := &agent.NotifyResult{
			Status:  "rejected",
			Code:    agent.CodeSchemaValidationError,
			Message: "validation failed",
		}

		err := al.Append(event, result)
		require.NoError(t, err)

		errorPath := filepath.Join(tmpDir, "agent-notify-error.ndjson")
		entries := readNDJSON(t, errorPath)
		require.Len(t, entries, 1)
		assert.Equal(t, "rejected", entries[0].Status)
		assert.Equal(t, agent.CodeSchemaValidationError, entries[0].Code)

		// Success file should not exist
		logPath := filepath.Join(tmpDir, "agent-notify.ndjson")
		_, err = os.Stat(logPath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("accepted_with_warnings goes to agent-notify.ndjson", func(t *testing.T) {
		tmpDir := t.TempDir()
		al := NewAuditLog(tmpDir)

		event := &agent.IntakeEvent{
			NotifyPayload: agent.NotifyPayload{
				Agent:       "codex",
				TaskSummary: "Task with warnings",
			},
		}
		result := &agent.NotifyResult{
			Status:  "accepted_with_warnings",
			Code:    agent.CodeOK,
			EventID: "E-01JZTEST0000000000000002",
		}

		err := al.Append(event, result)
		require.NoError(t, err)

		logPath := filepath.Join(tmpDir, "agent-notify.ndjson")
		entries := readNDJSON(t, logPath)
		require.Len(t, entries, 1)
		assert.Equal(t, "accepted_with_warnings", entries[0].Status)
	})

	t.Run("multiple entries are appended", func(t *testing.T) {
		tmpDir := t.TempDir()
		al := NewAuditLog(tmpDir)

		for i := range 3 {
			event := &agent.IntakeEvent{
				NotifyPayload: agent.NotifyPayload{
					Agent:       "antigravity",
					TaskSummary: "task",
				},
			}
			result := &agent.NotifyResult{
				Status:  "accepted",
				Code:    agent.CodeOK,
				EventID: "E-" + string(rune('A'+i)),
			}
			require.NoError(t, al.Append(event, result))
		}

		logPath := filepath.Join(tmpDir, "agent-notify.ndjson")
		entries := readNDJSON(t, logPath)
		assert.Len(t, entries, 3)
	})

	t.Run("each line is valid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		al := NewAuditLog(tmpDir)

		event := &agent.IntakeEvent{
			NotifyPayload: agent.NotifyPayload{
				Agent:       "antigravity",
				TaskSummary: "test",
			},
		}
		result := &agent.NotifyResult{
			Status: "accepted",
			Code:   agent.CodeOK,
		}
		require.NoError(t, al.Append(event, result))

		logPath := filepath.Join(tmpDir, "agent-notify.ndjson")
		data, err := os.ReadFile(logPath)
		require.NoError(t, err)

		var entry AuditEntry
		err = json.Unmarshal(data[:len(data)-1], &entry) // strip trailing newline
		assert.NoError(t, err)
		assert.NotEmpty(t, entry.Timestamp)
	})
}

func readNDJSON(t *testing.T, path string) []AuditEntry {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	var entries []AuditEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry AuditEntry
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &entry))
		entries = append(entries, entry)
	}
	require.NoError(t, scanner.Err())
	return entries
}
