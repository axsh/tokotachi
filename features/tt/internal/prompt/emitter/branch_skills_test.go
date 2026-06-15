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
	skills, err := ScanBranchSkills(root)
	require.NoError(t, err)
	assert.Empty(t, skills)
}

func TestScanBranchSkills_EmptyBranches(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "prompts", "memory", "branches"), 0o755))

	skills, err := ScanBranchSkills(root)
	require.NoError(t, err)
	assert.Empty(t, skills)
}

func TestScanBranchSkills_BranchWithoutSkills(t *testing.T) {
	root := t.TempDir()
	branchDir := filepath.Join(root, "prompts", "memory", "branches", "BR-test-123")
	require.NoError(t, os.MkdirAll(branchDir, 0o755))

	skills, err := ScanBranchSkills(root)
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

	skills, err := ScanBranchSkills(root)
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

	skills, err := ScanBranchSkills(root)
	require.NoError(t, err)
	assert.Len(t, skills, 3)
}

func TestScanBranchSkills_SkipsDirWithoutSKILLMD(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "prompts", "memory", "branches", "BR-test", "skills", "__far-knowledge-broken")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	// No SKILL.md file

	skills, err := ScanBranchSkills(root)
	require.NoError(t, err)
	assert.Empty(t, skills)
}

func TestEmitBranchSkills(t *testing.T) {
	targetDir := t.TempDir()
	skillsDir := filepath.Join(targetDir, "skills")

	branchSkills := []BranchSkill{
		{
			ID:       "__far-knowledge-errors",
			Content:  "---\nname: errors\n---\n\n# Error Patterns\n",
			BranchID: "BR-test",
		},
		{
			ID:       "__far-knowledge-logging",
			Content:  "---\nname: logging\n---\n\n# Logging Patterns\n",
			BranchID: "BR-test",
		},
	}

	emitted, err := EmitBranchSkills(branchSkills, skillsDir, EmitOptions{Mode: EmitModeOverwrite})
	require.NoError(t, err)
	assert.Len(t, emitted, 2)

	// Verify files exist
	for _, skill := range branchSkills {
		path := filepath.Join(skillsDir, skill.ID, "SKILL.md")
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(data), "Patterns")
	}
}
