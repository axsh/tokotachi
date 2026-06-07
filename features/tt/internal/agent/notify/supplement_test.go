package notify

import (
	"fmt"
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/stretchr/testify/assert"
)

func TestSupplementEnvironment(t *testing.T) {
	t.Run("git available: populates git info and scope=branch", func(t *testing.T) {
		mock := newMockGitExecutor()
		mock.responses[fmt.Sprintf("%v", []string{"rev-parse", "--show-toplevel"})] = "/repo"
		mock.responses[fmt.Sprintf("%v", []string{"rev-parse", "--abbrev-ref", "HEAD"})] = "feature/test"
		mock.responses[fmt.Sprintf("%v", []string{"rev-parse", "HEAD"})] = "abc123def"
		mock.responses[fmt.Sprintf("%v", []string{"status", "--porcelain"})] = "M file.go"
		mock.responses[fmt.Sprintf("%v", []string{"rev-parse", "--verify", "refs/heads/main"})] = "exists"
		mock.responses[fmt.Sprintf("%v", []string{"merge-base", "HEAD", "main"})] = "base123"
		mock.responses[fmt.Sprintf("%v", []string{"remote", "get-url", "origin"})] = "git@github.com:axsh/tokotachi.git"

		event := &agent.IntakeEvent{
			NotifyPayload: agent.NotifyPayload{
				ChangedPaths: []string{"a.go"},
			},
		}

		warnings := SupplementEnvironment(event, mock, false)

		assert.Empty(t, warnings)
		assert.NotNil(t, event.Git)
		assert.Equal(t, "feature/test", event.Git.Branch)
		assert.Equal(t, "abc123def", event.Git.HeadCommit)
		assert.True(t, event.Git.IsDirty)
		assert.Equal(t, "main", event.Git.DefaultBranch)
		assert.Equal(t, "base123", event.Git.MergeBase)
		assert.Equal(t, "branch", event.Scope)
		assert.NotNil(t, event.BranchPackage)
		assert.Contains(t, event.BranchPackage.Key, "axsh/tokotachi")
		assert.NotEmpty(t, event.Provenance.Hostname)
		assert.NotEmpty(t, event.Provenance.Cwd)
	})

	t.Run("no git: scope=session with NO_GIT_REPOSITORY warning", func(t *testing.T) {
		mock := newMockGitExecutor()
		mock.errors[fmt.Sprintf("%v", []string{"rev-parse", "--show-toplevel"})] = fmt.Errorf("not a git repo")

		event := &agent.IntakeEvent{}

		warnings := SupplementEnvironment(event, mock, false)

		assert.Contains(t, warnings, agent.CodeNoGitRepository)
		assert.Nil(t, event.Git)
		assert.Equal(t, "session", event.Scope)
		assert.Nil(t, event.BranchPackage)
	})

	t.Run("collectGitPaths merges changed_paths with dirty paths", func(t *testing.T) {
		mock := newMockGitExecutor()
		mock.responses[fmt.Sprintf("%v", []string{"rev-parse", "--show-toplevel"})] = "/repo"
		mock.responses[fmt.Sprintf("%v", []string{"rev-parse", "--abbrev-ref", "HEAD"})] = "main"
		mock.responses[fmt.Sprintf("%v", []string{"rev-parse", "HEAD"})] = "abc"
		mock.responses[fmt.Sprintf("%v", []string{"status", "--porcelain"})] = "M file.go"
		mock.responses[fmt.Sprintf("%v", []string{"rev-parse", "--verify", "refs/heads/main"})] = "exists"
		mock.responses[fmt.Sprintf("%v", []string{"merge-base", "HEAD", "main"})] = "base"
		mock.responses[fmt.Sprintf("%v", []string{"remote", "get-url", "origin"})] = "git@github.com:axsh/tokotachi.git"
		mock.responses[fmt.Sprintf("%v", []string{"diff", "--name-only", "--cached", "HEAD"})] = "staged.go"
		mock.responses[fmt.Sprintf("%v", []string{"diff", "--name-only", "HEAD"})] = "unstaged.go"
		mock.responses[fmt.Sprintf("%v", []string{"ls-files", "--others", "--exclude-standard"})] = "untracked.go"

		event := &agent.IntakeEvent{
			NotifyPayload: agent.NotifyPayload{
				ChangedPaths: []string{"explicit.go", "staged.go"}, // staged.go is duplicate
			},
		}

		warnings := SupplementEnvironment(event, mock, true)

		assert.Empty(t, warnings)
		assert.Contains(t, event.EffectiveChangedPaths, "explicit.go")
		assert.Contains(t, event.EffectiveChangedPaths, "staged.go")
		assert.Contains(t, event.EffectiveChangedPaths, "unstaged.go")
		assert.Contains(t, event.EffectiveChangedPaths, "untracked.go")
		// No duplicates
		seen := make(map[string]bool)
		for _, p := range event.EffectiveChangedPaths {
			assert.False(t, seen[p], "duplicate path: %s", p)
			seen[p] = true
		}
	})

	t.Run("provenance populated with hostname, user, cwd", func(t *testing.T) {
		mock := newMockGitExecutor()
		mock.errors[fmt.Sprintf("%v", []string{"rev-parse", "--show-toplevel"})] = fmt.Errorf("no git")

		event := &agent.IntakeEvent{}
		SupplementEnvironment(event, mock, false)

		assert.NotEmpty(t, event.Provenance.Hostname)
		assert.NotEmpty(t, event.Provenance.User)
		assert.NotEmpty(t, event.Provenance.Cwd)
	})
}
