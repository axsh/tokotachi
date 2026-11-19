package backlog

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axsh/tokotachi/features/tt/internal/kdb/paths"
)

func newOps(t *testing.T) Ops {
	t.Helper()
	root := t.TempDir()
	return Ops{P: paths.Paths{
		Workspace: root,
		CorpusDir: filepath.Join(root, "features", "backlog", "corpus"),
		DataDir:   filepath.Join(root, "features", "backlog", "data"),
		DBPath:    filepath.Join(root, "features", "backlog", "data", "backlog.sqlite"),
		TmpDir:    filepath.Join(root, "tmp", "kdb", "backlog"),
		DataJSON:  filepath.Join(root, "features", "backlog", "data", "items.json"),
		TmpJSON:   filepath.Join(root, "tmp", "kdb", "backlog", "items.json"),
		Today:     "2026-11-18",
	}}
}

func addItem(t *testing.T, o Ops, title string, extra ...string) Item {
	t.Helper()
	args := append([]string{"--phase", "000-foundation", "--branch", "main", "--title", title, "--kind", "確認事項"}, extra...)
	f, _, err := ParseFlags(AddFlags, args)
	require.NoError(t, err)
	res, err := o.Add(f)
	require.NoError(t, err)
	return res["item"].(Item)
}

func flagsOf(t *testing.T, allowed []string, args ...string) Flags {
	t.Helper()
	f, _, err := ParseFlags(allowed, args)
	require.NoError(t, err)
	return f
}

func TestParseFlags(t *testing.T) {
	f, pos, err := ParseFlags([]string{"a", "b"}, []string{"--a", "1", "--b=2", "x"})
	require.NoError(t, err)
	assert.Equal(t, "1", f.Get("a"))
	assert.Equal(t, "2", f.Get("b"))
	assert.Equal(t, []string{"x"}, pos)

	_, _, err = ParseFlags([]string{"a"}, []string{"--zzz", "1"})
	assert.Equal(t, 2, CodeOf(err))
	assert.Contains(t, err.Error(), "unknown flag --zzz")
	_, _, err = ParseFlags([]string{"a"}, []string{"--a"})
	assert.Contains(t, err.Error(), "requires a value")
	f, _, err = ParseFlags([]string{"a"}, []string{"--a", ""})
	require.NoError(t, err)
	assert.True(t, f.Has("a"))
	assert.Error(t, f.Need("a"), "empty value fails Need")
}

func TestOps_AddDefaultsAndAutoID(t *testing.T) {
	o := newOps(t)
	a := addItem(t, o, "first")
	b := addItem(t, o, "second", "--problem", "p")
	assert.Equal(t, "000", a.LocalID)
	assert.Equal(t, "001", b.LocalID)
	assert.Equal(t, StateUnconfirmed, a.State)
	require.NotNil(t, a.StateDate)
	assert.Equal(t, "2026-11-18", *a.StateDate)
	assert.Equal(t, "2026-11-18", a.CreatedAt)
	assert.Equal(t, "2026-11-18", a.UpdatedAt)
	assert.True(t, ValidUUID(a.UUID))
	assert.NotEqual(t, a.UUID, b.UUID)
	assert.Equal(t, "p", b.Problem)

	// Explicit duplicate id fails with exit 1.
	f := flagsOf(t, AddFlags, "--phase", "000-foundation", "--branch", "main", "--title", "x", "--kind", "先送り", "--id", "000")
	_, err := o.Add(f)
	require.Error(t, err)
	assert.Equal(t, 1, CodeOf(err))
}

func TestOps_AddUsageErrors(t *testing.T) {
	o := newOps(t)
	cases := [][]string{
		{"--branch", "b", "--title", "t", "--kind", "確認事項"},                       // phase missing
		{"--phase", "p", "--branch", "b", "--title", "t", "--kind", "bad"},           // bad kind
		{"--phase", "p q", "--branch", "b", "--title", "t", "--kind", "確認事項"},      // bad name
		{"--phase", "p", "--branch", "b", "--title", "t", "--kind", "確認事項", "--state", "x"},
		{"--phase", "p", "--branch", "b", "--title", "t", "--kind", "確認事項", "--state-date", "2026"},
		{"--phase", "p", "--branch", "b", "--title", "t", "--kind", "確認事項", "--id", "1"},
	}
	for _, args := range cases {
		f := flagsOf(t, AddFlags, args...)
		_, err := o.Add(f)
		require.Error(t, err, "%v", args)
		assert.Equal(t, 2, CodeOf(err), "%v", args)
	}
}

func TestOps_GetByUUIDAndKey(t *testing.T) {
	o := newOps(t)
	a := addItem(t, o, "first")
	other := addItem(t, o, "second")

	res, err := o.Get(flagsOf(t, GetFlags, "--uuid", a.UUID))
	require.NoError(t, err)
	assert.Equal(t, a, res["item"])

	res, err = o.Get(flagsOf(t, GetFlags, "--phase", a.Phase, "--branch", a.Branch, "--id", a.LocalID))
	require.NoError(t, err)
	assert.Equal(t, a, res["item"])

	// Upper-case uuid is accepted (lower-cased for lookup).
	res, err = o.Get(flagsOf(t, GetFlags, "--uuid", upper(a.UUID)))
	require.NoError(t, err)
	assert.Equal(t, a, res["item"])

	// uuid and key of different items -> exit 1.
	_, err = o.Get(flagsOf(t, GetFlags, "--uuid", a.UUID, "--phase", other.Phase, "--branch", other.Branch, "--id", other.LocalID))
	require.Error(t, err)
	assert.Equal(t, 1, CodeOf(err))
	assert.Contains(t, err.Error(), "does not match")

	_, err = o.Get(flagsOf(t, GetFlags))
	assert.Equal(t, 2, CodeOf(err))
	_, err = o.Get(flagsOf(t, GetFlags, "--uuid", "nope"))
	assert.Equal(t, 2, CodeOf(err))
	_, err = o.Get(flagsOf(t, GetFlags, "--phase", "p"))
	assert.Equal(t, 2, CodeOf(err), "incomplete key")
	_, err = o.Get(flagsOf(t, GetFlags, "--uuid", "550e8400-e29b-41d4-a716-446655440000"))
	assert.Equal(t, 1, CodeOf(err), "unknown uuid")
}

func upper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'f' {
			b[i] = c - 32
		}
	}
	return string(b)
}

func TestOps_UpdateTransitions(t *testing.T) {
	o := newOps(t)
	a := addItem(t, o, "first", "--history", "h1")
	key := []string{"--phase", a.Phase, "--branch", a.Branch, "--id", a.LocalID}
	upd := func(extra ...string) (map[string]any, error) {
		return o.Update(flagsOf(t, UpdateFlags, append(append([]string{}, key...), extra...)...))
	}

	res, err := upd("--state", "確認中", "--append-history", "h2")
	require.NoError(t, err)
	it := res["item"].(Item)
	assert.Equal(t, "確認中", it.State)
	assert.Equal(t, "h1\nh2", it.History)
	assert.Equal(t, a.UUID, it.UUID, "uuid never changes")

	// append-history on empty history does not add a leading newline.
	b := addItem(t, o, "second")
	res, err = o.Update(flagsOf(t, UpdateFlags, "--phase", b.Phase, "--branch", b.Branch, "--id", b.LocalID, "--append-history", "first"))
	require.NoError(t, err)
	assert.Equal(t, "first", res["item"].(Item).History)

	// state-date empty clears to null.
	res, err = upd("--state-date", "")
	require.NoError(t, err)
	assert.Nil(t, res["item"].(Item).StateDate)

	for _, bad := range [][]string{
		{"--state", "決着"},
		{"--state", "取り下げ"},
		{"--history", "x", "--append-history", "y"},
		{"--title", ""},
		{},
		{"--kind", "nope"},
		{"--state-date", "bad"},
	} {
		_, err := upd(bad...)
		require.Error(t, err, "%v", bad)
		assert.Equal(t, 2, CodeOf(err), "%v", bad)
	}

	_, err = o.Update(flagsOf(t, UpdateFlags, "--phase", "x", "--branch", "y", "--id", "001", "--title", "t"))
	assert.Equal(t, 1, CodeOf(err), "missing item")
}

func TestOps_Resolve(t *testing.T) {
	o := newOps(t)
	a := addItem(t, o, "first")
	key := []string{"--phase", a.Phase, "--branch", a.Branch, "--id", a.LocalID}

	_, err := o.Resolve(flagsOf(t, ResolveFlags, append(key, "--state", "確認中")...))
	assert.Equal(t, 2, CodeOf(err), "resolve only closes")

	res, err := o.Resolve(flagsOf(t, ResolveFlags, append(key, "--state", "決着", "--decision", "go")...))
	require.NoError(t, err)
	it := res["item"].(Item)
	assert.Equal(t, "決着", it.State)
	assert.Equal(t, "go", it.Decision)
	assert.Equal(t, "2026-11-18", *it.StateDate)

	// Without --decision the column is left unchanged.
	res, err = o.Resolve(flagsOf(t, ResolveFlags, append(key, "--state", "取り下げ")...))
	require.NoError(t, err)
	assert.Equal(t, "go", res["item"].(Item).Decision)
	assert.Equal(t, "取り下げ", res["item"].(Item).State)

	_, err = o.Resolve(flagsOf(t, ResolveFlags, "--phase", "x", "--branch", "y", "--id", "001", "--state", "決着"))
	assert.Equal(t, 1, CodeOf(err))
}

func TestOps_ListFilters(t *testing.T) {
	o := newOps(t)
	a := addItem(t, o, "Alpha rule")
	addItem(t, o, "Beta", "--kind", "先送り")
	c := addItem(t, o, "Gamma")
	_, err := o.Resolve(flagsOf(t, ResolveFlags, "--phase", c.Phase, "--branch", c.Branch, "--id", c.LocalID, "--state", "決着"))
	require.NoError(t, err)

	total := func(args ...string) (int, []Item) {
		res, err := o.List(flagsOf(t, ListFlags, args...))
		require.NoError(t, err)
		return res["total"].(int), res["items"].([]Item)
	}
	n, _ := total()
	assert.Equal(t, 2, n, "closed items are excluded by default")
	n, _ = total("--include-closed", "1")
	assert.Equal(t, 3, n)
	n, items := total("--state", "決着")
	assert.Equal(t, 1, n, "explicit --state returns closed items")
	assert.Equal(t, c.UUID, items[0].UUID)
	n, items = total("--match", "RULE")
	assert.Equal(t, 1, n)
	assert.Equal(t, a.UUID, items[0].UUID)
	n, _ = total("--kind", "先送り")
	assert.Equal(t, 1, n)
	n, items = total("--include-closed", "1", "--limit", "1", "--offset", "1", "--sort", "title", "--order", "desc")
	assert.Equal(t, 3, n)
	require.Len(t, items, 1)
	assert.Equal(t, "Beta", items[0].Title)

	_, err = o.List(flagsOf(t, ListFlags, "--state", "bad"))
	assert.Equal(t, 2, CodeOf(err))
	_, err = o.List(flagsOf(t, ListFlags, "--match", "("))
	assert.Equal(t, 2, CodeOf(err))
}

// Full pipeline: distill -> dump -> merge -> pack -> unpack -> restore keeps uuids,
// and the content hash is stable across repeated pack runs.
func TestOps_PipelineRoundTrip(t *testing.T) {
	o := newOps(t)
	a := addItem(t, o, "first")
	b := addItem(t, o, "second")

	_, err := o.Distill()
	require.NoError(t, err)
	_, err = o.Dump()
	require.NoError(t, err)
	res, err := o.Merge("")
	require.NoError(t, err)
	data := res["data"].(map[string]any)
	assert.Equal(t, true, data["restored"])
	assert.Equal(t, 0, data["merge"].(MergeSummary).ConflictCount)

	res, err = o.Pack()
	require.NoError(t, err)
	hash1 := res["data"].(PackResult).ContentHash
	res, err = o.Pack()
	require.NoError(t, err)
	assert.Equal(t, hash1, res["data"].(PackResult).ContentHash)

	rep, err := o.StatusReport()
	require.NoError(t, err)
	assert.False(t, rep.Dirty, "reasons: %v", rep.Reasons)
	assert.Equal(t, hash1, rep.CorpusHash)
	assert.Equal(t, hash1, rep.DBHash)
	assert.Equal(t, 2, rep.ItemCount)

	// Local change makes the status dirty.
	_, err = o.Update(flagsOf(t, UpdateFlags, "--phase", a.Phase, "--branch", a.Branch, "--id", a.LocalID, "--title", "changed"))
	require.NoError(t, err)
	rep, err = o.StatusReport()
	require.NoError(t, err)
	assert.True(t, rep.Dirty)
	assert.NotEmpty(t, rep.Reasons)

	// Wipe the cache + data, then rebuild from the corpus: uuid and hash survive.
	require.NoError(t, os.Remove(o.P.DataJSON))
	require.NoError(t, os.Remove(o.P.DBPath))
	_, err = o.Unpack()
	require.NoError(t, err)
	res, err = o.Restore()
	require.NoError(t, err)
	assert.Equal(t, 2, res["data"].(map[string]any)["restored"])
	got, err := o.Get(flagsOf(t, GetFlags, "--uuid", b.UUID))
	require.NoError(t, err)
	assert.Equal(t, b, got["item"])
	rep, err = o.StatusReport()
	require.NoError(t, err)
	assert.False(t, rep.Dirty, "reasons: %v", rep.Reasons)
}

func TestOps_MergeConflictAbsorbed(t *testing.T) {
	o := newOps(t)
	a := addItem(t, o, "local")
	_, err := o.Dump()
	require.NoError(t, err)
	// The corpus side has the same key with different content.
	other := a
	other.Title = "from corpus"
	require.NoError(t, WriteItems(o.P.DataJSON, []Item{other}))

	res, err := o.Merge("")
	require.NoError(t, err, "merge CLI contract: conflicts are not an error")
	data := res["data"].(map[string]any)
	assert.Equal(t, false, data["restored"])
	assert.Equal(t, 1, data["merge"].(MergeSummary).ConflictCount)

	// The internal runner reports the conflict (used by systematize).
	_, err = o.MergeResult("")
	assert.ErrorIs(t, err, ErrMergeConflict)

	// distill and pack now refuse the conflicting data/items.json.
	_, err = o.Distill()
	require.Error(t, err)
	assert.Equal(t, 1, CodeOf(err))
	_, err = o.Pack()
	require.Error(t, err)

	// The conflicting merge left both rows in data/items.json (same as the former
	// merge.sh); the documented recovery is to restore data (unpack) and merge with --prefer.
	require.NoError(t, WriteItems(o.P.DataJSON, []Item{other}))
	res, err = o.Merge("data")
	require.NoError(t, err)
	assert.Equal(t, true, res["data"].(map[string]any)["restored"])
	got, err := o.Get(flagsOf(t, GetFlags, "--uuid", a.UUID))
	require.NoError(t, err)
	assert.Equal(t, "from corpus", got["item"].(Item).Title)
}

func TestOps_MergeRequiresDump(t *testing.T) {
	o := newOps(t)
	_, err := o.Merge("")
	require.Error(t, err)
	assert.Equal(t, 1, CodeOf(err))
	assert.Contains(t, err.Error(), "dump not found")
	_, err = o.Merge("bogus")
	assert.Equal(t, 2, CodeOf(err))
}

func TestOps_DistillCollapsesIdenticalDuplicatesInData(t *testing.T) {
	o := newOps(t)
	it := mkItem("p", "b", "001", "x")
	require.NoError(t, WriteItems(o.P.DataJSON, []Item{it, it}))
	res, err := o.Distill()
	require.NoError(t, err)
	rep := res["data"].(map[string]any)["data_json"].(DistillReport)
	assert.Equal(t, 1, rep.Collapsed)
	assert.True(t, rep.Rewritten)
	back, err := ReadItems(o.P.DataJSON)
	require.NoError(t, err)
	assert.Len(t, back, 1)
}

func TestOps_RestoreAndPackNeedDataJSON(t *testing.T) {
	o := newOps(t)
	_, err := o.Restore()
	require.Error(t, err)
	assert.Equal(t, 1, CodeOf(err))
	_, err = o.Pack()
	require.Error(t, err)
	_, err = o.Unpack()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manifest not found")
}

func TestOps_MigrateUUIDs(t *testing.T) {
	o := newOps(t)
	in := filepath.Join(t.TempDir(), "legacy.json")
	valid := "550e8400-e29b-41d4-a716-446655440000"
	require.NoError(t, os.WriteFile(in, []byte(`[
	  {"phase":"p","branch":"b","local_id":"001","title":"a","uuid":"`+valid+`","logical_uuid":"old"},
	  {"phase":"p","branch":"b","local_id":"002","title":"b"},
	  {"phase":"p","branch":"b","local_id":"003","title":"c","uuid":"garbage"}
	]`), 0o644))
	res, err := o.MigrateUUIDs(flagsOf(t, []string{"in", "out"}, "--in", in))
	require.NoError(t, err)
	assert.Equal(t, 3, res["count"])
	assert.Equal(t, 2, res["assigned"])
	assert.Equal(t, 1, res["logical_uuid_removed"])
	items, err := ReadItems(in)
	require.NoError(t, err)
	assert.Equal(t, valid, items[0].UUID, "valid uuid is kept")
	assert.True(t, ValidUUID(items[1].UUID))
	assert.True(t, ValidUUID(items[2].UUID))
	assert.Empty(t, FindProblems(items))
	raw, _ := os.ReadFile(in)
	var generic []map[string]any
	require.NoError(t, json.Unmarshal(raw, &generic))
	assert.NotContains(t, generic[0], "logical_uuid")

	_, err = o.MigrateUUIDs(flagsOf(t, []string{"in", "out"}, "--in", in, "--out", "-"))
	assert.Equal(t, 2, CodeOf(err))
	_, err = o.MigrateUUIDs(flagsOf(t, []string{"in", "out"}))
	assert.Equal(t, 2, CodeOf(err))
}

func TestAdapters_StatusAndPipeline(t *testing.T) {
	o := newOps(t)
	addItem(t, o, "first")
	ctx := context.Background()

	raw, err := statusAdapter{}.Status(ctx, o.P)
	require.NoError(t, err)
	var st struct {
		OK   bool `json:"ok"`
		Data struct {
			Dirty bool `json:"dirty"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &st))
	assert.True(t, st.OK)
	assert.True(t, st.Data.Dirty, "no corpus yet")

	var pa pipelineAdapter
	require.NoError(t, pa.Distill(ctx, o.P))
	require.NoError(t, pa.Dump(ctx, o.P))
	require.NoError(t, pa.Merge(ctx, o.P, ""))
	require.NoError(t, pa.Pack(ctx, o.P))

	raw, err = statusAdapter{}.Status(ctx, o.P)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &st))
	assert.False(t, st.Data.Dirty)
}

func TestAdapters_DistillProblemsInMessage(t *testing.T) {
	o := newOps(t)
	s, err := Open(o.P.DBPath)
	require.NoError(t, err)
	bad := mkItem("p", "b", "001", "x")
	bad.State = "weird"
	require.NoError(t, s.Insert(bad))
	s.Close()
	err = pipelineAdapter{}.Distill(context.Background(), o.P)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid state: weird")
}
