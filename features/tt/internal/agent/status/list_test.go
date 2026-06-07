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
