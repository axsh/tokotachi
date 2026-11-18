package track

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axsh/tokotachi/features/tt/internal/kdb/builtin"
	"github.com/axsh/tokotachi/features/tt/internal/kdb/paths"
	"github.com/axsh/tokotachi/features/tt/internal/kdb/registry"
)

func requireTools(t *testing.T) {
	t.Helper()
	for _, tool := range []string{"bash", "git"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not available", tool)
		}
	}
}

type fakeBuiltin struct {
	dirty    bool
	calls    []string
	failStep string
}

func (f *fakeBuiltin) Status(context.Context, paths.Paths) (json.RawMessage, error) {
	if f.dirty {
		return json.RawMessage(`{"ok":true,"dirty":true}`), nil
	}
	return json.RawMessage(`{"ok":true,"dirty":false}`), nil
}

func (f *fakeBuiltin) step(name string) error {
	f.calls = append(f.calls, name)
	if f.failStep == name {
		return errors.New("boom " + name)
	}
	return nil
}
func (f *fakeBuiltin) Distill(context.Context, paths.Paths) error { return f.step("distill") }
func (f *fakeBuiltin) Dump(context.Context, paths.Paths) error    { return f.step("dump") }
func (f *fakeBuiltin) Merge(context.Context, paths.Paths, string) error {
	return f.step("merge")
}
func (f *fakeBuiltin) Pack(context.Context, paths.Paths) error { return f.step("pack") }

func entryFor(id, statusCmd string) registry.Entry {
	return registry.Entry{
		ID:         id,
		ModulePath: "features/" + id,
		CorpusPath: "features/" + id + "/corpus",
		StatusCmd:  statusCmd,
		DistillCmd: "echo distill",
		DumpCmd:    "echo dump",
		MergeCmd:   "echo merge",
		PackCmd:    "echo pack",
	}
}

func newWorkspace(t *testing.T) string {
	t.Helper()
	ws := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = ws
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	run("init", "-q")
	run("config", "user.email", "t@example.com")
	run("config", "user.name", "t")
	for _, id := range []string{"backlog", "designdb"} {
		require.NoError(t, os.MkdirAll(filepath.Join(ws, "features", id, "corpus"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(ws, "features", id, "corpus", "a.md"), []byte("x"), 0o644))
	}
	run("add", "-A")
	run("commit", "-q", "-m", "init")
	return ws
}

func twoEntryFile() *registry.File {
	return &registry.File{Version: 1, KDBs: []registry.Entry{
		entryFor("backlog", "echo backlog-status"),
		entryFor("designdb", `echo '{"ok":true,"dirty":false}'`),
	}}
}

func TestList(t *testing.T) {
	r := List(twoEntryFile())
	assert.True(t, r.OK)
	assert.Len(t, r.Entries, 2)
	assert.NotNil(t, List(&registry.File{}).Entries)
}

func TestUnregistered(t *testing.T) {
	ws := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(ws, "features", "backlog", "corpus"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(ws, "features", "extra", "corpus"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(ws, "features", "nocorpus"), 0o755))

	r, err := Unregistered(ws, twoEntryFile())
	require.NoError(t, err)
	assert.Equal(t, []string{"extra"}, r.IDs)
}

func TestUnregistered_NoFeaturesDir(t *testing.T) {
	r, err := Unregistered(t.TempDir(), twoEntryFile())
	require.NoError(t, err)
	assert.Empty(t, r.IDs)
	assert.NotNil(t, r.IDs)
}

func TestStatusCmds(t *testing.T) {
	m := StatusCmds(twoEntryFile())
	assert.Equal(t, true, m["ok"])
	cmds := m["commands"].([]map[string]string)
	require.Len(t, cmds, 2)
	assert.Equal(t, "backlog", cmds[0]["id"])
	assert.Equal(t, "echo backlog-status", cmds[0]["status_cmd"])
}

func TestStatusAll_DelegateAndBuiltin(t *testing.T) {
	requireTools(t)
	fb := &fakeBuiltin{dirty: true}
	builtin.RegisterBacklog(fb, fb)
	t.Cleanup(func() { builtin.RegisterBacklog(nil, nil) })

	ws := newWorkspace(t)
	sts, anyFail := StatusAll(context.Background(), ws, twoEntryFile())
	require.False(t, anyFail)
	require.Len(t, sts, 2)
	assert.Equal(t, "backlog", sts[0].ID)
	assert.True(t, sts[0].Dirty)
	assert.JSONEq(t, `{"ok":true,"dirty":true}`, string(sts[0].Output))
	assert.False(t, sts[1].Dirty)
	assert.True(t, sts[1].OK)
}

func TestStatusAll_PartialFailureContinues(t *testing.T) {
	requireTools(t)
	builtin.RegisterBacklog(nil, nil) // backlog not wired -> fails
	ws := newWorkspace(t)
	sts, anyFail := StatusAll(context.Background(), ws, twoEntryFile())
	assert.True(t, anyFail)
	require.Len(t, sts, 2)
	assert.False(t, sts[0].OK)
	assert.NotEmpty(t, sts[0].Error)
	assert.True(t, sts[1].OK, "later entries must still be processed")
}

func TestStatusAll_NonJSONWrapped(t *testing.T) {
	requireTools(t)
	f := &registry.File{Version: 1, KDBs: []registry.Entry{entryFor("designdb", "echo plain-text")}}
	sts, anyFail := StatusAll(context.Background(), newWorkspace(t), f)
	assert.False(t, anyFail)
	assert.JSONEq(t, `"plain-text"`, string(sts[0].Output))
}

func TestStatusAll_OkFalseIsFailure(t *testing.T) {
	requireTools(t)
	f := &registry.File{Version: 1, KDBs: []registry.Entry{entryFor("designdb", `echo '{"ok":false,"error":"bad"}'`)}}
	sts, anyFail := StatusAll(context.Background(), newWorkspace(t), f)
	assert.True(t, anyFail)
	assert.Equal(t, "bad", sts[0].Error)
}

func TestSystematize_DirtyBuiltinRunsPipeline(t *testing.T) {
	requireTools(t)
	fb := &fakeBuiltin{dirty: true}
	builtin.RegisterBacklog(fb, fb)
	t.Cleanup(func() { builtin.RegisterBacklog(nil, nil) })

	ws := newWorkspace(t)
	res, code := Systematize(context.Background(), ws, twoEntryFile(), "")
	assert.Equal(t, 0, code)
	assert.True(t, res.OK)
	assert.Equal(t, []string{"distill", "dump", "merge", "pack"}, fb.calls)
	assert.Equal(t, []string{"features/backlog/corpus"}, res.AddCandidates)
	require.Len(t, res.Entries, 2)
	assert.Equal(t, "systematized", res.Entries[0].Action)
	assert.Equal(t, "skip", res.Entries[1].Action)
}

func TestSystematize_GitDirtyDelegate(t *testing.T) {
	requireTools(t)
	ws := newWorkspace(t)
	// Make designdb corpus dirty in git.
	require.NoError(t, os.WriteFile(filepath.Join(ws, "features", "designdb", "corpus", "new.md"), []byte("y"), 0o644))

	f := &registry.File{Version: 1, KDBs: []registry.Entry{entryFor("designdb", `echo '{"ok":true,"dirty":false}'`)}}
	res, code := Systematize(context.Background(), ws, f, "")
	assert.Equal(t, 0, code)
	assert.Equal(t, []string{"features/designdb/corpus"}, res.AddCandidates)
	assert.True(t, res.Entries[0].Dirty)
}

func TestSystematize_FailureContinuesAndExits1(t *testing.T) {
	requireTools(t)
	fb := &fakeBuiltin{dirty: true, failStep: "dump"}
	builtin.RegisterBacklog(fb, fb)
	t.Cleanup(func() { builtin.RegisterBacklog(nil, nil) })

	ws := newWorkspace(t)
	require.NoError(t, os.WriteFile(filepath.Join(ws, "features", "designdb", "corpus", "new.md"), []byte("y"), 0o644))

	res, code := Systematize(context.Background(), ws, twoEntryFile(), "")
	assert.Equal(t, 1, code)
	assert.False(t, res.OK)
	require.Len(t, res.Entries, 2)
	assert.False(t, res.Entries[0].OK)
	assert.Equal(t, "dump", res.Entries[0].Stage)
	assert.True(t, res.Entries[1].OK, "designdb must still be processed")
	assert.Equal(t, []string{"features/designdb/corpus"}, res.AddCandidates)
	assert.Equal(t, []string{"distill", "dump"}, fb.calls, "stages after failure must not run")
}

func TestSystematize_OnlyID(t *testing.T) {
	requireTools(t)
	ws := newWorkspace(t)
	res, code := Systematize(context.Background(), ws, twoEntryFile(), "designdb")
	assert.Equal(t, 0, code)
	require.Len(t, res.Entries, 1)
	assert.Equal(t, "designdb", res.Entries[0].ID)

	_, code = Systematize(context.Background(), ws, twoEntryFile(), "nope")
	assert.Equal(t, 2, code)
}

func TestSystematize_UnregisteredWarning(t *testing.T) {
	requireTools(t)
	ws := newWorkspace(t)
	require.NoError(t, os.MkdirAll(filepath.Join(ws, "features", "orphan", "corpus"), 0o755))
	res, _ := Systematize(context.Background(), ws, twoEntryFile(), "designdb")
	assert.Contains(t, res.Warnings, "unregistered corpus: features/orphan/corpus")
}
