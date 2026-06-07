package notify

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
)

// GitExecutor abstracts git command execution for testability.
type GitExecutor interface {
	Run(args ...string) (string, error)
}

// RealGitExecutor executes real git commands.
type RealGitExecutor struct{}

// Run executes a git command and returns trimmed stdout.
func (g *RealGitExecutor) Run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// DeriveBranchPackage computes the branch_package identifier.
// Format: "{repo_id}:{branch_name}:{merge_base}"
func DeriveBranchPackage(git *agent.GitInfo, executor GitExecutor) string {
	if git == nil || git.Branch == "" || git.Branch == "HEAD" {
		return ""
	}

	repoID := deriveRepoID(executor)
	mergeBase := git.MergeBase

	return fmt.Sprintf("%s:%s:%s", repoID, git.Branch, mergeBase)
}

// DeriveScope determines scope based on git availability.
// Returns "branch" if in a git repo with a named branch.
// Returns "session" if no git, or detached HEAD.
func DeriveScope(git *agent.GitInfo) string {
	if git == nil {
		return "session"
	}
	if git.Branch == "" || git.Branch == "HEAD" {
		return "session"
	}
	return "branch"
}

// deriveRepoID extracts owner/repo from git remote origin URL.
func deriveRepoID(executor GitExecutor) string {
	url, err := executor.Run("remote", "get-url", "origin")
	if err != nil || url == "" {
		return "unknown"
	}

	// Handle SSH format: git@github.com:owner/repo.git
	if strings.Contains(url, ":") && strings.HasPrefix(url, "git@") {
		parts := strings.SplitN(url, ":", 2)
		if len(parts) == 2 {
			return strings.TrimSuffix(parts[1], ".git")
		}
	}

	// Handle HTTPS format: https://github.com/owner/repo.git
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "/" + parts[len(parts)-1]
	}

	return "unknown"
}
