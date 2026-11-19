package backlog

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTemp(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "sub", "backlog.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

func TestStore_UUIDAndNextID(t *testing.T) {
	s := openTemp(t)
	id, err := s.NextLocalID("p", "b")
	require.NoError(t, err)
	assert.Equal(t, "000", id)

	a := mkItem("p", "b", "000", "a")
	b := mkItem("p", "b", "005", "b")
	b.StateDate = nil
	require.NoError(t, s.Insert(a))
	require.NoError(t, s.Insert(b))

	id, err = s.NextLocalID("p", "b")
	require.NoError(t, err)
	assert.Equal(t, "006", id)
	id, err = s.NextLocalID("p", "other")
	require.NoError(t, err)
	assert.Equal(t, "000", id)

	got, ok, err := s.GetByUUID(b.UUID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, b, got, "null state_date survives SQLite")
	got, ok, err = s.GetByKey("p", "b", "000")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, a, got)
	_, ok, err = s.GetByKey("p", "b", "999")
	require.NoError(t, err)
	assert.False(t, ok)

	// Duplicate key and duplicate uuid both fail with UNIQUE.
	assert.ErrorContains(t, s.Insert(mkItem("p", "b", "000", "dup key")), "UNIQUE")
	dupUUID := mkItem("p", "b", "010", "dup uuid")
	dupUUID.UUID = a.UUID
	assert.ErrorContains(t, s.Insert(dupUUID), "UNIQUE")
}

func TestStore_UpdateReplaceAllAndList(t *testing.T) {
	s := openTemp(t)
	a := mkItem("p", "b", "001", "a")
	require.NoError(t, s.Insert(a))
	a.Title = "renamed"
	a.State = "確認中"
	require.NoError(t, s.Update(a))
	got, _, err := s.GetByKey("p", "b", "001")
	require.NoError(t, err)
	assert.Equal(t, a, got)
	assert.Error(t, s.Update(mkItem("p", "b", "404", "missing")))

	items := []Item{mkItem("q", "b", "002", "x"), mkItem("p", "b", "009", "y")}
	require.NoError(t, s.ReplaceAll(items))
	all, err := s.ListAll()
	require.NoError(t, err)
	require.Len(t, all, 2)
	assert.Equal(t, "p/b/009", KeyOf(all[0]).String(), "ordered by phase, branch, local_id")

	// A failing restore (duplicate key) rolls back and keeps the old rows.
	err = s.ReplaceAll([]Item{mkItem("z", "z", "001", "n1"), mkItem("z", "z", "001", "n2")})
	require.Error(t, err)
	n, err := s.Count()
	require.NoError(t, err)
	assert.Equal(t, 2, n)
}

func TestStore_DumpJSONRoundTrip(t *testing.T) {
	s := openTemp(t)
	items := []Item{mkItem("p", "b", "001", "a <b> & c"), mkItem("p", "b", "002", "日本語")}
	require.NoError(t, s.ReplaceAll(items))
	out := filepath.Join(t.TempDir(), "tmp", "items.json")
	n, err := s.DumpJSON(out)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	back, err := ReadItems(out)
	require.NoError(t, err)
	assert.Equal(t, SortItems(items), back)
}

func TestSchemaEmbedded(t *testing.T) {
	assert.Contains(t, Schema(), "CREATE TABLE IF NOT EXISTS items")
	assert.Contains(t, Schema(), "items_uuid")
}
