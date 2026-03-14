package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestCacheStore_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewCacheStore(tmpDir)

	updatedAt := "2026-03-10T19:00:00+09:00"
	catalogData := []byte("scaffolds:\n  root:\n    default: catalog/scaffolds/6/j/v/n.yaml\n")

	require.NoError(t, store.Save(updatedAt, catalogData))

	meta, data, err := store.Load()
	require.NoError(t, err)
	require.NotNil(t, meta)
	require.NotNil(t, data)
	assert.Equal(t, updatedAt, meta.UpdatedAt)
	assert.NotEmpty(t, meta.CachedAt)
	assert.Equal(t, catalogData, data)
}

func TestCacheStore_SaveCreatesDirectoryStructure(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewCacheStore(tmpDir)

	updatedAt := "2026-03-10T19:00:00+09:00"
	catalogData := []byte("test data")

	require.NoError(t, store.Save(updatedAt, catalogData))

	// Verify directory structure: .cache/repository_data/catalog.yaml/
	itemDir := store.CachePath()
	info, err := os.Stat(itemDir)
	require.NoError(t, err, "cache item directory should exist")
	assert.True(t, info.IsDir())

	// Verify meta.yaml exists and contains valid YAML
	metaContent, err := os.ReadFile(store.metaPath())
	require.NoError(t, err, "meta.yaml should exist")

	var meta CacheMeta
	require.NoError(t, yaml.Unmarshal(metaContent, &meta))
	assert.Equal(t, updatedAt, meta.UpdatedAt)
	assert.NotEmpty(t, meta.CachedAt)

	// Verify data file exists and contains raw content
	dataContent, err := os.ReadFile(store.dataPath())
	require.NoError(t, err, "data file should exist")
	assert.Equal(t, catalogData, dataContent)
}

func TestCacheStore_Load_NotExist(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewCacheStore(tmpDir)

	meta, data, err := store.Load()
	require.NoError(t, err)
	assert.Nil(t, meta)
	assert.Nil(t, data)
}

func TestCacheStore_IsValid(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewCacheStore(tmpDir)

	updatedAt := "2026-03-10T19:00:00+09:00"
	require.NoError(t, store.Save(updatedAt, []byte("test")))

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

func TestCacheStore_DataFileIntegrity(t *testing.T) {
	tests := []struct {
		name        string
		updatedAt   string
		catalogData []byte
	}{
		{"yaml_content", "2026-03-10T19:00:00+09:00",
			[]byte("scaffolds:\n  root:\n    default: path\n")},
		{"binary_content", "2026-03-15T00:00:00Z",
			[]byte{0x00, 0xFF, 0x89, 0x50, 0x4E, 0x47}},
		{"empty_content", "2026-03-15T00:00:00Z",
			[]byte{}},
		{"large_content", "2026-03-15T00:00:00Z",
			make([]byte, 1024*100)}, // 100KB
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			store := NewCacheStore(tmpDir)

			require.NoError(t, store.Save(tt.updatedAt, tt.catalogData))

			meta, data, err := store.Load()
			require.NoError(t, err)
			require.NotNil(t, meta)
			assert.Equal(t, tt.updatedAt, meta.UpdatedAt)
			assert.Equal(t, tt.catalogData, data, "data file should contain exact same bytes as input")
		})
	}
}
