package resolve_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/escape-dev/devctl/internal/resolve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDevcontainerConfig_FromJSON(t *testing.T) {
	root := t.TempDir()
	dcDir := filepath.Join(root, "work", "feat-a", ".devcontainer")
	require.NoError(t, os.MkdirAll(dcDir, 0755))

	content := `{
		"build": { "dockerfile": "Dockerfile" },
		"workspaceFolder": "/workspace",
		"containerEnv": { "GO111MODULE": "on" }
	}`
	require.NoError(t, os.WriteFile(filepath.Join(dcDir, "devcontainer.json"), []byte(content), 0644))

	cfg, err := resolve.LoadDevcontainerConfig(root, "feat-a")
	require.NoError(t, err)
	assert.Equal(t, "Dockerfile", cfg.Build.Dockerfile)
	assert.Equal(t, "/workspace", cfg.WorkspaceFolder)
	assert.Equal(t, "on", cfg.ContainerEnv["GO111MODULE"])
}

func TestLoadDevcontainerConfig_WithImage(t *testing.T) {
	root := t.TempDir()
	dcDir := filepath.Join(root, "work", "feat-b", ".devcontainer")
	require.NoError(t, os.MkdirAll(dcDir, 0755))

	content := `{ "image": "golang:1.22" }`
	require.NoError(t, os.WriteFile(filepath.Join(dcDir, "devcontainer.json"), []byte(content), 0644))

	cfg, err := resolve.LoadDevcontainerConfig(root, "feat-b")
	require.NoError(t, err)
	assert.Equal(t, "golang:1.22", cfg.Image)
}

func TestLoadDevcontainerConfig_NotFound(t *testing.T) {
	root := t.TempDir()
	cfg, err := resolve.LoadDevcontainerConfig(root, "nonexistent")
	require.NoError(t, err)
	assert.True(t, cfg.IsEmpty())
}

func TestLoadDevcontainerConfig_FallbackDockerfile(t *testing.T) {
	root := t.TempDir()
	worktree := filepath.Join(root, "work", "feat-c")
	require.NoError(t, os.MkdirAll(worktree, 0755))

	require.NoError(t, os.WriteFile(
		filepath.Join(worktree, "Dockerfile"),
		[]byte("FROM golang:1.22\n"),
		0644,
	))

	cfg, err := resolve.LoadDevcontainerConfig(root, "feat-c")
	require.NoError(t, err)
	assert.True(t, cfg.HasDockerfile())
	assert.Contains(t, cfg.Build.Dockerfile, "Dockerfile")
	assert.Equal(t, "/workspace", cfg.WorkspaceFolder)
}

func TestLoadDevcontainerConfig_Priority(t *testing.T) {
	root := t.TempDir()
	worktree := filepath.Join(root, "work", "feat-d")
	dcDir := filepath.Join(worktree, ".devcontainer")
	require.NoError(t, os.MkdirAll(dcDir, 0755))

	// Priority 1: devcontainer.json
	content := `{ "build": { "dockerfile": "Dockerfile" }, "workspaceFolder": "/ws" }`
	require.NoError(t, os.WriteFile(filepath.Join(dcDir, "devcontainer.json"), []byte(content), 0644))

	// Priority 3: root Dockerfile (should be ignored)
	require.NoError(t, os.WriteFile(filepath.Join(worktree, "Dockerfile"), []byte("FROM alpine\n"), 0644))

	cfg, err := resolve.LoadDevcontainerConfig(root, "feat-d")
	require.NoError(t, err)
	assert.Equal(t, "Dockerfile", cfg.Build.Dockerfile)
	assert.Equal(t, "/ws", cfg.WorkspaceFolder)
}
