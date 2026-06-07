package status

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/axsh/tokotachi/features/tt/internal/agent/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShow_Found(t *testing.T) {
	tmpDir := t.TempDir()
	varDir := tmpDir

	// Create file store and index
	intakeDir := filepath.Join(varDir, "intake")
	fs := storage.NewFileStore(intakeDir)
	idx := setupTestIndex(t, varDir)
	defer idx.Close()

	now := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	event := &agent.IntakeEvent{
		NotifyPayload: agent.NotifyPayload{
			Version:     1,
			Agent:       "antigravity",
			TaskSummary: "Test show command",
			RawNotes:    []string{"note1", "note2"},
		},
		EventID:     "E-SHOW001",
		InstanceID:  "E-SHOW001",
		ContentHash: "sha256:abc",
		ContentID:   "RAWC-abc",
		Scope:       "branch",
		Git:         &agent.GitInfo{Branch: "main"},
		Timestamps: agent.Timestamps{
			CreatedAt: now,
			StoredAt:  now,
		},
		Provenance: agent.Provenance{
			Hostname: "test",
			User:     "test",
			Cwd:      "/test",
		},
	}

	// Write file and index
	_, err := fs.Write(event)
	require.NoError(t, err)
	_, err = idx.Store(event)
	require.NoError(t, err)

	// Show
	result, err := Show(varDir, "E-SHOW001")
	require.NoError(t, err)
	assert.Equal(t, "E-SHOW001", result.EventID)
	assert.Equal(t, "Test show command", result.TaskSummary)
	assert.Equal(t, "antigravity", result.Agent)
}

func TestShow_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	idx := setupTestIndex(t, tmpDir)
	idx.Close()

	_, err := Show(tmpDir, "E-NONEXISTENT")
	assert.Error(t, err)
}

func TestRedactProvenance(t *testing.T) {
	event := &agent.IntakeEvent{
		NotifyPayload: agent.NotifyPayload{
			Agent:       "antigravity",
			TaskSummary: "Test redact",
		},
		EventID: "E-REDACT001",
		Provenance: agent.Provenance{
			Hostname: "YamDesktop",
			User:     "yamya",
			Cwd:      "/home/yamya/project",
		},
	}

	redacted := RedactProvenance(event)

	// Provenance should be redacted
	assert.Equal(t, "<redacted>", redacted.Provenance.Hostname)
	assert.Equal(t, "<redacted>", redacted.Provenance.User)
	assert.Equal(t, "<redacted>", redacted.Provenance.Cwd)

	// Other fields should be preserved
	assert.Equal(t, "E-REDACT001", redacted.EventID)
	assert.Equal(t, "antigravity", redacted.Agent)
	assert.Equal(t, "Test redact", redacted.TaskSummary)

	// Original event should NOT be modified
	assert.Equal(t, "YamDesktop", event.Provenance.Hostname)
	assert.Equal(t, "yamya", event.Provenance.User)
	assert.Equal(t, "/home/yamya/project", event.Provenance.Cwd)
}
