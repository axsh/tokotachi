package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestEvent(t *testing.T) *agent.IntakeEvent {
	t.Helper()
	now := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	return &agent.IntakeEvent{
		NotifyPayload: agent.NotifyPayload{
			Version:     1,
			SourceType:  "coding_agent",
			Agent:       "antigravity",
			TaskSummary: "Test task",
			RawNotes:    []string{"test note"},
		},
		EventID:    "E-01JZTEST0000000000000000",
		InstanceID: "E-01JZTEST0000000000000000",
		ContentHash: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		ContentID:   "RAWC-0000000000000000000000000000000000000000000000000000000000000000",
		Scope:       "branch",
		Timestamps: agent.Timestamps{
			CreatedAt: now,
			StoredAt:  now,
		},
		Provenance: agent.Provenance{
			Hostname: "test-host",
			User:     "test-user",
			Cwd:      "/test",
		},
	}
}

func TestFileStore_Write(t *testing.T) {
	t.Run("normal: file exists after write and contains valid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		fs := NewFileStore(tmpDir)
		event := makeTestEvent(t)

		relPath, err := fs.Write(event)
		require.NoError(t, err)

		// Verify file exists
		fullPath := filepath.Join(tmpDir, relPath)
		data, err := os.ReadFile(fullPath)
		require.NoError(t, err)

		// Verify valid JSON
		var decoded agent.IntakeEvent
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, event.EventID, decoded.EventID)
		assert.Equal(t, event.TaskSummary, decoded.TaskSummary)
	})

	t.Run("path structure: pending/YYYY/MM/DD/{event_id}.json", func(t *testing.T) {
		tmpDir := t.TempDir()
		fs := NewFileStore(tmpDir)
		event := makeTestEvent(t)

		relPath, err := fs.Write(event)
		require.NoError(t, err)

		expected := filepath.Join("pending", "2026", "06", "07",
			event.EventID+".json")
		assert.Equal(t, expected, relPath)
	})

	t.Run("auto-creates directories for new paths", func(t *testing.T) {
		tmpDir := t.TempDir()
		deepDir := filepath.Join(tmpDir, "a", "b", "c")
		fs := NewFileStore(deepDir)
		event := makeTestEvent(t)

		relPath, err := fs.Write(event)
		require.NoError(t, err)

		fullPath := filepath.Join(deepDir, relPath)
		_, err = os.Stat(fullPath)
		assert.NoError(t, err)
	})

	t.Run("tmp directory is cleaned up after successful write", func(t *testing.T) {
		tmpDir := t.TempDir()
		fs := NewFileStore(tmpDir)
		event := makeTestEvent(t)

		_, err := fs.Write(event)
		require.NoError(t, err)

		// _tmp directory may exist but should be empty
		tmpPath := filepath.Join(tmpDir, "_tmp")
		entries, err := os.ReadDir(tmpPath)
		if err == nil {
			assert.Empty(t, entries, "_tmp should be empty after successful write")
		}
	})

	t.Run("data roundtrip: written data matches original event", func(t *testing.T) {
		tmpDir := t.TempDir()
		fs := NewFileStore(tmpDir)
		event := makeTestEvent(t)
		event.RawNotes = []string{"note1", "note2"}
		event.ChangedPaths = []string{"a.go", "b.go"}

		relPath, err := fs.Write(event)
		require.NoError(t, err)

		fullPath := filepath.Join(tmpDir, relPath)
		data, err := os.ReadFile(fullPath)
		require.NoError(t, err)

		var decoded agent.IntakeEvent
		require.NoError(t, json.Unmarshal(data, &decoded))
		assert.Equal(t, event.Agent, decoded.Agent)
		assert.Equal(t, event.RawNotes, decoded.RawNotes)
		assert.Equal(t, event.ChangedPaths, decoded.ChangedPaths)
		assert.Equal(t, event.ContentHash, decoded.ContentHash)
	})

	t.Run("JSON is indented for readability", func(t *testing.T) {
		tmpDir := t.TempDir()
		fs := NewFileStore(tmpDir)
		event := makeTestEvent(t)

		relPath, err := fs.Write(event)
		require.NoError(t, err)

		fullPath := filepath.Join(tmpDir, relPath)
		data, err := os.ReadFile(fullPath)
		require.NoError(t, err)

		content := string(data)
		assert.True(t, strings.Contains(content, "  "), "JSON should be indented")
	})
}
