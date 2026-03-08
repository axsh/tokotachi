package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveCheckpoint_LoadCheckpoint(t *testing.T) {
	tmpDir := t.TempDir()
	info := &CheckpointInfo{
		CreatedAt:    "2026-03-08T18:45:00+09:00",
		ScaffoldName: "default",
		HeadCommit:   "abc1234",
		FilesCreated: []string{"README.md", "scripts/.gitkeep"},
	}

	err := SaveCheckpoint(tmpDir, info)
	require.NoError(t, err)

	loaded, err := LoadCheckpoint(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, "default", loaded.ScaffoldName)
	assert.Equal(t, "abc1234", loaded.HeadCommit)
	assert.Equal(t, []string{"README.md", "scripts/.gitkeep"}, loaded.FilesCreated)
}

func TestSaveCheckpoint_WithStash(t *testing.T) {
	tmpDir := t.TempDir()
	info := &CheckpointInfo{
		CreatedAt:    "2026-03-08T18:45:00+09:00",
		ScaffoldName: "default",
		HeadCommit:   "abc1234",
		StashRef:     "stash@{0}",
		FilesCreated: []string{"README.md"},
		FilesModified: []ModifiedFile{
			{Path: ".gitignore", Action: "append", OriginalContentHash: "sha256:xyz"},
		},
	}

	err := SaveCheckpoint(tmpDir, info)
	require.NoError(t, err)

	loaded, err := LoadCheckpoint(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, "stash@{0}", loaded.StashRef)
	assert.Len(t, loaded.FilesModified, 1)
}

func TestLoadCheckpoint_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := LoadCheckpoint(tmpDir)
	assert.Error(t, err)
}

func TestRemoveCheckpoint(t *testing.T) {
	tmpDir := t.TempDir()
	info := &CheckpointInfo{ScaffoldName: "test"}
	require.NoError(t, SaveCheckpoint(tmpDir, info))

	// Verify exists
	_, err := os.Stat(filepath.Join(tmpDir, CheckpointFileName))
	require.NoError(t, err)

	// Remove
	err = RemoveCheckpoint(tmpDir)
	require.NoError(t, err)

	// Verify gone
	_, err = os.Stat(filepath.Join(tmpDir, CheckpointFileName))
	assert.True(t, os.IsNotExist(err))
}

func TestBuildCheckpointFromPlan(t *testing.T) {
	plan := &Plan{
		ScaffoldName: "default",
		FilesToCreate: []FileAction{
			{Path: "README.md", Action: "create"},
		},
		FilesToModify: []FileAction{
			{Path: ".gitignore", Action: "append"},
		},
	}

	info := BuildCheckpointFromPlan(plan, "abc1234", "")
	assert.Equal(t, "default", info.ScaffoldName)
	assert.Equal(t, "abc1234", info.HeadCommit)
	assert.Equal(t, []string{"README.md"}, info.FilesCreated)
	assert.Len(t, info.FilesModified, 1)
}
