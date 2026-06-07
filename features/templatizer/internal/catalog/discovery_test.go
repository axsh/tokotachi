package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverOriginals(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) string // returns searchRoot
		wantErr   bool
		errSubstr string
		wantDefs  int    // expected number of definitions (when no error)
		checkBase string // expected suffix of BaseDir (when no error)
	}{
		{
			name: "single originals found",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				origDir := filepath.Join(tmpDir, "catalog", "originals", "org", "test-scaffold")
				require.NoError(t, os.MkdirAll(origDir, 0o755))
				scaffoldYAML := `
name: "test"
category: "unit"
description: "Test scaffold"
original_ref: "catalog/originals/org/test-scaffold"
`
				require.NoError(t, os.WriteFile(
					filepath.Join(origDir, "scaffold.yaml"), []byte(scaffoldYAML), 0o644,
				))
				return tmpDir
			},
			wantErr:   false,
			wantDefs:  1,
			checkBase: "catalog",
		},
		{
			name: "no originals found",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "some", "other", "dir"), 0o755))
				return tmpDir
			},
			wantErr:   true,
			errSubstr: "no originals directory found",
		},
		{
			name: "multiple originals found",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "a", "originals"), 0o755))
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "b", "originals"), 0o755))
				return tmpDir
			},
			wantErr:   true,
			errSubstr: "multiple originals directories found",
		},
		{
			name: "nested originals inside originals are skipped",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				// Create main originals with a scaffold.
				origDir := filepath.Join(tmpDir, "catalog", "originals", "org", "test")
				require.NoError(t, os.MkdirAll(origDir, 0o755))
				scaffoldYAML := `
name: "nested-test"
category: "unit"
description: "Nested test"
original_ref: "catalog/originals/org/test"
`
				require.NoError(t, os.WriteFile(
					filepath.Join(origDir, "scaffold.yaml"), []byte(scaffoldYAML), 0o644,
				))
				// Create a nested "originals" inside the first originals.
				// This should NOT be counted as a separate originals directory.
				nestedOrig := filepath.Join(tmpDir, "catalog", "originals", "sub", "originals")
				require.NoError(t, os.MkdirAll(nestedOrig, 0o755))
				return tmpDir
			},
			wantErr:  false,
			wantDefs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searchRoot := tt.setup(t)
			result, err := DiscoverOriginals(searchRoot)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
				assert.Nil(t, result)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Len(t, result.Definitions, tt.wantDefs)
			// BaseDir should be the parent of the originals directory.
			assert.NotEmpty(t, result.BaseDir)
			assert.NotEmpty(t, result.OriginalsDir)
			assert.DirExists(t, result.OriginalsDir)
			if tt.checkBase != "" {
				assert.Equal(t, tt.checkBase, filepath.Base(result.BaseDir))
			}
		})
	}
}
