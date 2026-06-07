package notify

import (
	"crypto/sha256"
	"fmt"
	"os/exec"
	"regexp"
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

var (
	slugUnsafeChars    = regexp.MustCompile(`[^A-Za-z0-9._-]`)
	slugConsecutiveDash = regexp.MustCompile(`-{2,}`)
)

// Slugify converts a branch name to a path-safe slug.
// Rules:
//   - Replace characters outside [A-Za-z0-9._-] with "-"
//   - Collapse consecutive "-" to one
//   - Trim leading/trailing "-"
//   - Max length 64 chars; if exceeded, use first 56 + "-" + 7-char hash
func Slugify(name string) string {
	result := slugUnsafeChars.ReplaceAllString(name, "-")
	result = slugConsecutiveDash.ReplaceAllString(result, "-")
	result = strings.Trim(result, "-")

	if len(result) > 64 {
		h := sha256.Sum256([]byte(name))
		hashSuffix := fmt.Sprintf("%x", h[:4])[:7]
		result = result[:56] + "-" + hashSuffix
	}

	return result
}

// DeriveBranchPackage computes the structured branch package identifier.
// Returns nil if not in a named branch.
func DeriveBranchPackage(git *agent.GitInfo, executor GitExecutor) *agent.BranchPackageInfo {
	if git == nil || git.Branch == "" || git.Branch == "HEAD" {
		return nil
	}

	repoID := deriveRepoID(executor)
	mergeBase := git.MergeBase
	key := fmt.Sprintf("%s:%s:%s", repoID, git.Branch, mergeBase)

	slug := Slugify(git.Branch)
	short := mergeBase
	if len(short) > 8 {
		short = short[:8]
	}
	id := fmt.Sprintf("BR-%s-%s", slug, short)

	return &agent.BranchPackageInfo{
		Key:       key,
		ID:        id,
		Branch:    git.Branch,
		MergeBase: mergeBase,
	}
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
