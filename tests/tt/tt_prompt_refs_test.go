package integration_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type promptRefsCatalog struct {
	Refs []struct {
		File string `json:"file"`
		Kind string `json:"kind"`
		ID   string `json:"id"`
		Ref  string `json:"ref"`
	} `json:"refs"`
}

func decodePromptRefs(t *testing.T, stdout string) promptRefsCatalog {
	t.Helper()
	var catalog promptRefsCatalog
	require.NoError(t, json.Unmarshal([]byte(stdout), &catalog), "stdout: %s", stdout)
	return catalog
}

func refMap(catalog promptRefsCatalog) map[string]string {
	out := make(map[string]string, len(catalog.Refs))
	for _, e := range catalog.Refs {
		out[e.ID] = e.Ref
	}
	return out
}

func TestPromptRefs_JSONContainsPolicyAndProcedure(t *testing.T) {
	tmp := t.TempDir()
	copyCatalogTemplateFixture(t, tmp)

	stdout, stderr, code := runTTInDir(t, tmp, "prompt", "refs")
	require.Equal(t, 0, code, "stderr: %s", stderr)

	assert.NotContains(t, stdout, ".claude/")
	assert.NotContains(t, stdout, ".claude/rules/")
	assert.NotContains(t, stdout, ".agent/workflows")

	catalog := decodePromptRefs(t, stdout)
	refs := refMap(catalog)

	require.Equal(t, "{{policy:coding-rules}}", refs["coding-rules"])
	require.Equal(t, "{{procedure:build-pipeline}}", refs["build-pipeline"])

	for _, e := range catalog.Refs {
		assert.NotContains(t, e.File, "/")
		assert.NotContains(t, e.File, "\\")
		assert.Equal(t, "{{"+e.Kind+":"+e.ID+"}}", e.Ref)
	}
}

func TestPromptRefs_RefStableWithWorkspaceFlag(t *testing.T) {
	tmp := t.TempDir()
	copyCatalogTemplateFixture(t, tmp)

	stdoutDefault, stderrDefault, codeDefault := runTTInDir(t, tmp, "prompt", "refs")
	require.Equal(t, 0, codeDefault, "stderr: %s", stderrDefault)

	stdoutWS, stderrWS, codeWS := runTTInDir(t, tmp, "prompt", "refs", "--workspace", tmp)
	require.Equal(t, 0, codeWS, "stderr: %s", stderrWS)

	assert.Equal(t, refMap(decodePromptRefs(t, stdoutDefault)), refMap(decodePromptRefs(t, stdoutWS)))
}

func TestPromptRefs_MissingProjectFails(t *testing.T) {
	tmp := t.TempDir()
	stdout, stderr, code := runTTInDir(t, tmp, "prompt", "refs")
	require.NotEqual(t, 0, code, "expected failure without project.yaml; stdout=%s stderr=%s", stdout, stderr)
}
