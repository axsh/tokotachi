package storage

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestIndexEvent(eventID, clientReqID string) *agent.IntakeEvent {
	now := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	return &agent.IntakeEvent{
		NotifyPayload: agent.NotifyPayload{
			Version:         1,
			SourceType:      "coding_agent",
			Agent:           "antigravity",
			TaskSummary:     "Implement authentication middleware",
			RawNotes:        []string{"Added JWT validation", "Updated config schema"},
			ClientRequestID: clientReqID,
		},
		EventID:     eventID,
		InstanceID:  eventID,
		ContentHash: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		ContentID:   "RAWC-0000000000000000000000000000000000000000000000000000000000000000",
		Scope:       "branch",
		Git:         &agent.GitInfo{Branch: "feature/auth"},
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
}

func TestIndex_StoreAndGet(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	idx, err := NewIndex(dbPath)
	require.NoError(t, err)
	defer idx.Close()

	event := makeTestIndexEvent("E-01JZTEST0000000000000001", "req-001")

	existingID, err := idx.Store(event)
	require.NoError(t, err)
	assert.Empty(t, existingID, "first store should not return existing ID")

	// Retrieve by event_id
	record, err := idx.GetByEventID("E-01JZTEST0000000000000001")
	require.NoError(t, err)
	assert.Equal(t, "E-01JZTEST0000000000000001", record.EventID)
	assert.Equal(t, "Implement authentication middleware", record.TaskSummary)
	assert.Equal(t, "antigravity", record.Agent)
	assert.Equal(t, "feature/auth", record.Branch)
	assert.Equal(t, "branch", record.Scope)
	assert.Equal(t, "pending", record.Status)
}

func TestIndex_Idempotency(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	idx, err := NewIndex(dbPath)
	require.NoError(t, err)
	defer idx.Close()

	// First store
	event1 := makeTestIndexEvent("E-01JZTEST0000000000000002", "req-duplicate")
	existingID, err := idx.Store(event1)
	require.NoError(t, err)
	assert.Empty(t, existingID)

	// Second store with same client_request_id but different event_id
	event2 := makeTestIndexEvent("E-01JZTEST0000000000000003", "req-duplicate")
	existingID, err = idx.Store(event2)
	require.NoError(t, err)
	assert.Equal(t, "E-01JZTEST0000000000000002", existingID,
		"should return the original event_id on idempotent hit")
}

func TestIndex_WALMode(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	idx, err := NewIndex(dbPath)
	require.NoError(t, err)
	defer idx.Close()

	var journalMode string
	err = idx.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	require.NoError(t, err)
	assert.Equal(t, "wal", journalMode)
}

func TestIndex_FTSSearch(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	idx, err := NewIndex(dbPath)
	require.NoError(t, err)
	defer idx.Close()

	// Store events with different content
	event1 := makeTestIndexEvent("E-01JZTEST0000000000000004", "")
	event1.TaskSummary = "Implement JWT authentication"
	event1.RawNotes = []string{"Added bearer token validation"}
	_, err = idx.Store(event1)
	require.NoError(t, err)

	event2 := makeTestIndexEvent("E-01JZTEST0000000000000005", "")
	event2.TaskSummary = "Fix database migration"
	event2.RawNotes = []string{"Resolved schema conflict"}
	_, err = idx.Store(event2)
	require.NoError(t, err)

	t.Run("search by task_summary keyword", func(t *testing.T) {
		results, err := idx.SearchFTS("authentication")
		require.NoError(t, err)
		assert.Contains(t, results, "E-01JZTEST0000000000000004")
		assert.NotContains(t, results, "E-01JZTEST0000000000000005")
	})

	t.Run("search by raw_notes keyword", func(t *testing.T) {
		results, err := idx.SearchFTS("bearer")
		require.NoError(t, err)
		assert.Contains(t, results, "E-01JZTEST0000000000000004")
	})

	t.Run("search returns no results for non-matching query", func(t *testing.T) {
		results, err := idx.SearchFTS("nonexistent")
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestIndex_GetByEventID_NotFound(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	idx, err := NewIndex(dbPath)
	require.NoError(t, err)
	defer idx.Close()

	_, err = idx.GetByEventID("E-NONEXISTENT")
	assert.Error(t, err)
}
