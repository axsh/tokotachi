package backlog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const legacyMD = `# 012 RLS の扱い

- 状態: 確認中（2026-05-02 更新）
- 種類: 確認事項
- 確かめる相手: 依頼主
- いつまでに: 次回
- 出どころ: 仕様書
  - 第3章
- 謎のキー: ignored

## 何が問題か

RLS を付けるか決まっていない。

` + "```" + `
## not a section
` + "```" + `

## 今の扱い（暫定）

付けない。

## 経緯

- 2026-05-01 起票
- 2026-05-02 確認開始

## 追加メモ

extra
`

func TestParseMarkdownItem_Fields(t *testing.T) {
	var warnings []string
	rel := "prompts/phases/000-foundation/branches/main/backlogs/12-Rls-Policy.md"
	it, ok := ParseMarkdownItem(rel, legacyMD, "2026-11-18", &warnings)
	require.True(t, ok)
	assert.Equal(t, "000-foundation", it.Phase)
	assert.Equal(t, "main", it.Branch)
	assert.Equal(t, "012", it.LocalID, "local id is zero-padded to 3 digits")
	assert.Equal(t, "Rls-Policy", it.Slug)
	assert.Equal(t, "RLS の扱い", it.Title, "numeric prefix removed")
	assert.Equal(t, "確認中", it.State)
	require.NotNil(t, it.StateDate)
	assert.Equal(t, "2026-05-02", *it.StateDate)
	assert.Equal(t, "確認事項", it.Kind)
	assert.Equal(t, "依頼主", it.AskTo)
	assert.Equal(t, "次回", it.Due)
	assert.Equal(t, "仕様書\n- 第3章", it.Origin, "nested list lines are kept and dedented")
	assert.Contains(t, it.Problem, "RLS を付けるか決まっていない。")
	assert.Contains(t, it.Problem, "## not a section", "fenced headings stay in the body")
	assert.Contains(t, it.Problem, "## 追加メモ\n\nextra", "unknown sections are appended to problem")
	assert.Equal(t, "付けない。", it.CurrentHandling)
	assert.Equal(t, "- 2026-05-01 起票\n- 2026-05-02 確認開始", it.History)
	assert.Equal(t, "2026-05-01", it.CreatedAt)
	assert.Equal(t, "2026-05-02", it.UpdatedAt)
	assert.True(t, ValidUUID(it.UUID))
	assert.Contains(t, warnings, "000-foundation/main/012: unknown header key ignored: 謎のキー")
	assert.Contains(t, warnings, "000-foundation/main/012: unknown section '追加メモ' appended to problem")
}

func TestParseMarkdownItem_UnmappedStateAndKind(t *testing.T) {
	md := "# 1 T\n\n- 状態: 仮の決定\n- 種類: これは先送り案件\n\n## 経緯\n\nnone\n"
	var warnings []string
	it, ok := ParseMarkdownItem("prompts/phases/p/branches/b/backlogs/001-T.md", md, "2026-11-18", &warnings)
	require.True(t, ok)
	assert.Equal(t, "未確認", it.State)
	assert.Equal(t, "先送り", it.Kind, "kind is normalized to the contained allowed kind")
	assert.Contains(t, it.History, "- (import) original state line: 仮の決定")
	assert.Equal(t, "2026-11-18", it.CreatedAt, "today when no dates are present")
	assert.Nil(t, it.StateDate)
	assert.Contains(t, warnings, "p/b/001: unmapped state line '仮の決定' -> 未確認 (original kept in history)")
}

func TestParseMarkdownItem_RejectsOtherLayout(t *testing.T) {
	var w []string
	_, ok := ParseMarkdownItem("docs/readme.md", "# x", "2026-11-18", &w)
	assert.False(t, ok)
}

func TestOps_ImportMD(t *testing.T) {
	o := newOps(t)
	root := t.TempDir()
	dir := filepath.Join(root, "prompts", "phases", "000-foundation", "branches", "main", "backlogs")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "012-Rls.md"), []byte(legacyMD), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# not an item"), 0o644))

	res, err := o.ImportMD([]string{root})
	require.NoError(t, err)
	assert.Equal(t, 1, res["imported"])
	assert.Equal(t, []string{"000-foundation/main/012"}, res["keys"])
	warnings := res["warnings"].([]string)
	assert.Contains(t, warnings[0], "skipped 1 markdown file(s)")

	got, err := o.Get(flagsOf(t, GetFlags, "--phase", "000-foundation", "--branch", "main", "--id", "012"))
	require.NoError(t, err)
	assert.Equal(t, "RLS の扱い", got["item"].(Item).Title)

	// Re-import replaces the row (INSERT OR REPLACE).
	_, err = o.ImportMD([]string{filepath.Join(dir, "012-Rls.md")})
	require.NoError(t, err)
	list, err := o.List(flagsOf(t, ListFlags, "--include-closed", "1"))
	require.NoError(t, err)
	assert.Equal(t, 1, list["total"])

	_, err = o.ImportMD([]string{filepath.Join(root, "nope")})
	assert.Equal(t, 2, CodeOf(err))
	other := filepath.Join(root, "x.md")
	require.NoError(t, os.WriteFile(other, []byte("# x"), 0o644))
	_, err = o.ImportMD([]string{other})
	assert.Equal(t, 2, CodeOf(err))
	_, err = o.ImportMD(nil)
	assert.Equal(t, 2, CodeOf(err))
	empty := t.TempDir()
	_, err = o.ImportMD([]string{empty})
	assert.Equal(t, 1, CodeOf(err))
}
