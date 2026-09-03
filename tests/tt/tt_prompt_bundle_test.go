//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeBundleCapabilityFixture(t *testing.T, root string) {
	t.Helper()

	capDir := filepath.Join(root, "prompts", "manifest", "code_content", "capabilities")
	require.NoError(t, os.MkdirAll(capDir, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "fixtures"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "fixtures", "payload.json"), []byte(`{"v":1}`), 0o644))

	bundled := `---
apiVersion: agent.meta/v1
kind: capability
id: bundled-cap
title: Bundled Cap
description: has bundle
bundle:
  - src: fixtures/payload.json
    dest: references/payload.json
body: inline
tags:
  - baseline
---

# Bundled Cap

See ` + "`fixtures/payload.json`" + ` for details.
`
	refsOnly := `---
apiVersion: agent.meta/v1
kind: capability
id: refs-only-cap
title: Refs Only Cap
description: references only
references:
  - fixtures/payload.json
scripts:
  - scripts/code/agent/record.sh
body: inline
tags:
  - baseline
---

# Refs Only

Run ` + "`./scripts/code/agent/record.sh`" + ` and maybe ` + "`fixtures/payload.json`" + `.
`
	require.NoError(t, os.WriteFile(filepath.Join(capDir, "bundled-cap.md"), []byte(bundled), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(capDir, "refs-only-cap.md"), []byte(refsOnly), 0o644))

	projectPath := filepath.Join(root, "prompts", "manifest", "project.yaml")
	proj := `version: 1
project_id: test-project
sources:
  policies: "prompts/manifest/code_content/policies/**/*.yaml"
  capabilities: "prompts/manifest/code_content/capabilities/**/*.md"
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

func TestPromptDeploy_CapabilityBundle(t *testing.T) {
	tmp := t.TempDir()
	copyCompilerValidFixture(t, tmp)
	writeBundleCapabilityFixture(t, tmp)

	_, stderr, code := runTTInDir(t, tmp, "prompt", "deploy", "--target", "cursor", "--force")
	require.Equal(t, 0, code, "stderr: %s", stderr)

	skillMD := filepath.Join(tmp, ".cursor", "skills", "bundled-cap", "SKILL.md")
	data, err := os.ReadFile(skillMD)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "`references/payload.json`")
	assert.NotContains(t, content, "fixtures/payload.json")

	companion := filepath.Join(tmp, ".cursor", "skills", "bundled-cap", "references", "payload.json")
	got, err := os.ReadFile(companion)
	require.NoError(t, err)
	assert.Equal(t, `{"v":1}`, string(got))
}

func TestPromptDeploy_ReferencesOnlyNoBundle(t *testing.T) {
	tmp := t.TempDir()
	copyCompilerValidFixture(t, tmp)
	writeBundleCapabilityFixture(t, tmp)

	_, stderr, code := runTTInDir(t, tmp, "prompt", "deploy", "--target", "cursor", "--force")
	require.Equal(t, 0, code, "stderr: %s", stderr)

	refsSkill := filepath.Join(tmp, ".cursor", "skills", "refs-only-cap")
	entries, err := os.ReadDir(refsSkill)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "SKILL.md", entries[0].Name())

	data, err := os.ReadFile(filepath.Join(refsSkill, "SKILL.md"))
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "`./scripts/code/agent/record.sh`")
	assert.Contains(t, content, "`fixtures/payload.json`")
}
