package git_test

import (
	"testing"

	"github.com/axsh/tokotachi/features/release-note/internal/git"
)

func TestExtractBranchFromMessage(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{
			name:    "merge branch into main",
			message: "Merge branch 'feat-xxx' into main",
			want:    "feat-xxx",
		},
		{
			name:    "merge branch into develop",
			message: "Merge branch 'fix-bug-123' into develop",
			want:    "fix-bug-123",
		},
		{
			name:    "merge pull request",
			message: "Merge pull request #42 from user/feat-yyy",
			want:    "feat-yyy",
		},
		{
			name:    "merge pull request with org",
			message: "Merge pull request #100 from axsh/fix-module-versioning",
			want:    "fix-module-versioning",
		},
		{
			name:    "not a merge commit",
			message: "Add new feature",
			want:    "",
		},
		{
			name:    "empty message",
			message: "",
			want:    "",
		},
		{
			name:    "merge branch without into",
			message: "Merge branch 'hotfix'",
			want:    "hotfix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := git.ExtractBranchFromMessage(tt.message)
			if got != tt.want {
				t.Errorf("ExtractBranchFromMessage(%q) = %q, want %q", tt.message, got, tt.want)
			}
		})
	}
}

func TestDeduplicateBranches(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  int
	}{
		{
			name:  "no duplicates",
			input: []string{"feat-a", "feat-b", "feat-c"},
			want:  3,
		},
		{
			name:  "with duplicates",
			input: []string{"feat-a", "feat-b", "feat-a", "feat-c", "feat-b"},
			want:  3,
		},
		{
			name:  "all same",
			input: []string{"feat-a", "feat-a", "feat-a"},
			want:  1,
		},
		{
			name:  "empty",
			input: []string{},
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := git.DeduplicateBranches(tt.input)
			if len(got) != tt.want {
				t.Errorf("DeduplicateBranches() returned %d items, want %d", len(got), tt.want)
			}
		})
	}
}
