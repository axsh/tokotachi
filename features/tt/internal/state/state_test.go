package state_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/axsh/tokotachi/features/devctl/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatePath(t *testing.T) {
	got := state.StatePath("/repo", "test-001")
	expected := filepath.Join("/repo", "work", "test-001.state.yaml")
	assert.Equal(t, expected, got)
}

func TestSave_Load_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.state.yaml")

	original := state.StateFile{
		Branch:    "test-001",
		CreatedAt: time.Date(2026, 3, 7, 14, 0, 0, 0, time.UTC),
		Features: map[string]state.FeatureState{
			"devctl": {
				Status:    state.StatusActive,
				StartedAt: time.Date(2026, 3, 7, 14, 0, 0, 0, time.UTC),
				Connectivity: state.Connectivity{
					Docker: state.DockerConnectivity{
						Enabled:       true,
						ContainerName: "devctl-devctl",
						Devcontainer:  true,
					},
					SSH: state.SSHConnectivity{Enabled: false},
				},
			},
		},
	}

	err := state.Save(path, original)
	require.NoError(t, err)

	loaded, err := state.Load(path)
	require.NoError(t, err)
	assert.Equal(t, original.Branch, loaded.Branch)
	assert.True(t, original.CreatedAt.Equal(loaded.CreatedAt))
	require.Contains(t, loaded.Features, "devctl")
	fs := loaded.Features["devctl"]
	assert.Equal(t, state.StatusActive, fs.Status)
	assert.True(t, fs.Connectivity.Docker.Enabled)
	assert.Equal(t, "devctl-devctl", fs.Connectivity.Docker.ContainerName)
	assert.True(t, fs.Connectivity.Docker.Devcontainer)
	assert.False(t, fs.Connectivity.SSH.Enabled)
}

func TestSave_Load_NoFeatures(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.state.yaml")

	original := state.StateFile{
		Branch:    "test-001",
		CreatedAt: time.Date(2026, 3, 7, 14, 0, 0, 0, time.UTC),
	}

	err := state.Save(path, original)
	require.NoError(t, err)

	loaded, err := state.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "test-001", loaded.Branch)
	assert.Empty(t, loaded.Features)
}

func TestSetFeature_NewEntry(t *testing.T) {
	sf := state.StateFile{Branch: "test"}
	sf.SetFeature("devctl", state.FeatureState{
		Status:    state.StatusActive,
		StartedAt: time.Now(),
		Connectivity: state.Connectivity{
			Docker: state.DockerConnectivity{
				Enabled:       true,
				ContainerName: "proj-devctl",
			},
		},
	})
	require.Contains(t, sf.Features, "devctl")
	assert.Equal(t, state.StatusActive, sf.Features["devctl"].Status)
	assert.Equal(t, "proj-devctl", sf.Features["devctl"].Connectivity.Docker.ContainerName)
}

func TestSetFeature_UpdateExisting(t *testing.T) {
	sf := state.StateFile{
		Branch: "test",
		Features: map[string]state.FeatureState{
			"devctl": {
				Status: state.StatusActive,
				Connectivity: state.Connectivity{
					Docker: state.DockerConnectivity{Enabled: true, ContainerName: "old"},
				},
			},
		},
	}
	// Overwrite with SetFeature
	sf.SetFeature("devctl", state.FeatureState{
		Status: state.StatusStopped,
		Connectivity: state.Connectivity{
			Docker: state.DockerConnectivity{Enabled: true, ContainerName: "new"},
		},
	})
	assert.Equal(t, state.StatusStopped, sf.Features["devctl"].Status)
	assert.Equal(t, "new", sf.Features["devctl"].Connectivity.Docker.ContainerName)
}

func TestRemoveFeature(t *testing.T) {
	sf := state.StateFile{
		Branch: "test",
		Features: map[string]state.FeatureState{
			"devctl":  {Status: state.StatusActive},
			"backend": {Status: state.StatusActive},
		},
	}
	sf.RemoveFeature("devctl")
	assert.NotContains(t, sf.Features, "devctl")
	assert.Contains(t, sf.Features, "backend")
}

func TestRemoveFeature_LastOne(t *testing.T) {
	sf := state.StateFile{
		Branch: "test",
		Features: map[string]state.FeatureState{
			"devctl": {Status: state.StatusActive},
		},
	}
	sf.RemoveFeature("devctl")
	assert.Empty(t, sf.Features)
}

func TestUpdateFeatureStatus(t *testing.T) {
	sf := state.StateFile{
		Branch: "test",
		Features: map[string]state.FeatureState{
			"devctl": {
				Status: state.StatusActive,
				Connectivity: state.Connectivity{
					Docker: state.DockerConnectivity{
						Enabled:       true,
						ContainerName: "proj-devctl",
						Devcontainer:  true,
					},
					SSH: state.SSHConnectivity{Enabled: true, Endpoint: "localhost:2222"},
				},
			},
		},
	}
	err := sf.UpdateFeatureStatus("devctl", state.StatusStopped)
	require.NoError(t, err)
	// Status changed
	assert.Equal(t, state.StatusStopped, sf.Features["devctl"].Status)
	// Connectivity preserved
	assert.True(t, sf.Features["devctl"].Connectivity.Docker.Enabled)
	assert.Equal(t, "proj-devctl", sf.Features["devctl"].Connectivity.Docker.ContainerName)
	assert.True(t, sf.Features["devctl"].Connectivity.Docker.Devcontainer)
	assert.True(t, sf.Features["devctl"].Connectivity.SSH.Enabled)
	assert.Equal(t, "localhost:2222", sf.Features["devctl"].Connectivity.SSH.Endpoint)
}

func TestUpdateFeatureStatus_NotFound(t *testing.T) {
	sf := state.StateFile{Branch: "test"}
	err := sf.UpdateFeatureStatus("nonexistent", state.StatusStopped)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestHasActiveFeatures_True(t *testing.T) {
	sf := state.StateFile{
		Features: map[string]state.FeatureState{
			"devctl": {Status: state.StatusActive},
		},
	}
	assert.True(t, sf.HasActiveFeatures())
}

func TestHasActiveFeatures_False(t *testing.T) {
	sf := state.StateFile{
		Features: map[string]state.FeatureState{
			"devctl": {Status: state.StatusStopped},
		},
	}
	assert.False(t, sf.HasActiveFeatures())
}

func TestHasActiveFeatures_Empty(t *testing.T) {
	sf := state.StateFile{}
	assert.False(t, sf.HasActiveFeatures())
}

func TestActiveFeatureNames(t *testing.T) {
	sf := state.StateFile{
		Features: map[string]state.FeatureState{
			"devctl":  {Status: state.StatusActive},
			"backend": {Status: state.StatusStopped},
			"api":     {Status: state.StatusActive},
		},
	}
	names := sf.ActiveFeatureNames()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "devctl")
	assert.Contains(t, names, "api")
}

func TestLoad_NotFound(t *testing.T) {
	_, err := state.Load("/nonexistent/path/file.yaml")
	require.Error(t, err)
}

func TestRemove_Existing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.state.yaml")

	err := state.Save(path, state.StateFile{Branch: "test"})
	require.NoError(t, err)

	err = state.Remove(path)
	require.NoError(t, err)
}

func TestRemove_NotFound(t *testing.T) {
	err := state.Remove("/nonexistent/path/file.yaml")
	require.NoError(t, err)
}

func TestStateFile_CodeStatus_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.state.yaml")

	prTime := time.Date(2026, 3, 8, 10, 30, 0, 0, time.UTC)
	checkedTime := time.Date(2026, 3, 9, 1, 0, 0, 0, time.UTC)

	original := state.StateFile{
		Branch:    "feat-test",
		CreatedAt: time.Date(2026, 3, 7, 14, 0, 0, 0, time.UTC),
		CodeStatus: &state.CodeStatus{
			Status:        state.CodeStatusPR,
			PRCreatedAt:   &prTime,
			LastCheckedAt: &checkedTime,
		},
	}

	err := state.Save(path, original)
	require.NoError(t, err)

	loaded, err := state.Load(path)
	require.NoError(t, err)
	require.NotNil(t, loaded.CodeStatus)
	assert.Equal(t, state.CodeStatusPR, loaded.CodeStatus.Status)
	require.NotNil(t, loaded.CodeStatus.PRCreatedAt)
	assert.True(t, prTime.Equal(*loaded.CodeStatus.PRCreatedAt))
	require.NotNil(t, loaded.CodeStatus.LastCheckedAt)
	assert.True(t, checkedTime.Equal(*loaded.CodeStatus.LastCheckedAt))
}

func TestStateFile_BackwardCompat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.state.yaml")

	// Save a StateFile without CodeStatus (simulating an existing old file)
	original := state.StateFile{
		Branch:    "legacy-branch",
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Features: map[string]state.FeatureState{
			"devctl": {Status: state.StatusActive},
		},
	}

	err := state.Save(path, original)
	require.NoError(t, err)

	loaded, err := state.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "legacy-branch", loaded.Branch)
	assert.Nil(t, loaded.CodeStatus)
	require.Contains(t, loaded.Features, "devctl")
}

func TestStateFile_CodeStatus_OmitEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.state.yaml")

	original := state.StateFile{
		Branch:    "no-code-status",
		CreatedAt: time.Date(2026, 3, 7, 14, 0, 0, 0, time.UTC),
		// CodeStatus is nil
	}

	err := state.Save(path, original)
	require.NoError(t, err)

	// Read raw YAML and verify code_status key is absent
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "code_status")

	// Load should still work
	loaded, err := state.Load(path)
	require.NoError(t, err)
	assert.Nil(t, loaded.CodeStatus)
}
