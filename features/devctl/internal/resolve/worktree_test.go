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

	t.Run("existing worktree", func(t *testing.T) {
		path, err := resolve.Worktree(root, "test-feature")
		require.NoError(t, err)
		assert.Equal(t, featureDir, path)
	})

	t.Run("non-existing worktree", func(t *testing.T) {
		_, err := resolve.Worktree(root, "nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}
