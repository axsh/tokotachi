package record

import (
	"strings"
	"testing"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateEventID(t *testing.T) {
	t.Run("has E- prefix", func(t *testing.T) {
		id, err := GenerateEventID()
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(id, "E-"), "event_id should start with E-")
	})

	t.Run("ULID part is 26 characters", func(t *testing.T) {
		id, err := GenerateEventID()
		require.NoError(t, err)
		ulidPart := id[2:]
		assert.Len(t, ulidPart, 26, "ULID part should be 26 characters")
	})

	t.Run("uniqueness: 100 IDs are all distinct", func(t *testing.T) {
		seen := make(map[string]bool)
		for range 100 {
			id, err := GenerateEventID()
			require.NoError(t, err)
			assert.False(t, seen[id], "duplicate event_id: %s", id)
			seen[id] = true
		}
	})
}

func TestComputeContentHash(t *testing.T) {
	makeEvent := func(summary string, notes []string, paths []string) *agent.IntakeEvent {
		return &agent.IntakeEvent{
			NotifyPayload: agent.NotifyPayload{
				TaskSummary: summary,
				RawNotes:    notes,
			},
			EffectiveChangedPaths: paths,
		}
	}

	t.Run("has sha256: prefix", func(t *testing.T) {
		event := makeEvent("task", []string{"note"}, nil)
		hash := ComputeContentHash(event)
		assert.True(t, strings.HasPrefix(hash, "sha256:"), "content_hash should start with sha256:")
	})

	t.Run("hex digest is 64 characters", func(t *testing.T) {
		event := makeEvent("task", []string{"note"}, nil)
		hash := ComputeContentHash(event)
		hexPart := hash[len("sha256:"):]
		assert.Len(t, hexPart, 64, "SHA-256 hex digest should be 64 characters")
	})

	t.Run("same input produces same hash", func(t *testing.T) {
		event1 := makeEvent("task", []string{"note1", "note2"}, []string{"a.go"})
		event2 := makeEvent("task", []string{"note1", "note2"}, []string{"a.go"})
		assert.Equal(t, ComputeContentHash(event1), ComputeContentHash(event2))
	})

	t.Run("different input produces different hash", func(t *testing.T) {
		event1 := makeEvent("task1", []string{"note"}, nil)
		event2 := makeEvent("task2", []string{"note"}, nil)
		assert.NotEqual(t, ComputeContentHash(event1), ComputeContentHash(event2))
	})

	t.Run("note order does not affect hash (sorted)", func(t *testing.T) {
		event1 := makeEvent("task", []string{"alpha", "beta"}, nil)
		event2 := makeEvent("task", []string{"beta", "alpha"}, nil)
		assert.Equal(t, ComputeContentHash(event1), ComputeContentHash(event2))
	})
}

func TestComputeContentID(t *testing.T) {
	t.Run("has RAWC- prefix", func(t *testing.T) {
		event := &agent.IntakeEvent{
			NotifyPayload: agent.NotifyPayload{
				TaskSummary: "task",
				RawNotes:    []string{"note"},
			},
		}
		id := ComputeContentID(event)
		assert.True(t, strings.HasPrefix(id, "RAWC-"), "content_id should start with RAWC-")
	})

	t.Run("hex digest is 64 characters", func(t *testing.T) {
		event := &agent.IntakeEvent{
			NotifyPayload: agent.NotifyPayload{
				TaskSummary: "task",
				RawNotes:    []string{"note"},
			},
		}
		id := ComputeContentID(event)
		hexPart := id[len("RAWC-"):]
		assert.Len(t, hexPart, 64)
	})

	t.Run("branch independence: different branch same content produces same content_id", func(t *testing.T) {
		event1 := &agent.IntakeEvent{
			NotifyPayload: agent.NotifyPayload{
				TaskSummary: "implement feature",
				RawNotes:    []string{"added auth"},
			},
			EffectiveChangedPaths: []string{"internal/auth/handler.go"},
			Git:                   &agent.GitInfo{Branch: "feature-a"},
			Timestamps:            agent.Timestamps{CreatedAt: time.Now()},
		}
		event2 := &agent.IntakeEvent{
			NotifyPayload: agent.NotifyPayload{
				TaskSummary: "implement feature",
				RawNotes:    []string{"added auth"},
			},
			EffectiveChangedPaths: []string{"internal/auth/handler.go"},
			Git:                   &agent.GitInfo{Branch: "feature-b"},
			Timestamps:            agent.Timestamps{CreatedAt: time.Now().Add(time.Hour)},
		}
		assert.Equal(t, ComputeContentID(event1), ComputeContentID(event2))
	})

	t.Run("timestamp independence", func(t *testing.T) {
		event1 := &agent.IntakeEvent{
			NotifyPayload: agent.NotifyPayload{
				TaskSummary: "task",
				RawNotes:    []string{"note"},
			},
			Timestamps: agent.Timestamps{CreatedAt: time.Now()},
		}
		event2 := &agent.IntakeEvent{
			NotifyPayload: agent.NotifyPayload{
				TaskSummary: "task",
				RawNotes:    []string{"note"},
			},
			Timestamps: agent.Timestamps{CreatedAt: time.Now().Add(24 * time.Hour)},
		}
		assert.Equal(t, ComputeContentID(event1), ComputeContentID(event2))
	})
}
