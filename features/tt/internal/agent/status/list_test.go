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

func setupTestIndex(t *testing.T, varDir string) *storage.Index {
	t.Helper()
	dbPath := filepath.Join(varDir, "intake", "index.db")
	idx, err := storage.NewIndex(dbPath)
	require.NoError(t, err)
	return idx
}

func storeTestEvent(t *testing.T, idx *storage.Index, eventID, agentName, branch, summary string) {
	t.Helper()
	now := time.Now().UTC()
	event := &agent.IntakeEvent{
		NotifyPayload: agent.NotifyPayload{
			Version:     1,
			Agent:       agentName,
			TaskSummary: summary,
			RawNotes:    []string{"note"},
		},
		EventID:     eventID,
		ContentHash: "sha256:test",
		ContentID:   "RAWC-test",
		Scope:       "branch",
		Git:         &agent.GitInfo{Branch: branch},
		Timestamps: agent.Timestamps{
			CreatedAt: now,
			StoredAt:  now,
		},
	}
	_, err := idx.Store(event)
	require.NoError(t, err)
}

func TestList_NoFilter(t *testing.T) {
	tmpDir := t.TempDir()
	idx := setupTestIndex(t, tmpDir)

	storeTestEvent(t, idx, "E-001", "antigravity", "main", "Task 1")
	storeTestEvent(t, idx, "E-002", "cursor", "feature/a", "Task 2")
	idx.Close()

	items, err := List(tmpDir, ListOptions{})
	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestList_FilterByAgent(t *testing.T) {
	tmpDir := t.TempDir()
	idx := setupTestIndex(t, tmpDir)

	storeTestEvent(t, idx, "E-003", "antigravity", "main", "AG task")
	storeTestEvent(t, idx, "E-004", "cursor", "main", "Cursor task")
	idx.Close()

	items, err := List(tmpDir, ListOptions{Agent: "antigravity"})
	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "antigravity", items[0].Agent)
}

func TestList_FTSSearch(t *testing.T) {
	tmpDir := t.TempDir()
	idx := setupTestIndex(t, tmpDir)

	storeTestEvent(t, idx, "E-005", "antigravity", "main", "Implement authentication")
	storeTestEvent(t, idx, "E-006", "cursor", "main", "Fix database migration")
	idx.Close()

	items, err := List(tmpDir, ListOptions{Query: "authentication"})
	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "E-005", items[0].EventID)
}

func TestList_Limit(t *testing.T) {
	tmpDir := t.TempDir()
	idx := setupTestIndex(t, tmpDir)

	storeTestEvent(t, idx, "E-007", "antigravity", "main", "Task A")
	storeTestEvent(t, idx, "E-008", "antigravity", "main", "Task B")
	storeTestEvent(t, idx, "E-009", "antigravity", "main", "Task C")
	idx.Close()

	items, err := List(tmpDir, ListOptions{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestList_EmptyResult(t *testing.T) {
	tmpDir := t.TempDir()
	idx := setupTestIndex(t, tmpDir)
	idx.Close()

	items, err := List(tmpDir, ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestList_FilterByStatus_AfterUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	idx := setupTestIndex(t, tmpDir)

	storeTestEvent(t, idx, "E-010", "antigravity", "main", "Task to process")
	storeTestEvent(t, idx, "E-011", "antigravity", "main", "Task stays pending")

	// Update E-010 to processed
	require.NoError(t, idx.UpdateStatus("E-010", "processed"))
	idx.Close()

	// List pending: should only see E-011
	pendingItems, err := List(tmpDir, ListOptions{Status: "pending"})
	require.NoError(t, err)
	assert.Len(t, pendingItems, 1)
	assert.Equal(t, "E-011", pendingItems[0].EventID)

	// List processed: should only see E-010
	processedItems, err := List(tmpDir, ListOptions{Status: "processed"})
	require.NoError(t, err)
	assert.Len(t, processedItems, 1)
	assert.Equal(t, "E-010", processedItems[0].EventID)
}
