package emitter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBundleEntries_Nil(t *testing.T) {
	entries, err := ParseBundleEntries(nil)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestParseBundleEntries_OK(t *testing.T) {
	raw := []any{
		map[string]any{
			"src":  "prompts/memory/schemas/foo.json",
			"dest": "references/foo.json",
		},
	}
	entries, err := ParseBundleEntries(raw)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "prompts/memory/schemas/foo.json", entries[0].Src)
	assert.Equal(t, "references/foo.json", entries[0].Dest)
}

func TestParseBundleEntries_MissingFields(t *testing.T) {
	_, err := ParseBundleEntries([]any{map[string]any{"src": "a.json"}})
	require.Error(t, err)
	_, err = ParseBundleEntries([]any{map[string]any{"dest": "references/a.json"}})
	require.Error(t, err)
}

func TestValidateSkillDest_RejectsTraversal(t *testing.T) {
	require.Error(t, ValidateSkillDest("../other/evil.json"))
	require.Error(t, ValidateSkillDest("foo/../../evil.json"))
}

func TestValidateSkillDest_RejectsAbsolute(t *testing.T) {
	require.Error(t, ValidateSkillDest("/tmp/evil.json"))
}

func TestValidateSkillDest_OK(t *testing.T) {
	require.NoError(t, ValidateSkillDest("references/foo.json"))
	require.NoError(t, ValidateSkillDest("./references/foo.json"))
}

func TestRewriteBundlePaths_BacktickAndMarkdownLink(t *testing.T) {
	entries := []BundleEntry{
		{Src: "testdata/foo.schema.json", Dest: "references/foo.schema.json"},
	}
	body := "See `testdata/foo.schema.json` and [doc](testdata/foo.schema.json) plus [rel](./testdata/foo.schema.json)."
	got := RewriteBundlePaths(body, entries)
	assert.Contains(t, got, "`references/foo.schema.json`")
	assert.Contains(t, got, "](references/foo.schema.json)")
	assert.Contains(t, got, "](./references/foo.schema.json)")
	assert.NotContains(t, got, "testdata/foo.schema.json")
}

func TestRewriteBundlePaths_LongestSrcFirst(t *testing.T) {
	entries := []BundleEntry{
		{Src: "a/b", Dest: "short"},
		{Src: "a/b/c.json", Dest: "references/c.json"},
	}
	body := "x `a/b/c.json` y"
	got := RewriteBundlePaths(body, entries)
	assert.Equal(t, "x `references/c.json` y", got)
}

func TestRewriteBundlePaths_LeavesUnbundledPaths(t *testing.T) {
	entries := []BundleEntry{
		{Src: "docs/a.json", Dest: "references/a.json"},
	}
	body := "run `./scripts/code/agent/record.sh` and see `docs/a.json`"
	got := RewriteBundlePaths(body, entries)
	assert.Contains(t, got, "`./scripts/code/agent/record.sh`")
	assert.Contains(t, got, "`references/a.json`")
}

func TestEmitBundledFiles_CopiesAndRegisters(t *testing.T) {
	root := t.TempDir()
	srcRel := "fixtures/payload.json"
	require.NoError(t, os.MkdirAll(filepath.Join(root, "fixtures"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "fixtures", "payload.json"), []byte(`{"ok":true}`), 0o644))

	skillDir := filepath.Join(root, ".cursor", "skills", "demo")
	entries := []BundleEntry{{Src: srcRel, Dest: "references/payload.json"}}
	emitted, err := EmitBundledFiles(skillDir, root, entries, EmitOptions{Mode: EmitModeOverwrite})
	require.NoError(t, err)

	out := filepath.Join(skillDir, "references", "payload.json")
	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, `{"ok":true}`, string(data))
	assert.True(t, emitted[filepath.Clean(out)])
}

func TestEmitBundledFiles_MissingSrcErrors(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "skills", "demo")
	_, err := EmitBundledFiles(skillDir, root, []BundleEntry{
		{Src: "missing.json", Dest: "references/missing.json"},
	}, EmitOptions{Mode: EmitModeOverwrite})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bundle src not found")
}

func TestEmitBundledFiles_DestEscapeErrors(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "x.json"), []byte("x"), 0o644))
	skillDir := filepath.Join(root, "skills", "demo")
	_, err := EmitBundledFiles(skillDir, root, []BundleEntry{
		{Src: "x.json", Dest: "../other/evil.json"},
	}, EmitOptions{Mode: EmitModeOverwrite})
	require.Error(t, err)
}

func TestRewriteBundlePaths_Empty(t *testing.T) {
	assert.Equal(t, "body", RewriteBundlePaths("body", nil))
	assert.Equal(t, "", RewriteBundlePaths("", []BundleEntry{{Src: "a", Dest: "b"}}))
}

func TestParseBundleEntries_WrongType(t *testing.T) {
	_, err := ParseBundleEntries("not-array")
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "expected array"))
}
