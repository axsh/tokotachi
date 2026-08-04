package manifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsTaggableKind(t *testing.T) {
	assert.True(t, IsTaggableKind("policy"))
	assert.True(t, IsTaggableKind("procedure"))
	assert.True(t, IsTaggableKind("capability"))
	assert.True(t, IsTaggableKind("skip"))
	assert.False(t, IsTaggableKind("guard"))
	assert.False(t, IsTaggableKind("worker"))
	assert.False(t, IsTaggableKind("bundle"))
	assert.False(t, IsTaggableKind("target"))
}

func TestEffectiveTags_Omitted(t *testing.T) {
	e := &Entity{ID: "a", Kind: "capability", Raw: map[string]any{"id": "a"}}
	tags, err := EffectiveTags(e)
	require.NoError(t, err)
	assert.Equal(t, []string{BaselineTag}, tags)
}

func TestEffectiveTags_Scalar(t *testing.T) {
	e := &Entity{ID: "a", Kind: "capability", Raw: map[string]any{"tags": "test"}}
	tags, err := EffectiveTags(e)
	require.NoError(t, err)
	assert.Equal(t, []string{"test"}, tags)
}

func TestEffectiveTags_Array(t *testing.T) {
	e := &Entity{
		ID:   "a",
		Kind: "capability",
		Raw:  map[string]any{"tags": []any{"baseline", "test"}},
	}
	tags, err := EffectiveTags(e)
	require.NoError(t, err)
	assert.Equal(t, []string{"baseline", "test"}, tags)
}

func TestEffectiveTags_EmptyArray(t *testing.T) {
	e := &Entity{ID: "a", Kind: "capability", Raw: map[string]any{"tags": []any{}}}
	_, err := EffectiveTags(e)
	require.Error(t, err)
}

func TestEffectiveTags_InvalidName(t *testing.T) {
	for _, name := range []string{"Foo", "a_b"} {
		e := &Entity{ID: "a", Kind: "capability", Raw: map[string]any{"tags": name}}
		_, err := EffectiveTags(e)
		require.Error(t, err, "name=%s", name)
	}
}

func TestNormalizeRequestedTags_TrimDedup(t *testing.T) {
	tags, warnings, err := NormalizeRequestedTags("baseline, test, baseline")
	require.NoError(t, err)
	assert.Equal(t, []string{"baseline", "test"}, tags)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "duplicate")
}

func TestNormalizeTagRefsMode(t *testing.T) {
	m, err := NormalizeTagRefsMode("")
	require.NoError(t, err)
	assert.Equal(t, TagRefsInclude, m)

	m, err = NormalizeTagRefsMode("strict")
	require.NoError(t, err)
	assert.Equal(t, TagRefsStrict, m)

	_, err = NormalizeTagRefsMode("foo")
	require.Error(t, err)
}

func TestApplyTagSelection_OR(t *testing.T) {
	a := &Entity{ID: "a", Kind: "capability", Raw: map[string]any{"tags": []any{"baseline"}}}
	b := &Entity{ID: "b", Kind: "capability", Raw: map[string]any{"tags": []any{"test"}}}
	errs := ApplyTagSelection([]*Entity{a, b}, []string{"test"}, TagRefsInclude)
	require.Empty(t, errs)
	assert.False(t, a.Selected)
	assert.True(t, b.Selected)
}

func TestApplyTagSelection_IncludeClosure(t *testing.T) {
	x := &Entity{ID: "x", Kind: "capability", Raw: map[string]any{"tags": []any{"test"}}}
	p := &Entity{
		ID:   "p",
		Kind: "procedure",
		Raw: map[string]any{
			"tags":              []any{"baseline"},
			"uses_capabilities": []any{"x"},
		},
	}
	errs := ApplyTagSelection([]*Entity{p, x}, []string{"baseline"}, TagRefsInclude)
	require.Empty(t, errs)
	assert.True(t, p.Selected)
	assert.True(t, x.Selected)
}

func TestApplyTagSelection_StrictError(t *testing.T) {
	x := &Entity{ID: "x", Kind: "capability", Raw: map[string]any{"tags": []any{"test"}}}
	p := &Entity{
		ID:   "p",
		Kind: "procedure",
		Raw: map[string]any{
			"tags":              []any{"baseline"},
			"uses_capabilities": []any{"x"},
		},
	}
	errs := ApplyTagSelection([]*Entity{p, x}, []string{"baseline"}, TagRefsStrict)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Message, "strict")
	assert.True(t, p.Selected)
	assert.False(t, x.Selected)
}

func TestApplyTagSelection_NonTaggableAlways(t *testing.T) {
	g := &Entity{ID: "g", Kind: "guard", Raw: map[string]any{}}
	w := &Entity{ID: "w", Kind: "worker", Raw: map[string]any{}}
	b := &Entity{ID: "b", Kind: "bundle", Raw: map[string]any{}}
	tg := &Entity{ID: "t", Kind: "target", Raw: map[string]any{}}
	errs := ApplyTagSelection([]*Entity{g, w, b, tg}, []string{"test"}, TagRefsInclude)
	require.Empty(t, errs)
	assert.True(t, g.Selected)
	assert.True(t, w.Selected)
	assert.True(t, b.Selected)
	assert.True(t, tg.Selected)
}
