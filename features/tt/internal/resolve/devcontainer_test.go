package resolve_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/resolve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDevcontainerConfig_FromWorktreeJSON(t *testing.T) {
	root := t.TempDir()
	// Unified structure: work/<branch>/.devcontainer/
	dcDir := filepath.Join(root, "work", "main", ".devcontainer")
	require.NoError(t, os.MkdirAll(dcDir, 0755))

	content := `{
		"build": { "dockerfile": "Dockerfile" },
		"workspaceFolder": "/workspace",
		"containerEnv": { "GO111MODULE": "on" }
	}`
	require.NoError(t, os.WriteFile(filepath.Join(dcDir, "devcontainer.json"), []byte(content), 0644))

	cfg, err := resolve.LoadDevcontainerConfig(root, "feat-a", "main")
	require.NoError(t, err)
	assert.Equal(t, "Dockerfile", cfg.Build.Dockerfile)
	assert.Equal(t, "/workspace", cfg.WorkspaceFolder)
	assert.Equal(t, "on", cfg.ContainerEnv["GO111MODULE"])
}

func TestLoadDevcontainerConfig_WithImage(t *testing.T) {
	root := t.TempDir()
	dcDir := filepath.Join(root, "work", "dev", ".devcontainer")
	require.NoError(t, os.MkdirAll(dcDir, 0755))

	content := `{ "image": "golang:1.22" }`
	require.NoError(t, os.WriteFile(filepath.Join(dcDir, "devcontainer.json"), []byte(content), 0644))

	cfg, err := resolve.LoadDevcontainerConfig(root, "feat-b", "dev")
	require.NoError(t, err)
	assert.Equal(t, "golang:1.22", cfg.Image)
}

func TestLoadDevcontainerConfig_NotFound(t *testing.T) {
	root := t.TempDir()
	cfg, err := resolve.LoadDevcontainerConfig(root, "nonexistent", "main")
	require.NoError(t, err)
	assert.True(t, cfg.IsEmpty())
}

func TestLoadDevcontainerConfig_EmptyFeature(t *testing.T) {
	root := t.TempDir()
	cfg, err := resolve.LoadDevcontainerConfig(root, "", "main")
	require.NoError(t, err)
	assert.True(t, cfg.IsEmpty())
}

func TestLoadDevcontainerConfig_FallbackDockerfile(t *testing.T) {
	root := t.TempDir()
	worktreeDir := filepath.Join(root, "work", "main")
	require.NoError(t, os.MkdirAll(worktreeDir, 0755))

	require.NoError(t, os.WriteFile(
		filepath.Join(worktreeDir, "Dockerfile"),
		[]byte("FROM golang:1.22\n"),
		0644,
	))

	cfg, err := resolve.LoadDevcontainerConfig(root, "feat-c", "main")
	require.NoError(t, err)
	assert.True(t, cfg.HasDockerfile())
	assert.Contains(t, cfg.Build.Dockerfile, "Dockerfile")
	assert.Equal(t, "/workspace", cfg.WorkspaceFolder)
}

func TestLoadDevcontainerConfig_Priority(t *testing.T) {
	root := t.TempDir()
	worktreeDir := filepath.Join(root, "work", "main")
	dcDir := filepath.Join(worktreeDir, ".devcontainer")
	require.NoError(t, os.MkdirAll(dcDir, 0755))

	// Priority: devcontainer.json
	content := `{ "build": { "dockerfile": "Dockerfile" }, "workspaceFolder": "/ws" }`
	require.NoError(t, os.WriteFile(filepath.Join(dcDir, "devcontainer.json"), []byte(content), 0644))

	// Lower priority: root Dockerfile (should be ignored)
	require.NoError(t, os.WriteFile(filepath.Join(worktreeDir, "Dockerfile"), []byte("FROM alpine\n"), 0644))

	cfg, err := resolve.LoadDevcontainerConfig(root, "feat-d", "main")
	require.NoError(t, err)
	assert.Equal(t, "Dockerfile", cfg.Build.Dockerfile)
	assert.Equal(t, "/ws", cfg.WorkspaceFolder)
}

func TestLoadDevcontainerConfig_WithMountsAndUser(t *testing.T) {
	root := t.TempDir()
	dcDir := filepath.Join(root, "work", "main", ".devcontainer")
	require.NoError(t, os.MkdirAll(dcDir, 0755))

	content := `{
		"name": "my-dev",
		"build": { "dockerfile": "./Dockerfile" },
		"workspaceFolder": "/workspace",
		"mounts": [
			"source=/var/run/docker.sock,target=/var/run/docker.sock,type=bind"
		],
		"remoteUser": "root",
		"containerEnv": { "TERM": "xterm-256color" }
	}`
	require.NoError(t, os.WriteFile(filepath.Join(dcDir, "devcontainer.json"), []byte(content), 0644))

	cfg, err := resolve.LoadDevcontainerConfig(root, "feat-e", "main")
	require.NoError(t, err)
	assert.Equal(t, "my-dev", cfg.Name)
	assert.Equal(t, "./Dockerfile", cfg.Build.Dockerfile)
	assert.Equal(t, "root", cfg.RemoteUser)
	assert.Len(t, cfg.Mounts, 1)
	assert.Contains(t, cfg.Mounts[0], "docker.sock")
	assert.Equal(t, "xterm-256color", cfg.ContainerEnv["TERM"])
	assert.NotEmpty(t, cfg.ConfigDir())
}

func TestLoadDevcontainerConfig_FeatureDir(t *testing.T) {
	root := t.TempDir()
	// Create devcontainer.json in features/<feature>/
	dcDir := filepath.Join(root, "features", "tt", ".devcontainer")
	require.NoError(t, os.MkdirAll(dcDir, 0755))

	content := `{
		"name": "tt-dev",
		"build": { "dockerfile": "./Dockerfile" },
		"workspaceFolder": "/workspace"
	}`
	require.NoError(t, os.WriteFile(filepath.Join(dcDir, "devcontainer.json"), []byte(content), 0644))

	cfg, err := resolve.LoadDevcontainerConfig(root, "tt", "test-001")
	require.NoError(t, err)
	assert.Equal(t, "tt-dev", cfg.Name)
	assert.Equal(t, "./Dockerfile", cfg.Build.Dockerfile)
	assert.Contains(t, cfg.ConfigDir(), filepath.Join("features", "tt", ".devcontainer"))
}

func TestLoadDevcontainerConfig_FeatureDirPriority(t *testing.T) {
	root := t.TempDir()

	// Priority 1: features/<feature>/.devcontainer/
	featureDC := filepath.Join(root, "features", "tt", ".devcontainer")
	require.NoError(t, os.MkdirAll(featureDC, 0755))
	featureContent := `{ "name": "from-features", "workspaceFolder": "/ws-feature" }`
	require.NoError(t, os.WriteFile(filepath.Join(featureDC, "devcontainer.json"), []byte(featureContent), 0644))

	// Priority 2: work/<branch>/.devcontainer/
	worktreeDC := filepath.Join(root, "work", "test-001", ".devcontainer")
	require.NoError(t, os.MkdirAll(worktreeDC, 0755))
	worktreeContent := `{ "name": "from-worktree", "workspaceFolder": "/ws-worktree" }`
	require.NoError(t, os.WriteFile(filepath.Join(worktreeDC, "devcontainer.json"), []byte(worktreeContent), 0644))

	cfg, err := resolve.LoadDevcontainerConfig(root, "tt", "test-001")
	require.NoError(t, err)
	// features/ should win
	assert.Equal(t, "from-features", cfg.Name)
	assert.Equal(t, "/ws-feature", cfg.WorkspaceFolder)
}
