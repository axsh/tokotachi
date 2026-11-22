package scaffold

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareUpdateWorkspace_DirtyWritesInstruction(t *testing.T) {
	repo := t.TempDir()
	local := filepath.Join(repo, "prompts", "manifest", "code_content", "capabilities")
	require.NoError(t, os.MkdirAll(local, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(local, "a.md"), []byte("---\nid: a\n---\n\n# local edit\n"), 0o644))

	workdir := filepath.Join(repo, "tmp", "update")
	var buf bytes.Buffer
	res, err := PrepareUpdateWorkspace(UpdateWorkspaceOptions{
		RepoRoot:      repo,
		WorkDir:       workdir,
		LocalManifest: filepath.Join(repo, "prompts", "manifest"),
		LatestFiles: map[string][]byte{
			"prompts/manifest/code_content/capabilities/a.md": []byte("---\nid: a\n---\n\n# upstream\n"),
		},
		Stdout: &buf,
	})
	require.NoError(t, err)
	require.False(t, res.CleanApplied)
	assert.FileExists(t, filepath.Join(workdir, "instruction.md"))
	assert.FileExists(t, filepath.Join(workdir, "doc-relations-001.md"))
	assert.Contains(t, buf.String(), "Do not discard or rewrite project-specific information")
	assert.Contains(t, buf.String(), "Instruction file saved at:")
	// local not overwritten
	data, err := os.ReadFile(filepath.Join(local, "a.md"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "local edit")
}

func TestPrepareUpdateWorkspace_CleanApplies(t *testing.T) {
	repo := t.TempDir()
	content := []byte("---\nid: a\n---\n\n# same\n")
	local := filepath.Join(repo, "prompts", "manifest", "code_content", "capabilities")
	require.NoError(t, os.MkdirAll(local, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(local, "a.md"), content, 0o644))

	workdir := filepath.Join(repo, "tmp", "update")
	var buf bytes.Buffer
	res, err := PrepareUpdateWorkspace(UpdateWorkspaceOptions{
		RepoRoot:      repo,
		WorkDir:       workdir,
		LocalManifest: filepath.Join(repo, "prompts", "manifest"),
		LatestFiles: map[string][]byte{
			"prompts/manifest/code_content/capabilities/a.md": content,
		},
		Stdout: &buf,
	})
	require.NoError(t, err)
	assert.True(t, res.CleanApplied)
	assert.Contains(t, buf.String(), "Clean update applied")
}

func TestDefaultMergeInstruction_Policies(t *testing.T) {
	s := DefaultMergeInstruction("/tmp/u", []string{"doc-relations-001.md"})
	for _, phrase := range []string{
		"Do not discard or rewrite project-specific information without cause.",
		"Prefer incorporating improvements and convenience from `latest/`.",
		"Do not break existing references",
		"Keep current behavior working while absorbing upstream improvements.",
		"doc-relations-NNN.md",
		"leave the workdir intact",
	} {
		assert.True(t, strings.Contains(s, phrase), "missing %q", phrase)
	}
}
