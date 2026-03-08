package listing_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/axsh/tokotachi/features/devctl/internal/listing"
	"github.com/axsh/tokotachi/features/devctl/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWorktreeOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []listing.WorktreeEntry
	}{
		{
			name:     "empty output",
			input:    "",
			expected: nil,
		},
		{
			name: "normal output with three worktrees",
			input: "worktree /home/user/repo\n" +
				"HEAD abc1234\n" +
				"branch refs/heads/main\n" +
				"bare\n" +
				"\n" +
				"worktree /home/user/repo/work/feat-a\n" +
				"HEAD def5678\n" +
				"branch refs/heads/feat-a\n" +
				"\n" +
				"worktree /home/user/repo/work/feat-b\n" +
				"HEAD ghi9012\n" +
				"branch refs/heads/feat-b\n" +
				"\n",
			expected: []listing.WorktreeEntry{
				{Path: "/home/user/repo", Branch: "main", Bare: true},
				{Path: "/home/user/repo/work/feat-a", Branch: "feat-a", Bare: false},
				{Path: "/home/user/repo/work/feat-b", Branch: "feat-b", Bare: false},
			},
		},
		{
			name: "bare worktree only",
			input: "worktree /home/user/repo\n" +
				"HEAD abc1234\n" +
				"branch refs/heads/main\n" +
				"bare\n" +
				"\n",
			expected: []listing.WorktreeEntry{
				{Path: "/home/user/repo", Branch: "main", Bare: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := listing.ParseWorktreeOutput(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCollectBranches(t *testing.T) {
	tests := []struct {
		name     string
		entries  []listing.WorktreeEntry
		states   map[string]state.StateFile
		expected []listing.BranchInfo
	}{
		{
			name: "worktree with state",
			entries: []listing.WorktreeEntry{
				{Path: "/repo/work/feat-a", Branch: "feat-a", Bare: false},
			},
			states: map[string]state.StateFile{
				"feat-a": {
					Branch: "feat-a",
					Features: map[string]state.FeatureState{
						"devctl": {Status: state.StatusActive},
					},
				},
			},
			expected: []listing.BranchInfo{
				{
					Branch:   "feat-a",
					Path:     "/repo/work/feat-a",
					Features: []listing.FeatureInfo{{Name: "devctl", Status: "active"}},
				},
			},
		},
		{
			name: "worktree without state",
			entries: []listing.WorktreeEntry{
				{Path: "/repo/work/feat-b", Branch: "feat-b", Bare: false},
			},
			states:   map[string]state.StateFile{},
			expected: []listing.BranchInfo{{Branch: "feat-b", Path: "/repo/work/feat-b", Features: []listing.FeatureInfo{}}},
		},
		{
			name: "bare worktree",
			entries: []listing.WorktreeEntry{
				{Path: "/repo", Branch: "main", Bare: true},
			},
			states: map[string]state.StateFile{},
			expected: []listing.BranchInfo{
				{Branch: "main", Path: "/repo", Features: []listing.FeatureInfo{}, MainWorktree: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := listing.CollectBranches(tt.entries, tt.states)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestFormatTable(t *testing.T) {
	prTime := time.Date(2026, 3, 8, 10, 30, 0, 0, time.UTC)
	branches := []listing.BranchInfo{
		{
			Branch:   "feat-a",
			Path:     "/repo/work/feat-a",
			Features: []listing.FeatureInfo{{Name: "devctl", Status: "active"}},
			CodeStatus: &state.CodeStatus{
				Status:      state.CodeStatusPR,
				PRCreatedAt: &prTime,
			},
		},
		{
			Branch:   "feat-b",
			Path:     "/repo/work/feat-b",
			Features: []listing.FeatureInfo{},
			CodeStatus: &state.CodeStatus{
				Status: state.CodeStatusLocal,
			},
		},
		{Branch: "main", Path: "/repo", Features: []listing.FeatureInfo{}, MainWorktree: true},
	}

	t.Run("without path", func(t *testing.T) {
		var buf bytes.Buffer
		listing.FormatTable(&buf, branches, false)
		out := buf.String()
		// Header verification
		assert.Contains(t, out, "BRANCH")
		assert.Contains(t, out, "FEATURE")
		assert.NotContains(t, out, "FEATURES")
		assert.Contains(t, out, "CONTAINER")
		assert.NotContains(t, out, "STATE")
		assert.Contains(t, out, "CODE")
		assert.NotContains(t, out, "PATH")
		// Body verification: feature name and container status are separated
		assert.Contains(t, out, "devctl")
		assert.Contains(t, out, "active")
		assert.NotContains(t, out, "devctl[active]")
		assert.Contains(t, out, "(main worktree)")
		assert.Contains(t, out, "(no state)")
		// CODE column content
		assert.Contains(t, out, "PR(")
		assert.Contains(t, out, "(local)")
	})

	t.Run("with path", func(t *testing.T) {
		var buf bytes.Buffer
		listing.FormatTable(&buf, branches, true)
		out := buf.String()
		// Header verification
		assert.Contains(t, out, "FEATURE")
		assert.NotContains(t, out, "FEATURES")
		assert.Contains(t, out, "CONTAINER")
		assert.NotContains(t, out, "STATE")
		assert.Contains(t, out, "CODE")
		assert.Contains(t, out, "PATH")
		// Body verification
		assert.Contains(t, out, "/repo/work/feat-a")
		assert.Contains(t, out, "/repo")
	})
}

func TestFormatCodeColumn(t *testing.T) {
	now := time.Date(2026, 3, 9, 2, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		bi       listing.BranchInfo
		expected string
	}{
		{
			name:     "main worktree",
			bi:       listing.BranchInfo{MainWorktree: true},
			expected: "-",
		},
		{
			name:     "nil code status",
			bi:       listing.BranchInfo{},
			expected: "(unknown)",
		},
		{
			name: "local",
			bi: listing.BranchInfo{
				CodeStatus: &state.CodeStatus{Status: state.CodeStatusLocal},
			},
			expected: "(local)",
		},
		{
			name: "hosted",
			bi: listing.BranchInfo{
				CodeStatus: &state.CodeStatus{Status: state.CodeStatusHosted},
			},
			expected: "hosted",
		},
		{
			name: "deleted",
			bi: listing.BranchInfo{
				CodeStatus: &state.CodeStatus{Status: state.CodeStatusDeleted},
			},
			expected: "deleted",
		},
		{
			name: "PR without created_at",
			bi: listing.BranchInfo{
				CodeStatus: &state.CodeStatus{Status: state.CodeStatusPR},
			},
			expected: "PR",
		},
		{
			name: "PR 3 minutes ago",
			bi: listing.BranchInfo{
				CodeStatus: &state.CodeStatus{
					Status:      state.CodeStatusPR,
					PRCreatedAt: timePtr(now.Add(-3 * time.Minute)),
				},
			},
			expected: "PR(3m ago)",
		},
		{
			name: "PR 2 hours ago",
			bi: listing.BranchInfo{
				CodeStatus: &state.CodeStatus{
					Status:      state.CodeStatusPR,
					PRCreatedAt: timePtr(now.Add(-2 * time.Hour)),
				},
			},
			expected: "PR(2h ago)",
		},
		{
			name: "PR 5 days ago",
			bi: listing.BranchInfo{
				CodeStatus: &state.CodeStatus{
					Status:      state.CodeStatusPR,
					PRCreatedAt: timePtr(now.Add(-5 * 24 * time.Hour)),
				},
			},
			expected: "PR(5d ago)",
		},
		{
			name: "PR 31 days ago",
			bi: listing.BranchInfo{
				CodeStatus: &state.CodeStatus{
					Status:      state.CodeStatusPR,
					PRCreatedAt: timePtr(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
				},
			},
			expected: "PR(01/15)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := listing.FormatCodeColumn(tt.bi, now)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestFormatJSON(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		branches := []listing.BranchInfo{
			{Branch: "feat-a", Path: "/repo/work/feat-a", Features: []listing.FeatureInfo{{Name: "devctl", Status: "active"}}},
		}
		var buf bytes.Buffer
		err := listing.FormatJSON(&buf, branches)
		require.NoError(t, err)

		var result []listing.BranchInfo
		require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
		assert.Len(t, result, 1)
		assert.Equal(t, "feat-a", result[0].Branch)
	})

	t.Run("empty", func(t *testing.T) {
		var buf bytes.Buffer
		err := listing.FormatJSON(&buf, []listing.BranchInfo{})
		require.NoError(t, err)

		var result []listing.BranchInfo
		require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
		assert.Len(t, result, 0)
	})
}
