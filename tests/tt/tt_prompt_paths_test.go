//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func copyCompilerValidFixture(t *testing.T, dst string) {
	t.Helper()
	src := filepath.Join(projectRoot(), "features", "tt", "internal", "prompt", "compiler", "testdata", "valid")
	require.NoError(t, os.MkdirAll(dst, 0o755))
	copyDirContents(t, src, dst)
}

func copyDirContents(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	require.NoError(t, err)
	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			require.NoError(t, os.MkdirAll(dstPath, 0o755))
			copyDirContents(t, srcPath, dstPath)
			continue
		}
		data, err := os.ReadFile(srcPath)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(dstPath, data, 0o644))
	}
}

func TestPromptPaths_BackwardCompatible(t *testing.T) {
	tmp := t.TempDir()
	copyCompilerValidFixture(t, tmp)

	stdout, stderr, code := runTTInDir(t, tmp, "prompt", "compile", "--dry-run", "--target", "cursor")
	require.Equal(t, 0, code, "stderr: %s", stderr)
	assert.Contains(t, stdout, "resolved manifest")
}

func TestPromptPaths_CustomBuildDir(t *testing.T) {
	tmp := t.TempDir()
	copyCompilerValidFixture(t, tmp)

	_, stderr, code := runTTInDir(t, tmp,
		"prompt", "compile",
		"--target", "cursor",
		"--build-dir", ".cache/tt-dist",
	)
	require.Equal(t, 0, code, "stderr: %s", stderr)

	customResolved := filepath.Join(tmp, ".cache", "tt-dist", "manifest.resolved.yaml")
	_, err := os.Stat(customResolved)
	require.NoError(t, err, "expected resolved manifest under custom build dir")
}

func TestPromptPaths_DeployRootSeparate(t *testing.T) {
	workspace := t.TempDir()
	deployRoot := t.TempDir()
	copyCompilerValidFixture(t, workspace)

	_, stderr, code := runTTInDir(t, workspace,
		"prompt", "deploy",
		"--force",
		"--target", "cursor",
		"--deploy-root", deployRoot,
	)
	require.Equal(t, 0, code, "stderr: %s", stderr)

	deployRules := filepath.Join(deployRoot, ".cursor", "rules")
	entries, err := os.ReadDir(deployRules)
	require.NoError(t, err)
	assert.Greater(t, len(entries), 0)

	workspaceRules := filepath.Join(workspace, ".cursor", "rules")
	_, err = os.Stat(workspaceRules)
	assert.True(t, os.IsNotExist(err), "workspace should not receive deploy output")
}

func TestPromptPaths_WrapperScriptPassesFlags(t *testing.T) {
	stdout, stderr, code := runTT(t, "prompt", "deploy", "--help")
	require.Equal(t, 0, code, "stderr: %s", stderr)
	assert.Contains(t, stdout+stderr, "--deploy-root")
}

func TestPromptPaths_HelpShowsPathExpressions(t *testing.T) {
	stdout, stderr, code := runTT(t, "prompt", "compile", "--help")
	require.Equal(t, 0, code, "stderr: %s", stderr)
	combined := stdout + stderr
	assert.Contains(t, combined, "{workspace} + {prompts-dir}")
	assert.Contains(t, combined, "--build-dir")
}
