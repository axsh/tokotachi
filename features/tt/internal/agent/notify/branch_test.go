package notify

import (
	"fmt"
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/stretchr/testify/assert"
)

// mockGitExecutor returns predefined results for git commands.
type mockGitExecutor struct {
	responses map[string]string
	errors    map[string]error
}

func newMockGitExecutor() *mockGitExecutor {
	return &mockGitExecutor{
		responses: make(map[string]string),
		errors:    make(map[string]error),
	}
}

func (m *mockGitExecutor) Run(args ...string) (string, error) {
	key := fmt.Sprintf("%v", args)
	if err, ok := m.errors[key]; ok {
		return "", err
	}
	if resp, ok := m.responses[key]; ok {
		return resp, nil
	}
	return "", fmt.Errorf("unexpected git command: %v", args)
}

func TestDeriveScope(t *testing.T) {
	tests := []struct {
		name     string
		git      *agent.GitInfo
		expected string
	}{
		{
			name:     "nil git info returns session",
			git:      nil,
			expected: "session",
		},
		{
			name:     "empty branch returns session",
			git:      &agent.GitInfo{Branch: ""},
			expected: "session",
		},
		{
			name:     "detached HEAD returns session",
			git:      &agent.GitInfo{Branch: "HEAD"},
			expected: "session",
		},
		{
			name:     "named branch returns branch",
			git:      &agent.GitInfo{Branch: "feature/my-feature"},
			expected: "branch",
		},
		{
			name:     "main branch returns branch",
			git:      &agent.GitInfo{Branch: "main"},
			expected: "branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeriveScope(tt.git)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeriveBranchPackage(t *testing.T) {
	tests := []struct {
		name      string
		git       *agent.GitInfo
		remoteURL string
		expected  string
	}{
		{
			name: "normal git with SSH remote",
			git: &agent.GitInfo{
				Branch:    "feature/auth",
				MergeBase: "abc123",
			},
			remoteURL: "git@github.com:axsh/tokotachi.git",
			expected:  "axsh/tokotachi:feature/auth:abc123",
		},
		{
			name: "normal git with HTTPS remote",
			git: &agent.GitInfo{
				Branch:    "main",
				MergeBase: "def456",
			},
			remoteURL: "https://github.com/axsh/tokotachi.git",
			expected:  "axsh/tokotachi:main:def456",
		},
		{
			name: "merge_base empty",
			git: &agent.GitInfo{
				Branch:    "develop",
				MergeBase: "",
			},
			remoteURL: "git@github.com:axsh/tokotachi.git",
			expected:  "axsh/tokotachi:develop:",
		},
		{
			name:      "nil git returns empty",
			git:       nil,
			remoteURL: "",
			expected:  "",
		},
		{
			name: "detached HEAD returns empty",
			git: &agent.GitInfo{
				Branch: "HEAD",
			},
			remoteURL: "",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGitExecutor()
			if tt.remoteURL != "" {
				mock.responses[fmt.Sprintf("%v", []string{"remote", "get-url", "origin"})] = tt.remoteURL
			}
			result := DeriveBranchPackage(tt.git, mock)
			assert.Equal(t, tt.expected, result)
		})
	}
}
