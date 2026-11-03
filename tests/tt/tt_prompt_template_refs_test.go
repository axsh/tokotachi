package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTemplateRefFixture(t *testing.T, root string) {
	t.Helper()

	polDir := filepath.Join(root, "prompts", "manifest", "code_content", "policies")
	capDir := filepath.Join(root, "prompts", "manifest", "code_content", "capabilities")
	procDir := filepath.Join(root, "prompts", "manifest", "code_content", "procedures")
	tgtDir := filepath.Join(root, "prompts", "manifest", "targets")
	require.NoError(t, os.MkdirAll(polDir, 0o755))
	require.NoError(t, os.MkdirAll(capDir, 0o755))
	require.NoError(t, os.MkdirAll(procDir, 0o755))
	require.NoError(t, os.MkdirAll(tgtDir, 0o755))

	coding := `---
apiVersion: agent.meta/v1
kind: policy
id: coding-rules
title: Coding Rules
scope: project
activation:
  mode: manual
body: inline
tags:
  - baseline
---

# Coding Rules
`
	require.NoError(t, os.WriteFile(filepath.Join(polDir, "coding-rules.md"), []byte(coding), 0o644))

	cap := `---
apiVersion: agent.meta/v1
kind: capability
id: demo-cap
title: Demo Cap
description: demo
body: inline
tags:
  - baseline
---

# Demo Cap
`
	require.NoError(t, os.WriteFile(filepath.Join(capDir, "demo-cap.md"), []byte(cap), 0o644))

	proc := `---
apiVersion: agent.meta/v1
kind: procedure
id: demo-proc
title: Demo Proc
trigger:
  command: demo-proc
description: demo procedure
tags:
  - baseline
---

# Demo Proc

Read {{policy:coding-rules}} then {{capability:demo-cap}} and {{procedure:demo-proc}}.
`
	require.NoError(t, os.WriteFile(filepath.Join(procDir, "demo-proc.md"), []byte(proc), 0o644))

	cursorYAML := `apiVersion: agent.meta/v1
kind: target
id: cursor
includes:
  policy: true
  capability: true
  procedure: true
  subagent: false
paths:
  rules: .cursor/rules/
  skills: .cursor/skills/
`
	agYAML := `apiVersion: agent.meta/v1
kind: target
id: antigravity
includes:
  policy: true
  capability: true
  procedure: true
  subagent: false
paths:
  rules: .agent/rules/
  skills: .agent/skills/
  workflows: .agent/workflows/
`
	require.NoError(t, os.WriteFile(filepath.Join(tgtDir, "cursor.yaml"), []byte(cursorYAML), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tgtDir, "antigravity.yaml"), []byte(agYAML), 0o644))

	projectPath := filepath.Join(root, "prompts", "manifest", "project.yaml")
	proj := `version: 1
project_id: test-project
sources:
  policies: "prompts/manifest/code_content/policies/**/*.md"
  capabilities: "prompts/manifest/code_content/capabilities/**/*.md"
  procedures: "prompts/manifest/code_content/procedures/**/*.md"
  targets: "prompts/manifest/targets/**/*.yaml"
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

func TestPromptTemplateRefs_CursorDeployResolves(t *testing.T) {
	tmp := t.TempDir()
	writeTemplateRefFixture(t, tmp)

	_, stderr, code := runTTInDir(t, tmp, "prompt", "deploy", "--target", "cursor", "--force")
	require.Equal(t, 0, code, "stderr: %s", stderr)

	skillMD := filepath.Join(tmp, ".cursor", "skills", "demo-proc", "SKILL.md")
	data, err := os.ReadFile(skillMD)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, ".cursor/rules/coding-rules.mdc")
	assert.Contains(t, content, ".cursor/skills/demo-cap/SKILL.md")
	assert.Contains(t, content, ".cursor/skills/demo-proc/SKILL.md")
	assert.NotContains(t, content, "{{policy:")
	assert.NotContains(t, content, "{{capability:")
	assert.NotContains(t, content, "{{procedure:")
}

func TestPromptTemplateRefs_AntigravityDeployResolves(t *testing.T) {
	tmp := t.TempDir()
	writeTemplateRefFixture(t, tmp)

	_, stderr, code := runTTInDir(t, tmp, "prompt", "deploy", "--target", "ag", "--force")
	require.Equal(t, 0, code, "stderr: %s", stderr)

	wf := filepath.Join(tmp, ".agent", "workflows", "demo-proc.md")
	data, err := os.ReadFile(wf)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, ".agent/rules/coding-rules.md")
	assert.Contains(t, content, ".agent/skills/demo-cap/SKILL.md")
	assert.Contains(t, content, ".agent/workflows/demo-proc.md")
	assert.NotContains(t, content, "{{policy:")
}

func TestPromptInstruction_PrintsManual(t *testing.T) {
	stdout, stderr, code := runTT(t, "prompt", "instruction")
	require.Equal(t, 0, code, "stderr: %s", stderr)
	assert.Contains(t, stdout, "{{kind:id}}")
	assert.Contains(t, stdout, "policy")
	assert.Contains(t, stdout, "procedure")
	assert.Contains(t, stdout, "capability")
	assert.Contains(t, stdout, ".mdc")
}

func TestPromptHelp_MentionsInstruction(t *testing.T) {
	stdout, stderr, code := runTT(t, "--help")
	require.Equal(t, 0, code, "stderr: %s", stderr)
	assert.True(t, strings.Contains(stdout, "instruction") || strings.Contains(stdout, "{{kind:id}}"),
		"root help should mention instruction or template refs:\n%s", stdout)

	stdout, stderr, code = runTT(t, "prompt", "--help")
	require.Equal(t, 0, code, "stderr: %s", stderr)
	assert.Contains(t, stdout, "instruction")
	assert.Contains(t, stdout, "{{kind:id}}")
}
