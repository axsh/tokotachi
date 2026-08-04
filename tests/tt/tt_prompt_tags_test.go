//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTaggedEntities(t *testing.T, root string) {
	t.Helper()
	capDir := filepath.Join(root, "prompts", "manifest", "code_content", "capabilities")
	procDir := filepath.Join(root, "prompts", "manifest", "code_content", "procedures")
	require.NoError(t, os.MkdirAll(capDir, 0o755))
	require.NoError(t, os.MkdirAll(procDir, 0o755))

	baselineCap := `---
apiVersion: agent.meta/v1
kind: capability
id: baseline-cap
title: Baseline Cap
description: baseline only
body: inline
tags:
  - baseline
---

# Baseline Cap
`
	testCap := `---
apiVersion: agent.meta/v1
kind: capability
id: test-cap
title: Test Cap
description: test only
body: inline
tags:
  - test
---

# Test Cap
`
	proc := `---
apiVersion: agent.meta/v1
kind: procedure
id: baseline-proc
title: Baseline Proc
trigger:
  command: baseline-proc
uses_capabilities:
  - test-cap
tags:
  - baseline
---

# Baseline Proc
`
	require.NoError(t, os.WriteFile(filepath.Join(capDir, "baseline-cap.md"), []byte(baselineCap), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(capDir, "test-cap.md"), []byte(testCap), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(procDir, "baseline-proc.md"), []byte(proc), 0o644))

	// Extend project.yaml sources for capabilities/procedures and copy schemas from prod-like catalog template if needed.
	projectPath := filepath.Join(root, "prompts", "manifest", "project.yaml")
	proj := `version: 1
project_id: test-project
sources:
  policies: "prompts/manifest/code_content/policies/**/*.yaml"
  capabilities: "prompts/manifest/code_content/capabilities/**/*.md"
  procedures: "prompts/manifest/code_content/procedures/**/*.md"
  memory_docs: "prompts/memory/**/*.md"
outputs:
  resolved_manifest: tmp/dist/manifest.resolved.yaml
defaults:
  language: ja
  generated_banner: true
  build_dir: tmp/dist/
`
	require.NoError(t, os.WriteFile(projectPath, []byte(proj), 0o644))

	schemaSrc := filepath.Join(projectRoot(), "prompts", "manifest", "schemas")
	schemaDst := filepath.Join(root, "prompts", "manifest", "schemas")
	require.NoError(t, os.MkdirAll(schemaDst, 0o755))
	entries, err := os.ReadDir(schemaSrc)
	require.NoError(t, err)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(schemaSrc, e.Name()))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(schemaDst, e.Name()), data, 0o644))
	}
}

func TestPromptTags_DefaultBaselineSelectsUntagged(t *testing.T) {
	tmp := t.TempDir()
	copyCompilerValidFixture(t, tmp)
	stdout, stderr, code := runTTInDir(t, tmp, "prompt", "compile", "--dry-run", "--target", "cursor")
	require.Equal(t, 0, code, "stderr: %s", stderr)
	assert.Contains(t, stdout, "selected: true")
}

func TestPromptTags_TestOnlyExcludesBaseline(t *testing.T) {
	tmp := t.TempDir()
	copyCompilerValidFixture(t, tmp)
	writeTaggedEntities(t, tmp)

	stdout, stderr, code := runTTInDir(t, tmp, "prompt", "compile", "--dry-run", "--target", "cursor", "--tags", "test")
	require.Equal(t, 0, code, "stderr: %s\nstdout: %s", stderr, stdout)
	assert.Contains(t, stdout, "id: test-cap")
	// baseline-cap should appear with selected: false (still listed in resolved)
	require.Contains(t, stdout, "id: baseline-cap")
	idx := strings.Index(stdout, "id: baseline-cap")
	snippet := stdout[idx:]
	if i := strings.Index(snippet, "\n- apiVersion:"); i > 0 {
		snippet = snippet[:i]
	}
	assert.Contains(t, snippet, "selected: false")
}

func TestPromptTags_IncludePullsReferencedCapability(t *testing.T) {
	tmp := t.TempDir()
	copyCompilerValidFixture(t, tmp)
	writeTaggedEntities(t, tmp)

	stdout, stderr, code := runTTInDir(t, tmp, "prompt", "compile", "--dry-run", "--target", "cursor", "--tags", "baseline")
	require.Equal(t, 0, code, "stderr: %s", stderr)
	assert.Contains(t, stdout, "id: baseline-proc")
	assert.Contains(t, stdout, "id: test-cap")
	// test-cap should be selected via include closure
	idx := strings.Index(stdout, "id: test-cap")
	require.GreaterOrEqual(t, idx, 0)
	snippet := stdout[idx:]
	assert.Contains(t, snippet, "selected: true")
}

func TestPromptTags_StrictFailsOnCrossTagRef(t *testing.T) {
	tmp := t.TempDir()
	copyCompilerValidFixture(t, tmp)
	writeTaggedEntities(t, tmp)

	_, stderr, code := runTTInDir(t, tmp, "prompt", "compile", "--dry-run", "--target", "cursor",
		"--tags", "baseline", "--tag-refs", "strict")
	require.NotEqual(t, 0, code, "expected failure, stderr: %s", stderr)
	assert.Contains(t, stderr, "strict")
}

func TestPromptTags_TT_TAGSEnv(t *testing.T) {
	tmp := t.TempDir()
	copyCompilerValidFixture(t, tmp)
	writeTaggedEntities(t, tmp)

	t.Setenv("TT_TAGS", "test")
	stdout, stderr, code := runTTInDir(t, tmp, "prompt", "compile", "--dry-run", "--target", "cursor")
	require.Equal(t, 0, code, "stderr: %s", stderr)
	idx := strings.Index(stdout, "id: test-cap")
	require.GreaterOrEqual(t, idx, 0)
	assert.Contains(t, stdout[idx:], "selected: true")
}

func TestPromptDeploy_DigestDiffersByTags(t *testing.T) {
	tmp := t.TempDir()
	copyCompilerValidFixture(t, tmp)
	writeTaggedEntities(t, tmp)

	_, stderr, code := runTTInDir(t, tmp, "prompt", "deploy", "--target", "cursor", "--tags", "baseline", "--force")
	require.Equal(t, 0, code, "stderr: %s", stderr)

	stdout, stderr, code := runTTInDir(t, tmp, "prompt", "deploy", "--target", "cursor", "--tags", "test")
	require.Equal(t, 0, code, "stderr: %s", stderr)
	assert.NotContains(t, stdout, "Skipping deploy")
	assert.Contains(t, stdout, "Deploy succeeded")
}
