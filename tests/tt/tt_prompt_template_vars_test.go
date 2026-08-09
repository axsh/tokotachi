package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func copyCatalogTemplateFixture(t *testing.T, dst string) {
	t.Helper()
	src := filepath.Join(projectRoot(), "features", "tt", "internal", "prompt", "compiler", "testdata", "catalog_template")
	require.NoError(t, os.MkdirAll(dst, 0o755))
	copyDirContents(t, src, dst)
}

func runPromptUpdateTarget(t *testing.T, workspace, target string) {
	t.Helper()
	_, stderr, code := runTTInDir(t, workspace, "prompt", "update", "--force", "--target", target)
	require.Equal(t, 0, code, "stderr: %s", stderr)
}

func TestPromptTemplateVars_ClaudeResolvesPolicyAndProcedure(t *testing.T) {
	tmp := t.TempDir()
	copyCatalogTemplateFixture(t, tmp)
	runPromptUpdateTarget(t, tmp, "claude-code")

	skill := filepath.Join(tmp, ".claude", "skills", "build-pipeline", "SKILL.md")
	data, err := os.ReadFile(skill)
	require.NoError(t, err)
	content := string(data)
	assert.NotContains(t, content, "{{policy:")
	assert.Contains(t, content, ".claude/rules/testing-rules.md")

	instr := filepath.Join(tmp, ".claude", "rules", "project-instructions.md")
	idata, err := os.ReadFile(instr)
	require.NoError(t, err)
	icontent := string(idata)
	assert.NotContains(t, icontent, ".agent/workflows")
	assert.Contains(t, icontent, ".claude/skills/")
	assert.Contains(t, icontent, ".claude/skills/build-pipeline/SKILL.md")
	assert.NotContains(t, icontent, "{{procedure:")
	assert.NotContains(t, icontent, "{{target:workflows}}")
}

func TestPromptTemplateVars_CodexResolves(t *testing.T) {
	tmp := t.TempDir()
	copyCatalogTemplateFixture(t, tmp)
	runPromptUpdateTarget(t, tmp, "codex")

	skill := filepath.Join(tmp, ".codex", "skills", "build-pipeline", "SKILL.md")
	data, err := os.ReadFile(skill)
	require.NoError(t, err)
	content := string(data)
	assert.NotContains(t, content, "{{policy:")
	assert.Contains(t, content, ".codex/rules/testing-rules.md")

	instr := filepath.Join(tmp, ".codex", "rules", "project-instructions.md")
	idata, err := os.ReadFile(instr)
	require.NoError(t, err)
	icontent := string(idata)
	assert.NotContains(t, icontent, ".agent/workflows")
	assert.Contains(t, icontent, ".codex/skills/build-pipeline/SKILL.md")
}

func TestPromptTemplateVars_CursorResolvesMdc(t *testing.T) {
	tmp := t.TempDir()
	copyCatalogTemplateFixture(t, tmp)
	runPromptUpdateTarget(t, tmp, "cursor")

	skill := filepath.Join(tmp, ".cursor", "skills", "build-pipeline", "SKILL.md")
	data, err := os.ReadFile(skill)
	require.NoError(t, err)
	content := string(data)
	assert.NotContains(t, content, "{{policy:")
	assert.Contains(t, content, ".cursor/rules/testing-rules.mdc")

	instr := filepath.Join(tmp, ".cursor", "rules", "project-instructions.mdc")
	idata, err := os.ReadFile(instr)
	require.NoError(t, err)
	icontent := string(idata)
	assert.NotContains(t, icontent, ".agent/workflows")
	assert.Contains(t, icontent, ".cursor/skills/")
	assert.Contains(t, icontent, ".cursor/skills/build-pipeline/SKILL.md")
}

func TestPromptTemplateVars_AntigravityRegression(t *testing.T) {
	tmp := t.TempDir()
	copyCatalogTemplateFixture(t, tmp)
	runPromptUpdateTarget(t, tmp, "antigravity")

	wf := filepath.Join(tmp, ".agent", "workflows", "build-pipeline.md")
	data, err := os.ReadFile(wf)
	require.NoError(t, err)
	content := string(data)
	assert.NotContains(t, content, "{{policy:")
	assert.Contains(t, content, ".agent/rules/testing-rules.md")

	instr := filepath.Join(tmp, ".agent", "rules", "instructions.md")
	idata, err := os.ReadFile(instr)
	require.NoError(t, err)
	icontent := string(idata)
	assert.Contains(t, icontent, ".agent/workflows/")
	assert.Contains(t, icontent, ".agent/workflows/build-pipeline.md")
	assert.True(t, strings.Contains(icontent, ".agent/workflows/") || strings.Contains(icontent, "build-pipeline"))
}
