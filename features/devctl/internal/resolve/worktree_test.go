package resolve_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/axsh/tokotachi/features/devctl/internal/resolve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveWorktree_NewPathStructure(t *testing.T) {
	root := t.TempDir()
	// New structure: work/<branch>/features/<feature>
	featureDir := filepath.Join(root, "work", "feat-x", "features", "devctl")
	require.NoError(t, os.MkdirAll(featureDir, 0755))

	path, err := resolve.Worktree(root, "devctl", "feat-x")
	require.NoError(t, err)
	assert.Equal(t, featureDir, path)
}

func TestResolveWorktree_NoFeature(t *testing.T) {
	root := t.TempDir()
	// No feature: work/<branch>/all/
	allDir := filepath.Join(root, "work", "feat-x", "all")
	require.NoError(t, os.MkdirAll(allDir, 0755))

	path, err := resolve.Worktree(root, "", "feat-x")
	require.NoError(t, err)
	assert.Equal(t, allDir, path)
}

func TestResolveWorktree_NoFeature_NotFound(t *testing.T) {
	root := t.TempDir()
	_, err := resolve.Worktree(root, "", "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestResolveWorktree_OldPathFallback(t *testing.T) {
	root := t.TempDir()
	// Old structure (backward compat): work/<feature>/<branch>
	oldDir := filepath.Join(root, "work", "devctl", "test-001")
	require.NoError(t, os.MkdirAll(oldDir, 0755))

	path, err := resolve.Worktree(root, "devctl", "test-001")
	require.NoError(t, err)
	assert.Equal(t, oldDir, path)
}

func TestResolveWorktree_OldFeatureOnlyFallback(t *testing.T) {
	root := t.TempDir()
	// Old structure (backward compat): work/<feature>
	featureDir := filepath.Join(root, "work", "test-feature")
	require.NoError(t, os.MkdirAll(featureDir, 0755))

	path, err := resolve.Worktree(root, "test-feature", "any-branch")
	require.NoError(t, err)
	assert.Equal(t, featureDir, path)
}

func TestResolveWorktree_NewPathTakesPriority(t *testing.T) {
	root := t.TempDir()
	// Both old and new paths exist; new should win
	newDir := filepath.Join(root, "work", "test-001", "features", "devctl")
	oldDir := filepath.Join(root, "work", "devctl", "test-001")
	require.NoError(t, os.MkdirAll(newDir, 0755))
	require.NoError(t, os.MkdirAll(oldDir, 0755))

	path, err := resolve.Worktree(root, "devctl", "test-001")
	require.NoError(t, err)
	assert.Equal(t, newDir, path)
}

func TestResolveWorktree_NotFound(t *testing.T) {
	root := t.TempDir()
	_, err := resolve.Worktree(root, "nonexistent", "any-branch")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
