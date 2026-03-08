package resolve_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/axsh/tokotachi/features/devctl/internal/resolve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveWorktree_Found(t *testing.T) {
	root := t.TempDir()
	// Unified structure: work/<branch>
	branchDir := filepath.Join(root, "work", "feat-x")
	require.NoError(t, os.MkdirAll(branchDir, 0755))

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
