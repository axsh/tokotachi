package resolve_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/resolve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveWorktree_Found(t *testing.T) {
	root := t.TempDir()
	// Unified structure: work/<branch>
	branchDir := filepath.Join(root, "work", "feat-x")
	require.NoError(t, os.MkdirAll(branchDir, 0755))
	// Create .git file to simulate valid worktree
	require.NoError(t, os.WriteFile(filepath.Join(branchDir, ".git"), []byte("gitdir: ../../.git/worktrees/feat-x\n"), 0644))

	path, err := resolve.Worktree(root, "feat-x")
	require.NoError(t, err)
	assert.Equal(t, branchDir, path)
}

func TestResolveWorktree_NotFound(t *testing.T) {
	root := t.TempDir()
	_, err := resolve.Worktree(root, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestResolveWorktree_GhostDirectory(t *testing.T) {
	root := t.TempDir()
	branchDir := filepath.Join(root, "work", "ghost-branch")
	require.NoError(t, os.MkdirAll(branchDir, 0755))
	// No .git file = ghost directory

	_, err := resolve.Worktree(root, "ghost-branch")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ghost directory")
}

func TestResolveWorktree_ValidWithGitFile(t *testing.T) {
	root := t.TempDir()
	branchDir := filepath.Join(root, "work", "valid-branch")
	require.NoError(t, os.MkdirAll(branchDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(branchDir, ".git"), []byte("gitdir: ../../.git/worktrees/valid-branch\n"), 0644))

	path, err := resolve.Worktree(root, "valid-branch")
	require.NoError(t, err)
	assert.Equal(t, branchDir, path)
}
