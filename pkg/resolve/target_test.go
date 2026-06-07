package resolve

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveTarget_ExactMatch(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"antigravity", "antigravity"},
		{"cursor", "cursor"},
		{"claude-code", "claude-code"},
		{"codex", "codex"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ResolveTarget(tt.input, true)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveTarget_AliasMatch(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"ag -> antigravity", "ag", "antigravity"},
		{"agy -> antigravity", "agy", "antigravity"},
		{"claude -> claude-code", "claude", "claude-code"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveTarget(tt.input, true)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveTarget_PrefixMatch(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"anti -> antigravity", "anti", "antigravity"},
		{"cur -> cursor", "cur", "cursor"},
		{"cl -> claude-code", "cl", "claude-code"},
		{"co -> codex", "co", "codex"},
		{"al -> all", "al", "all"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveTarget(tt.input, true)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveTarget_AmbiguousError(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"a matches all and antigravity", "a"},
		{"c matches claude-code, codex, cursor", "c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveTarget(tt.input, true)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "ambiguous")
		})
	}
}

func TestResolveTarget_AllowAll(t *testing.T) {
	got, err := ResolveTarget("all", true)
	require.NoError(t, err)
	assert.Equal(t, "all", got)
}

func TestResolveTarget_DisallowAll(t *testing.T) {
	_, err := ResolveTarget("all", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestResolveTarget_UnknownInput(t *testing.T) {
	_, err := ResolveTarget("xyz", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown")
}

func TestResolveTarget_EmptyInput(t *testing.T) {
	_, err := ResolveTarget("", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestResolveTargets_All(t *testing.T) {
	got, err := ResolveTargets("all")
	require.NoError(t, err)
	assert.Equal(t, []string{"antigravity", "claude-code", "codex", "cursor"}, got)
}

func TestResolveTargets_Single(t *testing.T) {
	got, err := ResolveTargets("anti")
	require.NoError(t, err)
	assert.Equal(t, []string{"antigravity"}, got)
}

func TestMetaDir(t *testing.T) {
	tests := []struct {
		target string
		want   string
	}{
		{"antigravity", ".agent/.meta/"},
		{"cursor", ".cursor/.meta/"},
		{"claude-code", ".claude/.meta/"},
		{"codex", ".agents/.meta/"},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			assert.Equal(t, tt.want, MetaDir(tt.target))
		})
	}
}

func TestMetaDir_Unknown(t *testing.T) {
	assert.Equal(t, "", MetaDir("unknown"))
}
