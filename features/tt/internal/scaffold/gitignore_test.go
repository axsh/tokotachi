package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitignoreAddEntries(t *testing.T) {
	gi := &Gitignore{lines: []string{"*.log", "tmp/"}}
	added := gi.AddEntries([]string{"work/*", "build/"})
	assert.Equal(t, 2, added)
	assert.True(t, gi.HasEntry("work/*"))
	assert.True(t, gi.HasEntry("build/"))
}

func TestGitignoreAddEntries_NoDuplicate(t *testing.T) {
	gi := &Gitignore{lines: []string{"*.log", "tmp/"}}
	added := gi.AddEntries([]string{"*.log", "new/"})
	assert.Equal(t, 1, added) // only "new/" is new
	assert.True(t, gi.HasEntry("new/"))
}

func TestGitignoreAddEntries_WithComments(t *testing.T) {
	gi := &Gitignore{lines: []string{"# Build artifacts", "*.o", "", "# Temp files", "tmp/"}}
	added := gi.AddEntries([]string{"dist/"})
	assert.Equal(t, 1, added)
	// Comments and empty lines should be preserved
	assert.Equal(t, "# Build artifacts", gi.lines[0])
	assert.Equal(t, "", gi.lines[2])
	assert.Equal(t, "# Temp files", gi.lines[3])
}

func TestGitignoreAddEntries_TrimTrailing(t *testing.T) {
	gi := &Gitignore{lines: []string{"work/*  "}} // trailing spaces
	added := gi.AddEntries([]string{"work/*"})
	assert.Equal(t, 0, added) // should detect as duplicate after trimming
}

func TestGitignoreRemoveEntries(t *testing.T) {
	gi := &Gitignore{lines: []string{"*.log", "tmp/", "work/*"}}
	removed := gi.RemoveEntries([]string{"tmp/"})
	assert.Equal(t, 1, removed)
	assert.False(t, gi.HasEntry("tmp/"))
	assert.True(t, gi.HasEntry("*.log"))
	assert.True(t, gi.HasEntry("work/*"))
}

func TestGitignoreHasEntry(t *testing.T) {
	gi := &Gitignore{lines: []string{"# comment", "", "work/*", "  tmp/  "}}
	assert.True(t, gi.HasEntry("work/*"))
	assert.True(t, gi.HasEntry("tmp/"))       // should match after trimming
	assert.False(t, gi.HasEntry("# comment")) // comments are not entries
	assert.False(t, gi.HasEntry(""))
	assert.False(t, gi.HasEntry("missing"))
}

func TestGitignoreLoadAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".gitignore")

	// Write a sample file
	require.NoError(t, os.WriteFile(path, []byte("*.log\n# comment\ntmp/\n"), 0o644))

	gi, err := LoadGitignore(path)
	require.NoError(t, err)
	assert.True(t, gi.HasEntry("*.log"))
	assert.True(t, gi.HasEntry("tmp/"))

	gi.AddEntries([]string{"work/*"})
	require.NoError(t, gi.Save(path))

	// Re-read and verify
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "work/*")
	assert.Contains(t, string(content), "# comment")
}

func TestGitignoreLoad_NotExist(t *testing.T) {
	gi, err := LoadGitignore("/nonexistent/.gitignore")
	require.NoError(t, err)
	assert.NotNil(t, gi)
	assert.Empty(t, gi.lines)
}

func TestGitignoreCreateFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".gitignore")

	gi, err := LoadGitignore(path)
	require.NoError(t, err)
	gi.AddEntries([]string{"work/*"})
	require.NoError(t, gi.Save(path))

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "work/*\n", string(content))
}
