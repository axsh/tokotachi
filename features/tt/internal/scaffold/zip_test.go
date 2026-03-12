package scaffold

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestZip creates a ZIP archive in memory with the given file entries.
func createTestZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		require.NoError(t, err)
		_, err = f.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func TestExtractZip_BasicFiles(t *testing.T) {
	data := createTestZip(t, map[string]string{
		"README.md":   "# Hello",
		"config.yaml": "key: value",
	})

	files, err := ExtractZip(data)
	require.NoError(t, err)
	require.Len(t, files, 2)

	// Build a map for easier assertion
	fileMap := make(map[string]string)
	for _, f := range files {
		fileMap[f.RelativePath] = string(f.Content)
	}
	assert.Equal(t, "# Hello", fileMap["README.md"])
	assert.Equal(t, "key: value", fileMap["config.yaml"])
}

func TestExtractZip_WithSubdirectories(t *testing.T) {
	data := createTestZip(t, map[string]string{
		"features/README.md":    "# Features",
		"shared/libs/README.md": "# Libs",
		"scripts/setup/init.sh": "#!/bin/bash",
	})

	files, err := ExtractZip(data)
	require.NoError(t, err)
	require.Len(t, files, 3)

	fileMap := make(map[string]string)
	for _, f := range files {
		fileMap[f.RelativePath] = string(f.Content)
	}
	assert.Equal(t, "# Features", fileMap["features/README.md"])
	assert.Equal(t, "# Libs", fileMap["shared/libs/README.md"])
	assert.Equal(t, "#!/bin/bash", fileMap["scripts/setup/init.sh"])
}

func TestExtractZip_EmptyZip(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	require.NoError(t, w.Close())

	files, err := ExtractZip(buf.Bytes())
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestExtractZip_InvalidData(t *testing.T) {
	_, err := ExtractZip([]byte("not a zip file"))
	assert.Error(t, err)
}

func TestExtractZip_SkipsDirectoryEntries(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// Add a directory entry
	_, err := w.Create("subdir/")
	require.NoError(t, err)

	// Add a file entry
	f, err := w.Create("subdir/file.txt")
	require.NoError(t, err)
	_, err = f.Write([]byte("contents"))
	require.NoError(t, err)

	require.NoError(t, w.Close())

	files, err := ExtractZip(buf.Bytes())
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "subdir/file.txt", files[0].RelativePath)
	assert.Equal(t, "contents", string(files[0].Content))
}
