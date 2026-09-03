package emitter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanBranchSkills_NoBranchesDir(t *testing.T) {
	root := t.TempDir()
	skills, err := ScanBranchSkills(root, "prompts")
	require.NoError(t, err)
	assert.Empty(t, skills)
}

func TestScanBranchSkills_EmptyBranches(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "prompts", "memory", "branches"), 0o755))

	skills, err := ScanBranchSkills(root, "prompts")
	require.NoError(t, err)
	assert.Empty(t, skills)
}

func TestScanBranchSkills_BranchWithoutSkills(t *testing.T) {
	root := t.TempDir()
	branchDir := filepath.Join(root, "prompts", "memory", "branches", "BR-test-123")
	require.NoError(t, os.MkdirAll(branchDir, 0o755))

	skills, err := ScanBranchSkills(root, "prompts")
	require.NoError(t, err)
	assert.Empty(t, skills)
}

func TestScanBranchSkills_SingleSkill(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "prompts", "memory", "branches", "BR-test-123", "skills", "__far-knowledge-error-handling")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("---\nname: error-handling\n---\n\n# Error Handling Patterns\n"),
		0o644,
	))

	skills, err := ScanBranchSkills(root, "prompts")
	require.NoError(t, err)
	require.Len(t, skills, 1)
	assert.Equal(t, "__far-knowledge-error-handling", skills[0].ID)
	assert.Equal(t, "BR-test-123", skills[0].BranchID)
	assert.Contains(t, skills[0].Content, "Error Handling Patterns")
}

func TestScanBranchSkills_MultipleSkillsMultipleBranches(t *testing.T) {
	root := t.TempDir()

	// Branch 1 with 2 skills
	for _, skillName := range []string{"__far-knowledge-errors", "__far-knowledge-logging"} {
		skillDir := filepath.Join(root, "prompts", "memory", "branches", "BR-branch-1", "skills", skillName)
		require.NoError(t, os.MkdirAll(skillDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(skillDir, "SKILL.md"),
			[]byte("---\nname: "+skillName+"\n---\n\n# "+skillName+"\n"),
			0o644,
		))
	}

	// Branch 2 with 1 skill
	skillDir := filepath.Join(root, "prompts", "memory", "branches", "BR-branch-2", "skills", "__far-knowledge-testing")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("---\nname: testing\n---\n\n# Testing\n"),
		0o644,
	))

	skills, err := ScanBranchSkills(root, "prompts")
	require.NoError(t, err)
	assert.Len(t, skills, 3)
}

func TestScanBranchSkills_SkipsDirWithoutSKILLMD(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "prompts", "memory", "branches", "BR-test", "skills", "__far-knowledge-broken")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	// No SKILL.md file

	skills, err := ScanBranchSkills(root, "prompts")
	require.NoError(t, err)
	assert.Empty(t, skills)
}

func TestEmitBranchSkills(t *testing.T) {
	srcRoot := t.TempDir()
	targetDir := t.TempDir()
	skillsDir := filepath.Join(targetDir, "skills")

	makeSkill := func(id, body string, companions map[string]string) BranchSkill {
		t.Helper()
		dir := filepath.Join(srcRoot, id)
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "references"), 0o755))
		skillPath := filepath.Join(dir, "SKILL.md")
		require.NoError(t, os.WriteFile(skillPath, []byte(body), 0o644))
		for rel, content := range companions {
			p := filepath.Join(dir, filepath.FromSlash(rel))
			require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
			require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
		}
		return BranchSkill{ID: id, Path: skillPath, Content: body, BranchID: "BR-test"}
	}

	branchSkills := []BranchSkill{
		makeSkill("__far-knowledge-errors", "---\nname: errors\n---\n\n# Error Patterns\n", map[string]string{
			"references/note.md": "# note\n",
		}),
		makeSkill("__far-knowledge-logging", "---\nname: logging\n---\n\n# Logging Patterns\n", nil),
	}

	emitted, err := EmitBranchSkills(branchSkills, skillsDir, EmitOptions{Mode: EmitModeOverwrite})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(emitted), 3)

	for _, skill := range branchSkills {
		path := filepath.Join(skillsDir, skill.ID, "SKILL.md")
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(data), "Patterns")
	}
	notePath := filepath.Join(skillsDir, "__far-knowledge-errors", "references", "note.md")
	data, err := os.ReadFile(notePath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "note")
}
