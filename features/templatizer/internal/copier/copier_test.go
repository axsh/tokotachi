package copier

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyDir(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string // relative path -> content
	}{
		{
			name: "flat files",
			files: map[string]string{
				"file1.txt": "hello",
				"file2.txt": "world",
			},
		},
		{
			name: "nested directories",
			files: map[string]string{
				"root.txt":          "root",
				"sub/nested.txt":    "nested",
				"sub/deep/deep.txt": "deep",
			},
		},
		{
			name: "empty file",
			files: map[string]string{
				"empty.txt": "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create source directory with test files
			srcDir := t.TempDir()
			for relPath, content := range tt.files {
				absPath := filepath.Join(srcDir, relPath)
				require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0o755))
				require.NoError(t, os.WriteFile(absPath, []byte(content), 0o644))
			}

			// Create destination directory
			destDir := t.TempDir()

			// Execute copy
			err := CopyDir(srcDir, destDir)
			require.NoError(t, err)

			// Verify all files are present with correct content
			for relPath, expectedContent := range tt.files {
				destPath := filepath.Join(destDir, relPath)
				data, err := os.ReadFile(destPath)
				require.NoError(t, err, "expected file %s to exist in dest", relPath)
				assert.Equal(t, expectedContent, string(data), "content mismatch for %s", relPath)
			}

			// Count files in dest to ensure no extra files
			var destFileCount int
			err = filepath.WalkDir(destDir, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() {
					destFileCount++
				}
				return nil
			})
			require.NoError(t, err)
			assert.Equal(t, len(tt.files), destFileCount, "unexpected number of files in dest")
		})
	}
}

func TestCopyDirNotFound(t *testing.T) {
	destDir := t.TempDir()
	err := CopyDir("/nonexistent/path/that/does/not/exist", destDir)
	assert.Error(t, err)
}

func TestCopyDirPreservesSource(t *testing.T) {
	// Create source directory with known content
	srcDir := t.TempDir()
	files := map[string]string{
		"file1.txt":       "original content 1",
		"sub/file2.txt":   "original content 2",
		"sub/a/file3.txt": "original content 3",
	}
	for relPath, content := range files {
		absPath := filepath.Join(srcDir, relPath)
		require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0o755))
		require.NoError(t, os.WriteFile(absPath, []byte(content), 0o644))
	}

	// Execute copy
	destDir := t.TempDir()
	err := CopyDir(srcDir, destDir)
	require.NoError(t, err)

	// Verify source files are unchanged
	for relPath, expectedContent := range files {
		srcPath := filepath.Join(srcDir, relPath)
		data, err := os.ReadFile(srcPath)
		require.NoError(t, err, "source file %s should still exist", relPath)
		assert.Equal(t, expectedContent, string(data), "source file %s was modified", relPath)
	}
}
