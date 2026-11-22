package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScaffoldManifestUpdateSuite_CatalogAndRepo(t *testing.T) {
	root := projectRoot()
	baseCap := filepath.Join(root, "catalog", "originals", "axsh", "go-standard-project",
		"base", "prompts", "manifest", "code_content", "capabilities")
	repoCap := filepath.Join(root, "prompts", "manifest", "code_content", "capabilities")

	for _, dir := range []string{baseCap, repoCap} {
		mustContainFile(t, filepath.Join(dir, "scaffold-manifest-update.md"),
			"--update", "instruction.md", "doc-relations-", "Do not discard or rewrite project-specific")
		mustContainFile(t, filepath.Join(dir, "prompt-manifest-update.md"),
			"scaffold-manifest-update", "uses_capabilities")
		mustContainFile(t, filepath.Join(dir, "acs-external-agent-sample.md"),
			"external-agent")
	}

	// uid present on scaffold-manifest-update
	data, err := os.ReadFile(filepath.Join(repoCap, "scaffold-manifest-update.md"))
	require.NoError(t, err)
	require.Contains(t, string(data), "uid:")

	schema := filepath.Join(root, "prompts", "manifest", "schemas", "capability.schema.json")
	sdata, err := os.ReadFile(schema)
	require.NoError(t, err)
	require.Contains(t, string(sdata), `"uid"`)
}

func TestScaffoldHelpMentionsUpdate(t *testing.T) {
	stdout, stderr, code := runTT(t, "scaffold", "--help")
	require.Equalf(t, 0, code, "stderr=%s", stderr)
	require.Contains(t, stdout+stderr, "update")
}

func mustContainFile(t *testing.T, path string, needles ...string) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoErrorf(t, err, "missing %s", path)
	text := string(data)
	for _, n := range needles {
		require.Truef(t, strings.Contains(text, n), "%s missing %q", path, n)
	}
}
