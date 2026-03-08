package resolve_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/resolve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectGitWorktree_WorktreeConfig(t *testing.T) {
	root := t.TempDir()

	// Simulate parent repo .git dir
	mainGitDir := filepath.Join(root, ".git")
	worktreeMetaDir := filepath.Join(mainGitDir, "worktrees", "test-branch")
	require.NoError(t, os.MkdirAll(worktreeMetaDir, 0755))

	// Create commondir file pointing to parent .git
	require.NoError(t, os.WriteFile(
		filepath.Join(worktreeMetaDir, "commondir"),
		[]byte("../..\n"),
		0644,
	))

	// Create HEAD file (standard worktree metadata)
	require.NoError(t, os.WriteFile(
		filepath.Join(worktreeMetaDir, "HEAD"),
		[]byte("ref: refs/heads/test-branch\n"),
		0644,
	))

	// Simulate worktree directory with .git file
	worktreeDir := filepath.Join(root, "work", "feat", "test-branch")
	require.NoError(t, os.MkdirAll(worktreeDir, 0755))

	// .git file pointing to worktree metadata
	gitFileContent := "gitdir: " + worktreeMetaDir + "\n"
	require.NoError(t, os.WriteFile(
		filepath.Join(worktreeDir, ".git"),
		[]byte(gitFileContent),
		0644,
	))

	info, err := resolve.DetectGitWorktree(worktreeDir)
	require.NoError(t, err)
	assert.True(t, info.IsWorktree)
	assert.Equal(t, worktreeMetaDir, info.WorktreeGitDir)
	assert.Equal(t, mainGitDir, info.MainGitDir)
}

func TestDetectGitWorktree_RegularGitDir(t *testing.T) {
	root := t.TempDir()

	// Create .git as a directory (regular repo)
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0755))

	info, err := resolve.DetectGitWorktree(root)
	require.NoError(t, err)
	assert.False(t, info.IsWorktree)
	assert.Empty(t, info.WorktreeGitDir)
	assert.Empty(t, info.MainGitDir)
}

func TestDetectGitWorktree_NoGitEntry(t *testing.T) {
	root := t.TempDir()

	info, err := resolve.DetectGitWorktree(root)
	require.NoError(t, err)
	assert.False(t, info.IsWorktree)
}

func TestDetectGitWorktree_RelativePath(t *testing.T) {
	root := t.TempDir()

	// Simulate parent repo .git dir
	mainGitDir := filepath.Join(root, ".git")
	worktreeMetaDir := filepath.Join(mainGitDir, "worktrees", "test-branch")
	require.NoError(t, os.MkdirAll(worktreeMetaDir, 0755))

	// Create commondir file
	require.NoError(t, os.WriteFile(
		filepath.Join(worktreeMetaDir, "commondir"),
		[]byte("../..\n"),
		0644,
	))

	// Simulate worktree directory with .git file using RELATIVE path
	worktreeDir := filepath.Join(root, "work", "feat", "test-branch")
	require.NoError(t, os.MkdirAll(worktreeDir, 0755))

	// Use relative path in .git file
	relPath, err := filepath.Rel(worktreeDir, worktreeMetaDir)
	require.NoError(t, err)

	gitFileContent := "gitdir: " + relPath + "\n"
	require.NoError(t, os.WriteFile(
		filepath.Join(worktreeDir, ".git"),
		[]byte(gitFileContent),
		0644,
	))

	info, err := resolve.DetectGitWorktree(worktreeDir)
	require.NoError(t, err)
	assert.True(t, info.IsWorktree)
	assert.Equal(t, mainGitDir, info.MainGitDir)
	// WorktreeGitDir should be resolved to absolute path
	assert.True(t, filepath.IsAbs(info.WorktreeGitDir))
}

func TestCreateContainerGitFile(t *testing.T) {
	tmpDir := t.TempDir()

	filePath, err := resolve.CreateContainerGitFile(tmpDir)
	require.NoError(t, err)
	assert.NotEmpty(t, filePath)

	// File should exist
	_, statErr := os.Stat(filePath)
	require.NoError(t, statErr, "override file should exist at %s", filePath)

	// Content should be "gitdir: /worktree-git\n"
	data, readErr := os.ReadFile(filePath)
	require.NoError(t, readErr)
	assert.Equal(t, "gitdir: /worktree-git\n", string(data))
}

func TestDetectGitWorktree_ContainerPathWithBackup(t *testing.T) {
	root := t.TempDir()

	// Simulate parent repo .git dir
	mainGitDir := filepath.Join(root, ".git")
	worktreeMetaDir := filepath.Join(mainGitDir, "worktrees", "test-branch")
	require.NoError(t, os.MkdirAll(worktreeMetaDir, 0755))

	// Create commondir file pointing to parent .git
	require.NoError(t, os.WriteFile(
		filepath.Join(worktreeMetaDir, "commondir"),
		[]byte("../..\n"),
		0644,
	))

	// Simulate worktree directory with .git file pointing to CONTAINER path
	worktreeDir := filepath.Join(root, "work", "branch", "features", "tt")
	require.NoError(t, os.MkdirAll(worktreeDir, 0755))

	// .git file has container-internal path (residue from previous container run)
	require.NoError(t, os.WriteFile(
		filepath.Join(worktreeDir, ".git"),
		[]byte("gitdir: /worktree-git\n"),
		0644,
	))

	// .git.tt-backup has the correct host path
	backupContent := "gitdir: " + worktreeMetaDir + "\n"
	require.NoError(t, os.WriteFile(
		filepath.Join(worktreeDir, ".git.tt-backup"),
		[]byte(backupContent),
		0644,
	))

	// DetectGitWorktree should restore from backup and succeed
	info, err := resolve.DetectGitWorktree(worktreeDir)
	require.NoError(t, err)
	assert.True(t, info.IsWorktree)
	assert.Equal(t, worktreeMetaDir, info.WorktreeGitDir)
	assert.Equal(t, mainGitDir, info.MainGitDir)

	// .git file should be restored on disk
	restoredData, readErr := os.ReadFile(filepath.Join(worktreeDir, ".git"))
	require.NoError(t, readErr)
	assert.Equal(t, backupContent, string(restoredData))
}

func TestDetectGitWorktree_ContainerPathWithoutBackup(t *testing.T) {
	root := t.TempDir()

	// Simulate worktree directory with .git file pointing to CONTAINER path
	worktreeDir := filepath.Join(root, "work", "branch", "features", "tt")
	require.NoError(t, os.MkdirAll(worktreeDir, 0755))

	// .git file has container-internal path, no backup exists
	require.NoError(t, os.WriteFile(
		filepath.Join(worktreeDir, ".git"),
		[]byte("gitdir: /worktree-git\n"),
		0644,
	))

	// DetectGitWorktree should return error (container path with no backup)
	_, err := resolve.DetectGitWorktree(worktreeDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no backup")
}
