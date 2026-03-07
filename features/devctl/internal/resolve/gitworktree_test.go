package resolve_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/axsh/tokotachi/features/devctl/internal/resolve"
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
