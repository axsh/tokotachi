package status

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestEvent(t *testing.T, dir, eventID string, createdAt time.Time) {
	t.Helper()
	event := &agent.IntakeEvent{
		NotifyPayload: agent.NotifyPayload{
			Version:     1,
			Agent:       "antigravity",
			TaskSummary: "test",
			RawNotes:    []string{"note"},
		},
		EventID: eventID,
		Timestamps: agent.Timestamps{
			CreatedAt: createdAt,
			StoredAt:  createdAt,
		},
	}
	data, err := json.MarshalIndent(event, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, eventID+".json"), data, 0644))
}

func TestGetStatus_CountsCorrectly(t *testing.T) {
	tmpDir := t.TempDir()
	varDir := tmpDir
	intakeDir := filepath.Join(varDir, "intake")

	now := time.Now().UTC()

	// Create pending files
	pendingDir := filepath.Join(intakeDir, "pending", "2026", "06", "07")
	createTestEvent(t, pendingDir, "E-PENDING1", now)
	createTestEvent(t, pendingDir, "E-PENDING2", now.Add(-time.Hour))

	// Create processed files
	processedDir := filepath.Join(intakeDir, "processed", "2026", "06", "07")
	createTestEvent(t, processedDir, "E-PROC1", now)

	// Create failed files
	failedDir := filepath.Join(intakeDir, "failed", "2026", "06", "07")
	createTestEvent(t, failedDir, "E-FAIL1", now)
	createTestEvent(t, failedDir, "E-FAIL2", now)
	createTestEvent(t, failedDir, "E-FAIL3", now)

	report, err := GetStatus(varDir)
	require.NoError(t, err)

	assert.Equal(t, 2, report.PendingCount)
	assert.Equal(t, 1, report.ProcessedCount)
	assert.Equal(t, 3, report.FailedCount)
	assert.Equal(t, 0, report.IgnoredCount)
}

func TestGetStatus_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	report, err := GetStatus(tmpDir)
	require.NoError(t, err)

	assert.Equal(t, 0, report.PendingCount)
	assert.Equal(t, 0, report.ProcessedCount)
	assert.Equal(t, 0, report.FailedCount)
	assert.Equal(t, 0, report.IgnoredCount)
	assert.Empty(t, report.OldestPendingAge)
	assert.Equal(t, "unavailable", report.IndexHealth)
}

func TestGetStatus_OldestPendingAge(t *testing.T) {
	tmpDir := t.TempDir()
	pendingDir := filepath.Join(tmpDir, "intake", "pending", "2026", "06", "07")

	now := time.Now().UTC()
	createTestEvent(t, pendingDir, "E-NEW", now)
	createTestEvent(t, pendingDir, "E-OLD", now.Add(-2*time.Hour))

	report, err := GetStatus(tmpDir)
	require.NoError(t, err)

	assert.NotEmpty(t, report.OldestPendingAge)
	assert.Equal(t, "2h", report.OldestPendingAge)
}
