package notify

import (
	"os"
	"os/user"
	"strings"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
)

// SupplementEnvironment fills in Git info, provenance, and effective paths.
// If not in a git repository, sets scope to "session" and adds NO_GIT_REPOSITORY warning.
func SupplementEnvironment(event *agent.IntakeEvent, executor GitExecutor, collectGitPaths bool) []string {
	var warnings []string

	// Supplement provenance
	event.Provenance = collectProvenance(event.Provenance)

	// Attempt to collect git info
	gitInfo, gitWarnings := collectGitInfo(executor)
	warnings = append(warnings, gitWarnings...)
	event.Git = gitInfo

	// Derive scope and branch package
	event.Scope = DeriveScope(gitInfo)
	event.BranchPackage = DeriveBranchPackage(gitInfo, executor)

	// Compute effective changed paths
	if collectGitPaths && gitInfo != nil {
		dirtyPaths := collectDirtyPaths(executor)
		event.EffectiveChangedPaths = mergeUniquePaths(event.ChangedPaths, dirtyPaths)
	} else {
		event.EffectiveChangedPaths = event.ChangedPaths
	}

	return warnings
}

// collectGitInfo gathers git repository state.
func collectGitInfo(executor GitExecutor) (*agent.GitInfo, []string) {
	var warnings []string

	// Check if we're in a git repo
	_, err := executor.Run("rev-parse", "--show-toplevel")
	if err != nil {
		warnings = append(warnings, agent.CodeNoGitRepository)
		return nil, warnings
	}

	info := &agent.GitInfo{}

	// Branch name
	branch, err := executor.Run("rev-parse", "--abbrev-ref", "HEAD")
	if err == nil {
		info.Branch = branch
	}

	// Head commit
	head, err := executor.Run("rev-parse", "HEAD")
	if err == nil {
		info.HeadCommit = head
	}

	// Is dirty
	status, _ := executor.Run("status", "--porcelain")
	info.IsDirty = status != ""

	// Default branch (try main, then master)
	for _, candidate := range []string{"main", "master"} {
		if _, err := executor.Run("rev-parse", "--verify", "refs/heads/"+candidate); err == nil {
			info.DefaultBranch = candidate
			break
		}
	}

	// Merge base
	if info.DefaultBranch != "" && info.Branch != "" && info.Branch != "HEAD" {
		mergeBase, err := executor.Run("merge-base", "HEAD", info.DefaultBranch)
		if err == nil {
			info.MergeBase = mergeBase
		}
	}

	return info, warnings
}

// collectDirtyPaths gets all modified/untracked files from git.
func collectDirtyPaths(executor GitExecutor) []string {
	var paths []string

	// Staged changes
	if staged, err := executor.Run("diff", "--name-only", "--cached", "HEAD"); err == nil && staged != "" {
		paths = append(paths, strings.Split(staged, "\n")...)
	}

	// Unstaged changes
	if unstaged, err := executor.Run("diff", "--name-only", "HEAD"); err == nil && unstaged != "" {
		paths = append(paths, strings.Split(unstaged, "\n")...)
	}

	// Untracked files
	if untracked, err := executor.Run("ls-files", "--others", "--exclude-standard"); err == nil && untracked != "" {
		paths = append(paths, strings.Split(untracked, "\n")...)
	}

	return paths
}

// collectProvenance gathers environment info, preserving any existing values.
func collectProvenance(existing agent.Provenance) agent.Provenance {
	p := existing
	if p.Hostname == "" {
		p.Hostname, _ = os.Hostname()
	}
	if p.User == "" {
		if u, err := user.Current(); err == nil {
			p.User = u.Username
		}
	}
	if p.Cwd == "" {
		p.Cwd, _ = os.Getwd()
	}
	return p
}

// mergeUniquePaths merges two path slices, deduplicating entries.
func mergeUniquePaths(a, b []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, p := range a {
		p = strings.TrimSpace(p)
		if p != "" && !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}
	for _, p := range b {
		p = strings.TrimSpace(p)
		if p != "" && !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}
	return result
}
