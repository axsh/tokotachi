package backlog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mkItem(phase, branch, id, title string) Item {
	return Item{
		Phase: phase, Branch: branch, LocalID: id, Title: title,
		State: StateUnconfirmed, StateDate: StrPtr("2026-11-18"), Kind: "確認事項",
		Problem: "p <tag> & \"quote\"\nline2\ttab 日本語", CreatedAt: "2026-11-18", UpdatedAt: "2026-11-18",
		UUID: NewUUID(),
	}
}

func TestCollapse_IdenticalAndConflict(t *testing.T) {
	a := mkItem("p", "b", "001", "A")
	b := mkItem("p", "b", "002", "B")
	b2 := b
	b2.Title = "B changed"

	out, collapsed, conflicts := Collapse([]Item{a, b, a, b2})
	assert.Equal(t, 1, collapsed, "the identical copy of a is dropped")
	assert.Equal(t, []Key{{"p", "b", "002"}}, conflicts)
	require.Len(t, out, 3, "both conflicting rows are kept")
	assert.Equal(t, "A", out[0].Title)
	assert.Equal(t, "B", out[1].Title)
	assert.Equal(t, "B changed", out[2].Title)
}

func TestFindProblems(t *testing.T) {
	ok := mkItem("p", "b", "001", "ok")
	badState := mkItem("p", "b", "002", "x")
	badState.State = "weird"
	emptyTitle := mkItem("p", "b", "003", "  ")
	badKind := mkItem("p", "b", "004", "k")
	badKind.Kind = "zzz"
	noUUID := mkItem("p", "b", "005", "n")
	noUUID.UUID = ""
	badUUID := mkItem("p", "b", "006", "u")
	badUUID.UUID = "abc"
	dup1 := mkItem("p", "b", "007", "d1")
	dup2 := mkItem("p", "b", "008", "d2")
	dup2.UUID = strings.ToUpper(dup1.UUID)

	probs := FindProblems([]Item{ok, badState, emptyTitle, badKind, noUUID, badUUID, dup1, dup2})
	var msgs []string
	for _, p := range probs {
		msgs = append(msgs, p.Key+" "+p.Problem)
	}
	joined := strings.Join(msgs, "\n")
	for _, want := range []string{
		"p/b/002 invalid state: weird", "p/b/003 empty title", "p/b/004 invalid kind: zzz",
		"p/b/005 empty uuid", "p/b/006 invalid uuid format: abc", "p/b/007, p/b/008 duplicate uuid: ",
	} {
		assert.Contains(t, joined, want)
	}
	assert.NotContains(t, joined, "p/b/001")
}

// Golden: the canonical bytes must match Python json.dumps(indent=2, ensure_ascii=False).
func TestCanonicalArrayBytes_Golden(t *testing.T) {
	it := Item{Phase: "p", Branch: "b", LocalID: "001", Title: "日本語 \"q\" <&>", State: "未確認", Problem: "a\nb\x01"}
	got := string(CanonicalArrayBytes([]Item{it}))
	want := "[\n  {\n" +
		"    \"phase\": \"p\",\n    \"branch\": \"b\",\n    \"local_id\": \"001\",\n    \"slug\": \"\",\n" +
		"    \"title\": \"日本語 \\\"q\\\" <&>\",\n    \"state\": \"未確認\",\n    \"state_date\": null,\n" +
		"    \"kind\": \"\",\n    \"ask_to\": \"\",\n    \"due\": \"\",\n    \"origin\": \"\",\n    \"decision\": \"\",\n" +
		"    \"problem\": \"a\\nb\\u0001\",\n    \"current_handling\": \"\",\n    \"options\": \"\",\n" +
		"    \"questions\": \"\",\n    \"after_decision\": \"\",\n    \"history\": \"\",\n    \"created_at\": \"\",\n" +
		"    \"updated_at\": \"\",\n    \"uuid\": \"\"\n  }\n]\n"
	assert.Equal(t, want, got)
	assert.Equal(t, "[]\n", string(CanonicalArrayBytes(nil)))
}

func TestPack_ContentHashStable(t *testing.T) {
	var items []Item
	for i := 0; i < 205; i++ { // three chunks
		items = append(items, mkItem("000-foundation", "main", fmt.Sprintf("%03d", i), fmt.Sprintf("item %d", i)))
	}
	// Input order must not matter: hash is computed over sorted items.
	shuffled := append([]Item{}, items[100:]...)
	shuffled = append(shuffled, items[:100]...)

	dir := t.TempDir()
	r1, err := Pack(items, dir)
	require.NoError(t, err)
	assert.Equal(t, 205, r1.ItemCount)
	assert.Equal(t, 3, r1.ChunkCount)
	m1, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	require.NoError(t, err)

	r2, err := Pack(shuffled, dir)
	require.NoError(t, err)
	assert.Equal(t, r1.ContentHash, r2.ContentHash)
	m2, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	require.NoError(t, err)
	assert.Equal(t, string(m1), string(m2), "generated_at is preserved when content is unchanged")

	man, err := LoadManifest(dir)
	require.NoError(t, err)
	assert.Equal(t, "backlog", man.KDB)
	assert.Equal(t, 100, man.ChunkSize)
	assert.Equal(t, "items/chunk-0002.json", man.Chunks[2].Path)
	assert.Equal(t, 5, man.Chunks[2].Count)

	// content_hash = sha256 of the concatenated chunk sha hex strings.
	var shas strings.Builder
	for _, c := range man.Chunks {
		shas.WriteString(c.SHA256)
	}
	assert.Equal(t, SHA256Hex([]byte(shas.String())), man.ContentHash)
}

func TestPack_RemovesStaleChunksAndRejectsDuplicates(t *testing.T) {
	var items []Item
	for i := 0; i < 150; i++ {
		items = append(items, mkItem("p", "b", fmt.Sprintf("%03d", i), "x"))
	}
	dir := t.TempDir()
	_, err := Pack(items, dir)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "items", "chunk-0001.json"))

	_, err = Pack(items[:10], dir)
	require.NoError(t, err)
	assert.NoFileExists(t, filepath.Join(dir, "items", "chunk-0001.json"))

	_, err = Pack([]Item{items[0], items[0]}, dir)
	require.Error(t, err)
	assert.Equal(t, 1, CodeOf(err))
}

func TestUnpack_RoundTripPreservesUUIDAndHash(t *testing.T) {
	var items []Item
	for i := 0; i < 120; i++ {
		items = append(items, mkItem("p", "b", fmt.Sprintf("%03d", i), fmt.Sprintf("t%d", i)))
	}
	items[3].StateDate = nil
	dir := t.TempDir()
	packed, err := Pack(items, dir)
	require.NoError(t, err)

	out := filepath.Join(t.TempDir(), "data", "items.json")
	res, err := Unpack(dir, out)
	require.NoError(t, err)
	assert.Equal(t, packed.ContentHash, res.ContentHash)

	back, err := ReadItems(out)
	require.NoError(t, err)
	assert.Equal(t, SortItems(items), back, "uuid and null state_date survive the round trip")
	assert.Equal(t, packed.ContentHash, ContentHashOf(back))
}

func TestUnpack_DetectsTampering(t *testing.T) {
	dir := t.TempDir()
	_, err := Pack([]Item{mkItem("p", "b", "001", "x")}, dir)
	require.NoError(t, err)
	chunk := filepath.Join(dir, "items", "chunk-0000.json")
	raw, _ := os.ReadFile(chunk)
	require.NoError(t, os.WriteFile(chunk, []byte(strings.Replace(string(raw), `"x"`, `"y"`, 1)), 0o644))
	_, err = Unpack(dir, filepath.Join(t.TempDir(), "o.json"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sha256 mismatch")

	_, err = Unpack(t.TempDir(), filepath.Join(t.TempDir(), "o.json"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manifest not found")
}

func TestUnpack_CRLFCheckoutStillVerifies(t *testing.T) {
	dir := t.TempDir()
	_, err := Pack([]Item{mkItem("p", "b", "001", "x")}, dir)
	require.NoError(t, err)
	chunk := filepath.Join(dir, "items", "chunk-0000.json")
	raw, _ := os.ReadFile(chunk)
	crlf := strings.ReplaceAll(string(raw), "\n", "\r\n")
	require.NoError(t, os.WriteFile(chunk, []byte(crlf), 0o644))
	_, err = Unpack(dir, filepath.Join(t.TempDir(), "o.json"))
	assert.NoError(t, err)
}

func TestMergeItems_UnionAndPrefer(t *testing.T) {
	a := mkItem("p", "b", "001", "local")
	aData := a
	aData.Title = "corpus"
	onlyTmp := mkItem("p", "b", "002", "only tmp")
	onlyData := mkItem("p", "b", "003", "only data")
	data := []Item{aData, onlyData}
	tmp := []Item{a, onlyTmp}

	merged, sum, err := MergeItems(data, tmp, "")
	require.NoError(t, err)
	assert.Equal(t, 1, sum.ConflictCount)
	assert.Equal(t, []string{"p/b/001"}, sum.Conflicts)
	assert.Len(t, merged, 4, "conflicting rows are both kept")

	merged, sum, err = MergeItems(data, tmp, "tmp")
	require.NoError(t, err)
	assert.Equal(t, 0, sum.ConflictCount)
	assert.Equal(t, "tmp", sum.Prefer)
	require.Len(t, merged, 3)
	assert.Equal(t, "local", merged[0].Title)
	assert.Equal(t, "p/b/002", KeyOf(merged[1]).String(), "sorted when conflict-free")

	merged, _, err = MergeItems(data, tmp, "data")
	require.NoError(t, err)
	assert.Equal(t, "corpus", merged[0].Title)

	_, _, err = MergeItems(data, tmp, "bogus")
	require.Error(t, err)
	assert.Equal(t, 2, CodeOf(err))
}

func TestMergeItems_IdenticalCollapsed(t *testing.T) {
	a := mkItem("p", "b", "001", "same")
	merged, sum, err := MergeItems([]Item{a}, []Item{a}, "")
	require.NoError(t, err)
	assert.Len(t, merged, 1)
	assert.Equal(t, 1, sum.Collapsed)
	assert.Equal(t, 0, sum.ConflictCount)
	assert.Equal(t, []string{}, sum.Conflicts)
}

func TestMergeFiles_ConflictWritesAndReturnsErr(t *testing.T) {
	dir := t.TempDir()
	a := mkItem("p", "b", "001", "x")
	b := a
	b.Title = "y"
	require.NoError(t, WriteItems(filepath.Join(dir, "tmp.json"), []Item{a}))
	require.NoError(t, WriteItems(filepath.Join(dir, "data.json"), []Item{b}))
	sum, err := MergeFiles(filepath.Join(dir, "tmp.json"), filepath.Join(dir, "data.json"), filepath.Join(dir, "data.json"), "")
	assert.ErrorIs(t, err, ErrMergeConflict)
	assert.Equal(t, 1, sum.ConflictCount)
	got, err := ReadItems(filepath.Join(dir, "data.json"))
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestReadItems_BOMEmptyAndMissing(t *testing.T) {
	dir := t.TempDir()
	items, err := ReadItems(filepath.Join(dir, "missing.json"))
	require.NoError(t, err)
	assert.Empty(t, items)

	p := filepath.Join(dir, "bom.json")
	require.NoError(t, os.WriteFile(p, append([]byte{0xEF, 0xBB, 0xBF}, []byte(`[{"phase":"p","branch":"b","local_id":"001","title":"t","uuid":"u"}]`)...), 0o644))
	items, err = ReadItems(p)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, StateUnconfirmed, items[0].State)

	require.NoError(t, os.WriteFile(p, []byte("  \n"), 0o644))
	items, err = ReadItems(p)
	require.NoError(t, err)
	assert.Empty(t, items)

	require.NoError(t, os.WriteFile(p, []byte(`{"a":1}`), 0o644))
	_, err = ReadItems(p)
	assert.Error(t, err)
	require.NoError(t, os.WriteFile(p, []byte(`[1]`), 0o644))
	_, err = ReadItems(p)
	assert.Error(t, err)
}

func TestListFilter(t *testing.T) {
	mk := func(id, title, problem, updated string) Item {
		it := mkItem("p", "b", id, title)
		it.Problem, it.UpdatedAt = problem, updated
		return it
	}
	items := []Item{
		mk("001", "Alpha", "about RLS", "2026-01-03"),
		mk("002", "Beta", "other", "2026-01-01"),
		mk("003", "Gamma", "rls again", "2026-01-02"),
	}
	page, err := ListFilter(items, ListOptions{Match: "rls"})
	require.NoError(t, err)
	assert.Equal(t, 2, page.Total, "case-insensitive match on problem")

	page, err = ListFilter(items, ListOptions{Sort: "updated_at", Order: "desc", Limit: "2", Offset: "1"})
	require.NoError(t, err)
	assert.Equal(t, 3, page.Total, "total is before limit/offset")
	require.Len(t, page.Items, 2)
	assert.Equal(t, "Gamma", page.Items[0].Title)
	assert.Equal(t, "Beta", page.Items[1].Title)
	require.NotNil(t, page.Limit)
	assert.Equal(t, 2, *page.Limit)

	page, err = ListFilter(items, ListOptions{Order: "desc"})
	require.NoError(t, err)
	assert.Equal(t, "003", page.Items[0].LocalID)

	page, err = ListFilter(items, ListOptions{Offset: "10"})
	require.NoError(t, err)
	assert.Empty(t, page.Items)
	assert.Nil(t, page.Limit)

	for _, bad := range []ListOptions{
		{Match: "("}, {Order: "up"}, {Sort: "nope"}, {Limit: "0"}, {Limit: "x"}, {Offset: "-1"},
	} {
		_, err := ListFilter(items, bad)
		require.Error(t, err, "%+v", bad)
		assert.Equal(t, 2, CodeOf(err))
	}
}
