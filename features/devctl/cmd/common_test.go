package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBranchFeature(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantBranch  string
		wantFeature string
	}{
		{"branch only", []string{"feat-x"}, "feat-x", ""},
		{"branch and feature", []string{"feat-x", "devctl"}, "feat-x", "devctl"},
		{"branch and feature with slash", []string{"feat/add-auth", "my-feature"}, "feat/add-auth", "my-feature"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branch, feature := ParseBranchFeature(tt.args)
			assert.Equal(t, tt.wantBranch, branch)
			assert.Equal(t, tt.wantFeature, feature)
		})
	}
}

func TestHasFeature(t *testing.T) {
	tests := []struct {
		name    string
		feature string
		want    bool
	}{
		{"empty feature", "", false},
		{"with feature", "devctl", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &AppContext{Feature: tt.feature}
			assert.Equal(t, tt.want, ctx.HasFeature())
		})
	}
}

func TestInitContext_BranchOnly(t *testing.T) {
	ctx, err := InitContext([]string{"feat-x"})
	require.NoError(t, err)
	assert.Equal(t, "feat-x", ctx.Branch)
	assert.Equal(t, "", ctx.Feature)
}

func TestInitContext_BranchAndFeature(t *testing.T) {
	ctx, err := InitContext([]string{"feat-x", "devctl"})
	require.NoError(t, err)
	assert.Equal(t, "feat-x", ctx.Branch)
	assert.Equal(t, "devctl", ctx.Feature)
}

func TestInitContext_NoArgs(t *testing.T) {
	_, err := InitContext([]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "branch name is required")
}
