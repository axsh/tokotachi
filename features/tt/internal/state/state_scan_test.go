package state_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanStateFiles(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, root string)
		wantCount  int
		wantBranch []string
	}{
		{
			name:       "no files",
			setup:      func(t *testing.T, root string) {},
			wantCount:  0,
			wantBranch: nil,
		},
		{
			name: "one file",
			setup: func(t *testing.T, root string) {
				sf := state.StateFile{
					Branch:    "feat-a",
					CreatedAt: time.Now(),
					Features: map[string]state.FeatureState{
						"tt": {Status: state.StatusActive, StartedAt: time.Now()},
					},
				}
				require.NoError(t, state.Save(state.StatePath(root, "feat-a"), sf))
			},
			wantCount:  1,
			wantBranch: []string{"feat-a"},
		},
		{
			name: "multiple files",
			setup: func(t *testing.T, root string) {
				for _, branch := range []string{"feat-x", "feat-y"} {
					sf := state.StateFile{
						Branch:    branch,
						CreatedAt: time.Now(),
					}
					require.NoError(t, state.Save(state.StatePath(root, branch), sf))
				}
			},
			wantCount:  2,
			wantBranch: []string{"feat-x", "feat-y"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			// Ensure work/ directory exists for glob
			require.NoError(t, createWorkDir(root))
			tt.setup(t, root)

			got, err := state.ScanStateFiles(root)
			require.NoError(t, err)
			assert.Len(t, got, tt.wantCount)

			for _, branch := range tt.wantBranch {
				_, ok := got[branch]
				assert.True(t, ok, "expected branch %q in result", branch)
			}
		})
	}
}

func createWorkDir(root string) error {
	return os.MkdirAll(filepath.Join(root, "work"), 0o755)
}
