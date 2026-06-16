package status

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
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

func TestGetStatus_NewFormat(t *testing.T) {
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

	report, err := GetStatus("prompts/memory", varDir, "fix-memory-compiling")
	require.NoError(t, err)

	// New format fields
	assert.Equal(t, "prompts/memory", report.MemoryRoot)
	assert.Equal(t, "fix-memory-compiling", report.CurrentBranch)

	// Counts
	assert.Equal(t, 2, report.Counts.Pending)
	assert.Equal(t, 1, report.Counts.Processed)
	assert.Equal(t, 3, report.Counts.Failed)
	assert.Equal(t, 0, report.Counts.Ignored)
}

func TestGetStatus_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	report, err := GetStatus("prompts/memory", tmpDir, "main")
	require.NoError(t, err)

	assert.Equal(t, "prompts/memory", report.MemoryRoot)
	assert.Equal(t, "main", report.CurrentBranch)
	assert.Equal(t, 0, report.Counts.Pending)
	assert.Equal(t, 0, report.Counts.Processed)
	assert.Equal(t, 0, report.Counts.Failed)
	assert.Equal(t, 0, report.Counts.Ignored)
	assert.Empty(t, report.OldestPending)
	assert.Equal(t, "missing", report.IndexHealth)
}

func TestGetStatus_OldestPendingISO(t *testing.T) {
	tmpDir := t.TempDir()
	pendingDir := filepath.Join(tmpDir, "intake", "pending", "2026", "06", "07")

	t1 := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 7, 10, 30, 0, 0, time.UTC)
	createTestEvent(t, pendingDir, "E-NEW", t1)
	createTestEvent(t, pendingDir, "E-OLD", t2)

	report, err := GetStatus("prompts/memory", tmpDir, "main")
	require.NoError(t, err)

	// Should be ISO8601 seconds precision
	assert.Equal(t, "2026-06-07T10:30:00Z", report.OldestPending)

	// Validate format is ISO8601
	iso8601Pattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`)
	assert.True(t, iso8601Pattern.MatchString(report.OldestPending),
		"OldestPending should be ISO8601 seconds precision, got %q", report.OldestPending)
}

func TestGetStatus_IndexMissing(t *testing.T) {
	tmpDir := t.TempDir()

	report, err := GetStatus("prompts/memory", tmpDir, "main")
	require.NoError(t, err)

	assert.Equal(t, "missing", report.IndexHealth)
	assert.Nil(t, report.CurrentBranchCounts)
}

func TestGetStatus_IndexOk(t *testing.T) {
	tmpDir := t.TempDir()
	idx := setupTestIndex(t, tmpDir)
	defer idx.Close()

	report, err := GetStatus("prompts/memory", tmpDir, "main")
	require.NoError(t, err)

	assert.Equal(t, "ok", report.IndexHealth)
}

func TestGetStatus_BranchCounts(t *testing.T) {
	tmpDir := t.TempDir()
	idx := setupTestIndex(t, tmpDir)

	now := time.Now().UTC()

	// Insert events for two branches
	e1 := &agent.IntakeEvent{
		NotifyPayload: agent.NotifyPayload{
			Agent:       "antigravity",
			TaskSummary: "task on fix-branch",
			RawNotes:    []string{"note"},
		},
		EventID:     "E-BR1",
		ContentHash: "sha256:br1",
		ContentID:   "RAWC-br1",
		Scope:       "branch",
		Git:         &agent.GitInfo{Branch: "fix-branch"},
		Timestamps:  agent.Timestamps{CreatedAt: now, StoredAt: now},
	}
	_, err := idx.Store(e1)
	require.NoError(t, err)

	e2 := &agent.IntakeEvent{
		NotifyPayload: agent.NotifyPayload{
			Agent:       "antigravity",
			TaskSummary: "task on main",
			RawNotes:    []string{"note"},
		},
		EventID:     "E-BR2",
		ContentHash: "sha256:br2",
		ContentID:   "RAWC-br2",
		Scope:       "branch",
		Git:         &agent.GitInfo{Branch: "main"},
		Timestamps:  agent.Timestamps{CreatedAt: now, StoredAt: now},
	}
	_, err = idx.Store(e2)
	require.NoError(t, err)

	idx.Close()

	// Check fix-branch counts
	report, err := GetStatus("prompts/memory", tmpDir, "fix-branch")
	require.NoError(t, err)

	require.NotNil(t, report.CurrentBranchCounts)
	assert.Equal(t, 1, report.CurrentBranchCounts.Pending)

	// Check main counts
	report2, err := GetStatus("prompts/memory", tmpDir, "main")
	require.NoError(t, err)

	require.NotNil(t, report2.CurrentBranchCounts)
	assert.Equal(t, 1, report2.CurrentBranchCounts.Pending)
}


