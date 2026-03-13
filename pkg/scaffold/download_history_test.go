package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestLoadHistory_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewDownloadHistoryStore(tmpDir)

	history, err := store.Load()
	require.NoError(t, err)
	assert.NotNil(t, history)
	assert.Empty(t, history.History)
}

func TestSaveAndLoad_Roundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewDownloadHistoryStore(tmpDir)

	// Save
	history := &DownloadHistory{
		History: map[string]map[string]DownloadRecord{
			"root": {
				"default": {DownloadedAt: "2026-03-12T09:00:00Z"},
			},
		},
	}
	err := store.Save(history)
	require.NoError(t, err)

	// Verify file exists
	filePath := filepath.Join(tmpDir, DownloadHistoryDir, DownloadHistoryFileName)
	_, err = os.Stat(filePath)
	require.NoError(t, err, "downloaded.yaml should exist")

	// Load and compare
	loaded, err := store.Load()
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "2026-03-12T09:00:00Z", loaded.History["root"]["default"].DownloadedAt)
}

func TestIsDownloaded_Found(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewDownloadHistoryStore(tmpDir)

	// Pre-populate history
	history := &DownloadHistory{
		History: map[string]map[string]DownloadRecord{
			"root": {
				"default": {DownloadedAt: "2026-03-12T09:00:00Z"},
			},
		},
	}
	require.NoError(t, store.Save(history))

	assert.True(t, store.IsDownloaded("root", "default"))
}

func TestIsDownloaded_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewDownloadHistoryStore(tmpDir)

	// No history file
	assert.False(t, store.IsDownloaded("feature", "nonexistent"))

	// With history but different entry
	history := &DownloadHistory{
		History: map[string]map[string]DownloadRecord{
			"root": {
				"default": {DownloadedAt: "2026-03-12T09:00:00Z"},
			},
		},
	}
	require.NoError(t, store.Save(history))

	assert.False(t, store.IsDownloaded("feature", "nonexistent"))
	assert.False(t, store.IsDownloaded("root", "nonexistent"))
}

func TestRecordDownload_NewEntry(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewDownloadHistoryStore(tmpDir)

	err := store.RecordDownload("root", "default")
	require.NoError(t, err)

	// Verify recorded
	history, err := store.Load()
	require.NoError(t, err)
	require.Contains(t, history.History, "root")
	require.Contains(t, history.History["root"], "default")
	assert.NotEmpty(t, history.History["root"]["default"].DownloadedAt)
}

func TestRecordDownload_ExistingCategory(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewDownloadHistoryStore(tmpDir)

	// Record first entry
	require.NoError(t, store.RecordDownload("root", "default"))

	// Record second entry in same category
	require.NoError(t, store.RecordDownload("root", "another"))

	// Verify both exist
	history, err := store.Load()
	require.NoError(t, err)
	assert.Contains(t, history.History["root"], "default")
	assert.Contains(t, history.History["root"], "another")
}

func TestIsDynamic_WithTemplate(t *testing.T) {
	placement := &Placement{BaseDir: "features/{{feature_name}}"}
	assert.True(t, IsDynamic(placement))
}

func TestIsDynamic_WithoutTemplate(t *testing.T) {
	tests := []struct {
		name    string
		baseDir string
	}{
		{"dot", "."},
		{"static path", "features/myapp"},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			placement := &Placement{BaseDir: tt.baseDir}
			assert.False(t, IsDynamic(placement))
		})
	}
}

func TestSave_YAMLStructure(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewDownloadHistoryStore(tmpDir)

	require.NoError(t, store.RecordDownload("root", "default"))
	require.NoError(t, store.RecordDownload("project", "axsh-go-standard"))

	// Read raw YAML and verify structure
	filePath := filepath.Join(tmpDir, DownloadHistoryDir, DownloadHistoryFileName)
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, yaml.Unmarshal(data, &raw))

	// Top-level key should be "history"
	assert.Contains(t, raw, "history")
}
