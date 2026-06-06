package archiver

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZipDirectory(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string // relative path -> content
	}{
		{
			name: "flat files",
			files: map[string]string{
				"file1.txt": "hello world",
				"file2.txt": "foo bar",
			},
		},
		{
			name: "nested directories",
			files: map[string]string{
				"root.txt":          "root content",
				"sub/nested.txt":    "nested content",
				"sub/deep/deep.txt": "deep content",
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

			// Create ZIP
			destPath := filepath.Join(t.TempDir(), "output.zip")
			err := ZipDirectory(srcDir, destPath)
			require.NoError(t, err)

			// Verify ZIP exists
			info, err := os.Stat(destPath)
			require.NoError(t, err)
			assert.True(t, info.Size() > 0)

			// Verify ZIP contents
			reader, err := zip.OpenReader(destPath)
			require.NoError(t, err)
			defer reader.Close()

			// Collect ZIP entries
			zipContents := make(map[string]string)
			for _, f := range reader.File {
				rc, err := f.Open()
				require.NoError(t, err)
				data, err := io.ReadAll(rc)
				require.NoError(t, err)
				rc.Close()
				zipContents[f.Name] = string(data)
			}

			// Verify all original files are present with correct content
			assert.Len(t, zipContents, len(tt.files))
			for relPath, expectedContent := range tt.files {
				// ZIP uses forward slashes
				zipKey := filepath.ToSlash(relPath)
				actualContent, exists := zipContents[zipKey]
				assert.True(t, exists, "expected file %s in ZIP", zipKey)
				assert.Equal(t, expectedContent, actualContent, "content mismatch for %s", zipKey)
			}
		})
	}
}

func TestZipDirectoryNotFound(t *testing.T) {
	destPath := filepath.Join(t.TempDir(), "output.zip")
	err := ZipDirectory("/nonexistent/path/that/does/not/exist", destPath)
	assert.Error(t, err)
}

func TestZipDirectoryOverwrite(t *testing.T) {
	// Create source directory
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("content"), 0o644))

	// Create destination and write initial content
	destPath := filepath.Join(t.TempDir(), "output.zip")
	require.NoError(t, os.WriteFile(destPath, []byte("old content"), 0o644))

	// ZipDirectory should overwrite
	err := ZipDirectory(srcDir, destPath)
	require.NoError(t, err)

	// Verify it's a valid ZIP (not old content)
	reader, err := zip.OpenReader(destPath)
	require.NoError(t, err)
	defer reader.Close()
	assert.Len(t, reader.File, 1)
}
