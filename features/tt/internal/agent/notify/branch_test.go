package notify

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestSlugify(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple branch name",
			input:    "main",
			expected: "main",
		},
		{
			name:     "slash to dash",
			input:    "feature/auth",
			expected: "feature-auth",
		},
		{
			name:     "multiple slashes",
			input:    "feature/foo/bar",
			expected: "feature-foo-bar",
		},
		{
			name:     "backslash to dash",
			input:    "feature\\auth",
			expected: "feature-auth",
		},
		{
			name:     "colon to dash",
			input:    "fix:urgent",
			expected: "fix-urgent",
		},
		{
			name:     "spaces to dash",
			input:    "fix something here",
			expected: "fix-something-here",
		},
		{
			name:     "consecutive dashes collapsed",
			input:    "fix--double--dash",
			expected: "fix-double-dash",
		},
		{
			name:     "leading trailing dashes trimmed",
			input:    "/feature/auth/",
			expected: "feature-auth",
		},
		{
			name:     "dots and underscores preserved",
			input:    "release_v1.0",
			expected: "release_v1.0",
		},
		{
			name:     "mixed unsafe chars",
			input:    "fix/bug#123@work",
			expected: "fix-bug-123-work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Slugify(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSlugify_LongName(t *testing.T) {
	longName := strings.Repeat("a", 70)
	result := Slugify(longName)
	assert.LessOrEqual(t, len(result), 64, "slug should be at most 64 chars")
	assert.True(t, len(result) >= 56, "slug should be at least 56 chars")
}

func TestSlugify_PathSafety(t *testing.T) {
	unsafePattern := regexp.MustCompile(`[/:\\]`)
	testNames := []string{
		"feature/auth",
		"fix:urgent",
		"path\\with\\backslash",
		"axsh/tokotachi:main:abc123",
	}
	for _, name := range testNames {
		slug := Slugify(name)
		assert.False(t, unsafePattern.MatchString(slug),
			"slug %q should not contain path-unsafe chars", slug)
	}
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
		expectNil bool
		checkFunc func(t *testing.T, bp *agent.BranchPackageInfo)
	}{
		{
			name: "normal git with SSH remote",
			git: &agent.GitInfo{
				Branch:    "feature/auth",
				MergeBase: "abc12345deadbeef",
			},
			remoteURL: "git@github.com:axsh/tokotachi.git",
			checkFunc: func(t *testing.T, bp *agent.BranchPackageInfo) {
				assert.Equal(t, "axsh/tokotachi:feature/auth:abc12345deadbeef", bp.Key)
				assert.Equal(t, "BR-feature-auth-abc12345", bp.ID)
				assert.Equal(t, "feature/auth", bp.Branch)
				assert.Equal(t, "abc12345deadbeef", bp.MergeBase)
			},
		},
		{
			name: "normal git with HTTPS remote",
			git: &agent.GitInfo{
				Branch:    "main",
				MergeBase: "def456789abcdef0",
			},
			remoteURL: "https://github.com/axsh/tokotachi.git",
			checkFunc: func(t *testing.T, bp *agent.BranchPackageInfo) {
				assert.Equal(t, "axsh/tokotachi:main:def456789abcdef0", bp.Key)
				assert.Equal(t, "BR-main-def45678", bp.ID)
				assert.Equal(t, "main", bp.Branch)
			},
		},
		{
			name: "merge_base empty",
			git: &agent.GitInfo{
				Branch:    "develop",
				MergeBase: "",
			},
			remoteURL: "git@github.com:axsh/tokotachi.git",
			checkFunc: func(t *testing.T, bp *agent.BranchPackageInfo) {
				assert.Equal(t, "BR-develop-", bp.ID)
			},
		},
		{
			name:      "nil git returns nil",
			git:       nil,
			remoteURL: "",
			expectNil: true,
		},
		{
			name: "detached HEAD returns nil",
			git: &agent.GitInfo{
				Branch: "HEAD",
			},
			remoteURL: "",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockGitExecutor()
			if tt.remoteURL != "" {
				mock.responses[fmt.Sprintf("%v", []string{"remote", "get-url", "origin"})] = tt.remoteURL
			}
			result := DeriveBranchPackage(tt.git, mock)
			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				tt.checkFunc(t, result)
			}
		})
	}
}

func TestDeriveBranchPackage_IDPathSafety(t *testing.T) {
	unsafePattern := regexp.MustCompile(`[/:\\]`)

	mock := newMockGitExecutor()
	mock.responses[fmt.Sprintf("%v", []string{"remote", "get-url", "origin"})] = "git@github.com:axsh/tokotachi.git"

	branches := []string{
		"feature/foo/bar",
		"fix:urgent",
		"path\\test",
		"fix-memory-compiling",
	}

	for _, branch := range branches {
		t.Run(branch, func(t *testing.T) {
			git := &agent.GitInfo{Branch: branch, MergeBase: "abc12345deadbeef"}
			result := DeriveBranchPackage(git, mock)
			require.NotNil(t, result)
			assert.False(t, unsafePattern.MatchString(result.ID),
				"ID %q should not contain path-unsafe chars", result.ID)
		})
	}
}

func TestDeriveBranchPackage_SameBranchSameID(t *testing.T) {
	mock := newMockGitExecutor()
	mock.responses[fmt.Sprintf("%v", []string{"remote", "get-url", "origin"})] = "git@github.com:axsh/tokotachi.git"

	git := &agent.GitInfo{Branch: "fix-memory-compiling", MergeBase: "4a67ef5a6457decad62348376de6a3547004fdb3"}

	result1 := DeriveBranchPackage(git, mock)
	result2 := DeriveBranchPackage(git, mock)

	require.NotNil(t, result1)
	require.NotNil(t, result2)
	assert.Equal(t, result1.ID, result2.ID, "same branch+merge_base should produce same ID")
}
