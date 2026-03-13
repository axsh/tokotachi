package resolve_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/axsh/tokotachi/pkg/resolve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
