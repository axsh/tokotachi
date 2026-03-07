package state_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/axsh/tokotachi/features/devctl/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatePath(t *testing.T) {
	got := state.StatePath("/repo", "devctl", "test-001")
	expected := filepath.Join("/repo", "work", "devctl", "test-001.state.yaml")
	assert.Equal(t, expected, got)
}

func TestSave_Load_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.state.yaml")

	original := state.StateFile{
		Feature:       "devctl",
		Branch:        "test-001",
		CreatedAt:     time.Date(2026, 3, 7, 14, 0, 0, 0, time.UTC),
		ContainerMode: "docker-local",
		Editor:        "cursor",
		Status:        state.StatusActive,
	}

	err := state.Save(path, original)
	require.NoError(t, err)

	loaded, err := state.Load(path)
	require.NoError(t, err)
	assert.Equal(t, original.Feature, loaded.Feature)
	assert.Equal(t, original.Branch, loaded.Branch)
	assert.Equal(t, original.ContainerMode, loaded.ContainerMode)
	assert.Equal(t, original.Editor, loaded.Editor)
	assert.Equal(t, original.Status, loaded.Status)
	assert.True(t, original.CreatedAt.Equal(loaded.CreatedAt))
}

func TestLoad_NotFound(t *testing.T) {
	_, err := state.Load("/nonexistent/path/file.yaml")
	require.Error(t, err)
}

func TestRemove_Existing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.state.yaml")

	err := state.Save(path, state.StateFile{Feature: "devctl", Branch: "test"})
	require.NoError(t, err)

	err = state.Remove(path)
	require.NoError(t, err)
}

func TestRemove_NotFound(t *testing.T) {
	err := state.Remove("/nonexistent/path/file.yaml")
	require.NoError(t, err)
}
