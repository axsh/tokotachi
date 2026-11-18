package paths

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axsh/tokotachi/features/tt/internal/kdb/registry"
)

func TestResolveWorkspace_Priority(t *testing.T) {
	flagDir := t.TempDir()
	envDir := t.TempDir()
	blDir := t.TempDir()

	t.Setenv("TT_WORKSPACE", envDir)
	t.Setenv("BL_ROOT", blDir)

	got, err := ResolveWorkspace(flagDir)
	require.NoError(t, err)
	assert.Equal(t, flagDir, got)

	got, err = ResolveWorkspace("")
	require.NoError(t, err)
	assert.Equal(t, envDir, got)

	t.Setenv("TT_WORKSPACE", "")
	got, err = ResolveWorkspace("")
	require.NoError(t, err)
	assert.Equal(t, blDir, got)

	t.Setenv("BL_ROOT", "")
	got, err = ResolveWorkspace("")
	require.NoError(t, err)
	assert.NotEmpty(t, got)
}

func TestRegistryPath_Priority(t *testing.T) {
	t.Setenv("KDB_REGISTRY", "")
	ws := filepath.Join("a", "ws")
	assert.Equal(t, filepath.Join(ws, "features", "kdb", "registry.yaml"), RegistryPath(ws, ""))

	t.Setenv("KDB_REGISTRY", "env.yaml")
	assert.Equal(t, "env.yaml", RegistryPath(ws, ""))
	assert.Equal(t, "flag.yaml", RegistryPath(ws, "flag.yaml"))
}

func TestForEntry_FeaturesLayout(t *testing.T) {
	ws := t.TempDir()
	e := registry.Entry{ID: "backlog", ModulePath: "features/backlog", CorpusPath: "features/backlog/corpus"}
	p, err := ForEntry(ResolveInput{Workspace: ws, Today: "2026-01-02"}, e)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(ws, "features", "backlog", "corpus"), p.CorpusDir)
	assert.Equal(t, filepath.Join(ws, "features", "backlog", "data"), p.DataDir)
	assert.Equal(t, filepath.Join(ws, "features", "backlog", "data", "backlog.sqlite"), p.DBPath)
	assert.Equal(t, filepath.Join(ws, "features", "backlog", "data", "items.json"), p.DataJSON)
	assert.Equal(t, filepath.Join(ws, "tmp", "kdb", "backlog"), p.TmpDir)
	assert.Equal(t, filepath.Join(ws, "tmp", "kdb", "backlog", "items.json"), p.TmpJSON)
	assert.Equal(t, "2026-01-02", p.Today)
}

func TestForEntry_ScriptsLayout(t *testing.T) {
	ws := t.TempDir()
	e := registry.Entry{ID: "backlog", ModulePath: "scripts/kdb/backlog", CorpusPath: "scripts/kdb/backlog/corpus"}
	p, err := ForEntry(ResolveInput{Workspace: ws}, e)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(ws, "scripts", "kdb", "backlog", "data"), p.DataDir)
	assert.Equal(t, filepath.Join(filepath.Dir(p.CorpusDir), "data"), p.DataDir)
	assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, p.Today)
}

func TestForEntry_Overrides(t *testing.T) {
	ws := t.TempDir()
	other := t.TempDir()
	e := registry.Entry{ID: "backlog", ModulePath: "features/backlog", CorpusPath: "features/backlog/corpus"}
	p, err := ForEntry(ResolveInput{
		Workspace: ws,
		DataDir:   filepath.Join(other, "d"),
		DBPath:    filepath.Join(other, "x.sqlite"),
		CorpusDir: filepath.Join(other, "c"),
		TmpDir:    filepath.Join(other, "t"),
	}, e)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(other, "d"), p.DataDir)
	assert.Equal(t, filepath.Join(other, "x.sqlite"), p.DBPath)
	assert.Equal(t, filepath.Join(other, "c"), p.CorpusDir)
	assert.Equal(t, filepath.Join(other, "t"), p.TmpDir)
	assert.Equal(t, filepath.Join(other, "d", "items.json"), p.DataJSON)
}

func TestForEntry_NonBacklogHasNoDBPath(t *testing.T) {
	ws := t.TempDir()
	e := registry.Entry{ID: "designdb", ModulePath: "features/designdb", CorpusPath: "features/designdb/corpus"}
	p, err := ForEntry(ResolveInput{Workspace: ws}, e)
	require.NoError(t, err)
	assert.Empty(t, p.DBPath)
}

func TestInputFromEnv(t *testing.T) {
	t.Setenv("BL_DATA_DIR", "dd")
	t.Setenv("BL_TODAY", "2026-05-05")
	in := InputFromEnv("ws", "reg")
	assert.Equal(t, "dd", in.DataDir)
	assert.Equal(t, "2026-05-05", in.Today)
	assert.Equal(t, "ws", in.Workspace)
}
