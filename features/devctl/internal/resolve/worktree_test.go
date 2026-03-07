package resolve_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/escape-dev/devctl/internal/resolve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveWorktree(t *testing.T) {
	root := t.TempDir()
	featureDir := filepath.Join(root, "work", "test-feature")
	require.NoError(t, os.MkdirAll(featureDir, 0755))

	t.Run("fallback to feature-only path", func(t *testing.T) {
		path, err := resolve.Worktree(root, "test-feature", "any-branch")
		require.NoError(t, err)
		assert.Equal(t, featureDir, path)
	})

	t.Run("non-existing worktree", func(t *testing.T) {
		_, err := resolve.Worktree(root, "nonexistent", "any-branch")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestResolveWorktree_FeatureBranch(t *testing.T) {
	root := t.TempDir()
	branchDir := filepath.Join(root, "work", "devctl", "test-001")
	require.NoError(t, os.MkdirAll(branchDir, 0755))

	path, err := resolve.Worktree(root, "devctl", "test-001")
	require.NoError(t, err)
	assert.Equal(t, branchDir, path)
}

func TestResolveWorktree_FallbackFeatureOnly(t *testing.T) {
	root := t.TempDir()
	featureDir := filepath.Join(root, "work", "devctl")
	require.NoError(t, os.MkdirAll(featureDir, 0755))

	// No work/devctl/test-001 dir, but work/devctl exists
	path, err := resolve.Worktree(root, "devctl", "test-001")
	require.NoError(t, err)
	assert.Equal(t, featureDir, path)
}
