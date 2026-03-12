package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheStore_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()

	store := NewCacheStore(tmpDir)
	catalog := &CachedCatalog{
		UpdatedAt:   "2026-03-10T19:00:00+09:00",
		CatalogData: []byte("scaffolds:\n  root:\n    default: catalog/scaffolds/6/j/v/n.yaml\n"),
	}

	require.NoError(t, store.Save(catalog))

	loaded, err := store.Load()
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, catalog.UpdatedAt, loaded.UpdatedAt)
	assert.Equal(t, catalog.CatalogData, loaded.CatalogData)
}

func TestCacheStore_Load_NotExist(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewCacheStore(tmpDir)

	loaded, err := store.Load()
	require.NoError(t, err)
	assert.Nil(t, loaded)
}

func TestCacheStore_IsValid(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewCacheStore(tmpDir)

	catalog := &CachedCatalog{
		UpdatedAt:   "2026-03-10T19:00:00+09:00",
		CatalogData: []byte("test"),
	}
	require.NoError(t, store.Save(catalog))

	// Same updated_at → valid
	assert.True(t, store.IsValid("2026-03-10T19:00:00+09:00"))

	// Different updated_at → invalid
	assert.False(t, store.IsValid("2026-03-11T00:00:00+09:00"))
}

func TestCacheStore_EnsureGitignore(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .kotoshiro/tokotachi/ directory
	kotoshiroDir := filepath.Join(tmpDir, ".kotoshiro", "tokotachi")
	require.NoError(t, os.MkdirAll(kotoshiroDir, 0o755))

	store := NewCacheStore(tmpDir)
	require.NoError(t, store.EnsureGitignore())

	// Check that .gitignore was created with .cache/ entry
	gitignorePath := filepath.Join(kotoshiroDir, ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), ".cache/")
}
