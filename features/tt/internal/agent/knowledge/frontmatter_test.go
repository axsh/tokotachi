package knowledge

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadWriteFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")

	now := time.Now().UTC().Truncate(time.Second)
	meta := &KnowledgeFileMeta{
		KnowledgeID:    "api-error-responses",
		Title:          "API Error Responses",
		CategoryPath:   "error-handling",
		CreatedAt:      now,
		LastUpdated:    now,
		SourceEventIDs: []string{"E-001", "E-002"},
	}
	body := "# API Error Responses\n\nUse apierror types for all API handlers."

	err := WriteFrontmatter(path, meta, body)
	require.NoError(t, err)

	readMeta, readBody, err := ReadFrontmatter(path)
	require.NoError(t, err)

	assert.Equal(t, meta.KnowledgeID, readMeta.KnowledgeID)
	assert.Equal(t, meta.Title, readMeta.Title)
	assert.Equal(t, meta.CategoryPath, readMeta.CategoryPath)
	assert.Equal(t, meta.SourceEventIDs, readMeta.SourceEventIDs)
	assert.Contains(t, readBody, "apierror types")
}

func TestReadFrontmatter_NoDelimiter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nofm.md")
	require.NoError(t, os.WriteFile(path, []byte("No frontmatter here"), 0o644))

	_, _, err := ReadFrontmatter(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not start with frontmatter delimiter")
}

func TestReadFrontmatter_Empty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.md")
	require.NoError(t, os.WriteFile(path, []byte(""), 0o644))

	_, _, err := ReadFrontmatter(path)
	require.Error(t, err)
}

func TestReadWriteCategoryMeta(t *testing.T) {
	dir := t.TempDir()

	now := time.Now().UTC().Truncate(time.Second)
	meta := &CategoryMeta{
		CategoryID:  "error-handling",
		Title:       "Error Handling",
		Description: "Error handling patterns",
		CreatedAt:   now,
		LastUpdated: now,
	}

	err := WriteCategoryMeta(dir, meta)
	require.NoError(t, err)

	readMeta, err := ReadCategoryMeta(dir)
	require.NoError(t, err)

	assert.Equal(t, meta.CategoryID, readMeta.CategoryID)
	assert.Equal(t, meta.Title, readMeta.Title)
	assert.Equal(t, meta.Description, readMeta.Description)
}

func TestReadCategoryMeta_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadCategoryMeta(filepath.Join(dir, "nonexistent"))
	require.Error(t, err)
}
