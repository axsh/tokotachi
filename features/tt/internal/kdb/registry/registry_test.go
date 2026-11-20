package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func write(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "registry.yaml")
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

func entry(id string) string {
	return "  - id: " + id + "\n" +
		"    module_path: features/" + id + "\n" +
		"    corpus_path: features/" + id + "/corpus\n" +
		"    status_cmd: echo status\n" +
		"    distill_cmd: echo distill\n" +
		"    dump_cmd: echo dump\n" +
		"    merge_cmd: echo merge\n" +
		"    pack_cmd: echo pack\n"
}

func TestLoadValidate_TwoEntries(t *testing.T) {
	p := write(t, "version: 1\nunknown_key: x\nkdbs:\n"+entry("backlog")+entry("designdb"))
	f, err := Load(p)
	require.NoError(t, err)
	require.NoError(t, f.Validate())
	assert.Len(t, f.KDBs, 2)

	e, ok := f.ByID("designdb")
	require.True(t, ok)
	assert.Equal(t, "features/designdb/corpus", e.CorpusPath)

	_, ok = f.ByID("nope")
	assert.False(t, ok)
}

func TestValidate_MissingVersion(t *testing.T) {
	p := write(t, "kdbs:\n"+entry("backlog"))
	f, err := Load(p)
	require.NoError(t, err)
	assert.Error(t, f.Validate())
}

func TestValidate_MissingField(t *testing.T) {
	p := write(t, "version: 1\nkdbs:\n  - id: x\n    module_path: a\n    corpus_path: b\n")
	f, err := Load(p)
	require.NoError(t, err)
	err = f.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status_cmd")
}

func TestValidate_DuplicateID(t *testing.T) {
	p := write(t, "version: 1\nkdbs:\n"+entry("a")+entry("a"))
	f, err := Load(p)
	require.NoError(t, err)
	err = f.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "none.yaml"))
	assert.Error(t, err)
}

func TestLoadOptional_Missing(t *testing.T) {
	f, missing, err := LoadOptional(filepath.Join(t.TempDir(), "none.yaml"))
	require.NoError(t, err)
	assert.True(t, missing)
	require.NotNil(t, f)
	assert.Equal(t, 1, f.Version)
	assert.Empty(t, f.KDBs)
}

func TestLoadOptional_Present(t *testing.T) {
	p := write(t, "version: 1\nkdbs:\n"+entry("backlog"))
	f, missing, err := LoadOptional(p)
	require.NoError(t, err)
	assert.False(t, missing)
	require.NoError(t, f.Validate())
	assert.Len(t, f.KDBs, 1)
}

func TestLoadOptional_Corrupt(t *testing.T) {
	_, missing, err := LoadOptional(write(t, "version: [1\n"))
	assert.Error(t, err)
	assert.False(t, missing)
}

func TestLoad_BadYAML(t *testing.T) {
	_, err := Load(write(t, "version: [1\n"))
	assert.Error(t, err)
}