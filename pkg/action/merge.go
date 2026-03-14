package action

import (
	"fmt"

	"github.com/axsh/tokotachi/internal/cmdexec"
	"github.com/axsh/tokotachi/pkg/state"
)

// MergeStrategy represents the git merge strategy option.
type MergeStrategy string

const (
	// MergeStrategyFFOnly only allows fast-forward merges (default).
	MergeStrategyFFOnly MergeStrategy = "ff-only"
	// MergeStrategyNoFF always creates a merge commit.
	MergeStrategyNoFF MergeStrategy = "no-ff"
	// MergeStrategyFF uses git default behavior (ff if possible, merge commit otherwise).
	MergeStrategyFF MergeStrategy = "ff"
)

// MergeOptions holds parameters for the merge action.
type MergeOptions struct {
	Branch   string        // branch to merge
	RepoRoot string        // repo root (where BaseBranch is checked out)
	Strategy MergeStrategy // merge strategy
}

// MergeResult holds the result of a merge operation.
type MergeResult struct {
	BaseBranch string        // the branch merged into
	Strategy   MergeStrategy // strategy used
	Success    bool
}

// Merge executes a local git merge of the specified branch into its base branch.
// The base branch is resolved from the StateFile; falls back to "main" if not recorded.
// The merge is executed at the RepoRoot directory where the base branch is checked out.
func (r *Runner) Merge(opts MergeOptions) (MergeResult, error) {
	result := MergeResult{Strategy: opts.Strategy}

	// Step 1: Resolve BaseBranch from StateFile
	baseBranch := resolveBaseBranch(opts.RepoRoot, opts.Branch)
	result.BaseBranch = baseBranch

	// Step 2: Check for uncommitted changes in worktree
	worktreePath := fmt.Sprintf("%s/work/%s", opts.RepoRoot, opts.Branch)
	if err := r.checkClean(worktreePath); err != nil {
		return result, fmt.Errorf("worktree has uncommitted changes: %w", err)
	}

	// Step 3: Check for uncommitted changes in root repository
	if err := r.checkClean(opts.RepoRoot); err != nil {
		return result, fmt.Errorf("root repository has uncommitted changes: %w", err)
	}

	// Step 4: Build git merge command
	gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")
	args := buildMergeArgs(opts.Strategy, opts.Branch)

	// Step 5: Execute merge at RepoRoot
	r.Logger.Info("Merging %s into %s (strategy: %s)...", opts.Branch, baseBranch, opts.Strategy)
	mergeOpts := cmdexec.RunOption{Dir: opts.RepoRoot}
	if _, err := r.CmdRunner.RunWithOpts(mergeOpts, gitCmd, args...); err != nil {
		return result, fmt.Errorf("git merge failed: %w", err)
	}

	result.Success = true
	r.Logger.Info("Merge completed: %s → %s", opts.Branch, baseBranch)
	return result, nil
}

// resolveBaseBranch reads BaseBranch from the state file.
// Returns "main" as fallback if the state file doesn't exist or BaseBranch is empty.
func resolveBaseBranch(repoRoot, branch string) string {
	statePath := state.StatePath(repoRoot, branch)
	sf, err := state.Load(statePath)
	if err != nil {
		return "main"
	}
	if sf.BaseBranch == "" {
		return "main"
	}
	return sf.BaseBranch
}

// buildMergeArgs constructs the git merge command arguments based on strategy.
func buildMergeArgs(strategy MergeStrategy, branch string) []string {
	switch strategy {
	case MergeStrategyFFOnly:
		return []string{"merge", "--ff-only", branch}
	case MergeStrategyNoFF:
		return []string{"merge", "--no-ff", branch}
	case MergeStrategyFF:
		return []string{"merge", branch}
	default:
		// Default to ff-only
		return []string{"merge", "--ff-only", branch}
	}
}

// checkClean runs "git status --porcelain" in the given directory.
// Returns an error if there are uncommitted changes.
// In dry-run mode, this check is skipped (always clean).
func (r *Runner) checkClean(dir string) error {
	if r.DryRun {
		return nil
	}
	gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")
	opts := cmdexec.RunOption{Dir: dir, QuietCmd: true}
	output, err := r.CmdRunner.RunWithOpts(opts, gitCmd, "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("git status failed: %w", err)
	}
	if output != "" {
		return fmt.Errorf("uncommitted changes found in %s", dir)
	}
	return nil
}
