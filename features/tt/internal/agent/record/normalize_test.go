package record

import (
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizePayload(t *testing.T) {
	tests := []struct {
		name        string
		payload     agent.NotifyPayload
		wantSummary string
		wantNotes   []string
		wantErr     bool
	}{
		{
			name: "NFC normalization: decomposed dakuten to composed",
			payload: agent.NotifyPayload{
				TaskSummary: "\u304b\u3099", // ka + combining dakuten -> ga
				RawNotes:    []string{"\u304b\u3099\u304d\u3099"}, // ka+dakuten ki+dakuten
			},
			wantSummary: "\u304c",               // composed ga
			wantNotes:   []string{"\u304c\u304e"}, // composed ga gi
			wantErr:     false,
		},
		{
			name: "whitespace compression",
			payload: agent.NotifyPayload{
				TaskSummary: "hello   world\t\ttab",
				RawNotes:    []string{"multiple   spaces   here"},
			},
			wantSummary: "hello world tab",
			wantNotes:   []string{"multiple spaces here"},
			wantErr:     false,
		},
		{
			name: "trim leading and trailing whitespace",
			payload: agent.NotifyPayload{
				TaskSummary: "  trimmed  ",
				RawNotes:    []string{"  note with spaces  "},
			},
			wantSummary: "trimmed",
			wantNotes:   []string{"note with spaces"},
			wantErr:     false,
		},
		{
			name: "empty notes removed after normalization",
			payload: agent.NotifyPayload{
				TaskSummary: "task",
				RawNotes:    []string{"valid note", "   ", "  \t  ", "another valid"},
			},
			wantSummary: "task",
			wantNotes:   []string{"valid note", "another valid"},
			wantErr:     false,
		},
		{
			name: "all notes become empty after normalization",
			payload: agent.NotifyPayload{
				TaskSummary: "task",
				RawNotes:    []string{"   ", "  \t  ", ""},
			},
			wantErr: true,
		},
		{
			name: "already normalized text passes through unchanged",
			payload: agent.NotifyPayload{
				TaskSummary: "already clean",
				RawNotes:    []string{"clean note"},
			},
			wantSummary: "already clean",
			wantNotes:   []string{"clean note"},
			wantErr:     false,
		},
		{
			name: "newlines and carriage returns compressed",
			payload: agent.NotifyPayload{
				TaskSummary: "line1\nline2\r\nline3",
				RawNotes:    []string{"note\nwith\nnewlines"},
			},
			wantSummary: "line1 line2 line3",
			wantNotes:   []string{"note with newlines"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.payload
			err := NormalizePayload(&p)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantSummary, p.TaskSummary)
			assert.Equal(t, tt.wantNotes, p.RawNotes)
		})
	}
}
