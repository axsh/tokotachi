package backlog

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormItem_DefaultsAndTypes(t *testing.T) {
	m := map[string]any{
		"phase": "p", "branch": "b", "local_id": json.Number("7"), "title": "t",
		"state_date": "", "kind": nil, "problem": true,
	}
	it, err := NormItem(m)
	require.NoError(t, err)
	assert.Equal(t, "7", it.LocalID, "numbers are stringified")
	assert.Equal(t, StateUnconfirmed, it.State, "missing state defaults to 未確認")
	assert.Nil(t, it.StateDate, "empty state_date is null")
	assert.Equal(t, "", it.Kind)
	assert.Equal(t, "True", it.Problem, "bool follows Python str()")
}

func TestNormItem_StateDatePresentAndEmptyState(t *testing.T) {
	it, err := NormItem(map[string]any{"state": "", "state_date": "2026-01-02"})
	require.NoError(t, err)
	assert.Equal(t, "", it.State, "explicit empty state is kept (reported by distill)")
	require.NotNil(t, it.StateDate)
	assert.Equal(t, "2026-01-02", *it.StateDate)
}

func TestNormItem_RejectsNested(t *testing.T) {
	_, err := NormItem(map[string]any{"title": []any{"x"}})
	assert.Error(t, err)
}

func TestValidators(t *testing.T) {
	assert.True(t, ValidUUID("550e8400-e29b-41d4-a716-446655440000"))
	assert.True(t, ValidUUID(NewUUID()))
	assert.False(t, ValidUUID("550e8400-e29b-61d4-a716-446655440000"), "version 6 is rejected")
	assert.False(t, ValidUUID("not-a-uuid"))
	for _, s := range States {
		assert.True(t, ValidState(s))
	}
	assert.False(t, ValidState("done"))
	for _, k := range Kinds {
		assert.True(t, ValidKind(k))
	}
	assert.Equal(t, []string{"未確認", "確認中", "決着", "取り下げ"}, States)
	assert.Equal(t, []string{"仮の決定", "確認事項", "先送り"}, Kinds)

	assert.NoError(t, CheckLocalID("001"))
	assert.NoError(t, CheckLocalID("1234"))
	assert.Error(t, CheckLocalID("1"))
	assert.NoError(t, CheckName("phase", "000-foundation"))
	assert.Error(t, CheckName("phase", "bad name"))
	assert.NoError(t, CheckDate("2026-11-18"))
	assert.Error(t, CheckDate("2026-1-18"))

	err := CheckState("nope")
	require.Error(t, err)
	assert.Equal(t, 2, CodeOf(err))
	assert.True(t, strings.Contains(err.Error(), "invalid state"))
}

func TestItemEqual_NullVersusEmptyStateDate(t *testing.T) {
	a := Item{Phase: "p", StateDate: nil}
	b := Item{Phase: "p", StateDate: StrPtr("")}
	assert.False(t, a.Equal(b))
	assert.True(t, a.Equal(Item{Phase: "p"}))
}
