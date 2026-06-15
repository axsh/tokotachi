package intake

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMoveToProcessed(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, varDir string) string // returns event ID
		eventID   string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "move pending event to processed",
			setup: func(t *testing.T, varDir string) string {
				t.Helper()
				pendingDir := filepath.Join(varDir, "intake", "pending", "2026-06-15")
				require.NoError(t, os.MkdirAll(pendingDir, 0o755))
				evt := map[string]any{
					"event_id":     "E-01TEST001",
					"task_summary": "test event",
				}
				data, _ := json.Marshal(evt)
				require.NoError(t, os.WriteFile(filepath.Join(pendingDir, "E-01TEST001.json"), data, 0o644))
				return "E-01TEST001"
			},
			eventID: "E-01TEST001",
			wantErr: false,
		},
		{
			name: "empty directory is cleaned up after move",
			setup: func(t *testing.T, varDir string) string {
				t.Helper()
				pendingDir := filepath.Join(varDir, "intake", "pending", "2026-06-10")
				require.NoError(t, os.MkdirAll(pendingDir, 0o755))
				evt := map[string]any{
					"event_id":     "E-01TEST002",
					"task_summary": "sole event",
				}
				data, _ := json.Marshal(evt)
				require.NoError(t, os.WriteFile(filepath.Join(pendingDir, "E-01TEST002.json"), data, 0o644))
				return "E-01TEST002"
			},
			eventID: "E-01TEST002",
			wantErr: false,
		},
		{
			name: "event not found in pending",
			setup: func(t *testing.T, varDir string) string {
				t.Helper()
				pendingDir := filepath.Join(varDir, "intake", "pending")
				require.NoError(t, os.MkdirAll(pendingDir, 0o755))
				return ""
			},
			eventID:   "E-NONEXISTENT",
			wantErr:   true,
			errSubstr: "not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			varDir := t.TempDir()
			tc.setup(t, varDir)

			err := MoveToProcessed(varDir, tc.eventID)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
				return
			}

			require.NoError(t, err)

			// Verify file moved to processed
			processedDir := filepath.Join(varDir, "intake", "processed")
			found := false
			err = filepath.WalkDir(processedDir, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				if filepath.Base(path) == tc.eventID+".json" {
					found = true
				}
				return nil
			})
			require.NoError(t, err)
			assert.True(t, found, "event should exist in processed directory")

			// Verify file removed from pending
			pendingDir := filepath.Join(varDir, "intake", "pending")
			pendingFound := false
			_ = filepath.WalkDir(pendingDir, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if !d.IsDir() && filepath.Base(path) == tc.eventID+".json" {
					pendingFound = true
				}
				return nil
			})
			assert.False(t, pendingFound, "event should not exist in pending directory")
		})
	}

	t.Run("empty date directory cleaned up", func(t *testing.T) {
		varDir := t.TempDir()
		pendingDir := filepath.Join(varDir, "intake", "pending", "2026-01-01")
		require.NoError(t, os.MkdirAll(pendingDir, 0o755))
		evt := map[string]any{"event_id": "E-01CLEANUP", "task_summary": "cleanup test"}
		data, _ := json.Marshal(evt)
		require.NoError(t, os.WriteFile(filepath.Join(pendingDir, "E-01CLEANUP.json"), data, 0o644))

		err := MoveToProcessed(varDir, "E-01CLEANUP")
		require.NoError(t, err)

		// Date directory should be removed since it's empty now
		_, err = os.Stat(pendingDir)
		assert.True(t, os.IsNotExist(err), "empty date directory should be removed")
	})
}
