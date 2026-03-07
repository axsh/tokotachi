package resolve_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/escape-dev/devctl/internal/resolve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadGlobalConfig_Defaults(t *testing.T) {
	root := t.TempDir()
	cfg, err := resolve.LoadGlobalConfig(root)
	require.NoError(t, err)
	assert.Equal(t, "cursor", cfg.DefaultEditor)
	assert.Equal(t, "work", cfg.WorkDir)
	assert.Equal(t, "docker-local", cfg.DefaultContainerMode)
}

func TestLoadGlobalConfig_FromFile(t *testing.T) {
	root := t.TempDir()
	content := `
project_name: testproj
default_editor: code
work_dir: workspaces
default_container_mode: devcontainer
`
	require.NoError(t, os.WriteFile(filepath.Join(root, ".devrc.yaml"), []byte(content), 0644))

	cfg, err := resolve.LoadGlobalConfig(root)
	require.NoError(t, err)
	assert.Equal(t, "testproj", cfg.ProjectName)
	assert.Equal(t, "code", cfg.DefaultEditor)
	assert.Equal(t, "workspaces", cfg.WorkDir)
	assert.Equal(t, "devcontainer", cfg.DefaultContainerMode)
}

func TestLoadFeatureConfig_NotFound(t *testing.T) {
	root := t.TempDir()
	cfg, err := resolve.LoadFeatureConfig(root, "nonexistent")
	require.NoError(t, err)
	assert.Zero(t, cfg)
}

func TestLoadFeatureConfig_FromWorkDir(t *testing.T) {
	root := t.TempDir()
	featureDir := filepath.Join(root, "work", "my-feature")
	require.NoError(t, os.MkdirAll(featureDir, 0755))
	content := `
dev:
  editor_default: ag
  ssh_supported: true
  shell: bash
`
	require.NoError(t, os.WriteFile(filepath.Join(featureDir, "feature.yaml"), []byte(content), 0644))

	cfg, err := resolve.LoadFeatureConfig(root, "my-feature")
	require.NoError(t, err)
	assert.Equal(t, "ag", cfg.Dev.EditorDefault)
	assert.True(t, cfg.Dev.SSHSupported)
	assert.Equal(t, "bash", cfg.Dev.Shell)
}
