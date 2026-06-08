package worktree

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/axsh/tokotachi/internal/cmdexec"
	pkglog "github.com/axsh/tokotachi/pkg/log"
)

const (
	// removeRetryDelay is the wait time before retrying git worktree remove.
	// This primarily addresses Windows file-lock issues where editors
	// may still hold references to worktree files.
	removeRetryDelay = 500 * time.Millisecond

	// removeMaxRetries is the maximum number of retries for git worktree remove.
	removeMaxRetries = 1
)

// WorktreeInfo represents a worktree entry.
type WorktreeInfo struct {
	Feature string
	Branch  string
	Path    string
}

// Manager handles git worktree operations.
type Manager struct {
	CmdRunner *cmdexec.Runner
	RepoRoot  string
}

// Path returns the worktree directory path: work/<branch>
func (m *Manager) Path(branch string) string {
	return filepath.Join(m.RepoRoot, "work", branch)
}

// Exists checks if the worktree directory exists and is a valid git worktree.
// A valid worktree has a .git file (not directory) pointing to the main repo's worktree metadata.
func (m *Manager) Exists(branch string) bool {
	wtPath := m.Path(branch)
	info, err := os.Stat(wtPath)
	if err != nil || !info.IsDir() {
		return false
	}
	// Valid worktrees have a .git file inside
	gitPath := filepath.Join(wtPath, ".git")
	_, err = os.Stat(gitPath)
	return err == nil
}

// Create creates a new git worktree.
// If the branch already exists, uses it without -b flag.
// Uses --force to handle branches already checked out in other worktrees.
func (m *Manager) Create(branch string) error {
	wtPath := m.Path(branch)

	// Clean up ghost directory: directory exists but is not a valid worktree
	if info, err := os.Stat(wtPath); err == nil && info.IsDir() {
		gitPath := filepath.Join(wtPath, ".git")
		if _, gitErr := os.Stat(gitPath); os.IsNotExist(gitErr) {
			// Ghost directory — remove before creating new worktree
			os.RemoveAll(wtPath)
		}
	}
	gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")

	// Prune stale worktree metadata before creating (R5).
	// This handles cases where a previous close failed and left
	// .git/worktrees/<name>/ metadata behind without the actual directory.
	m.Prune()

	// Check if branch already exists locally
	_, err := m.CmdRunner.RunWithOpts(cmdexec.CheckOpt(), gitCmd, "rev-parse", "--verify", branch)
	branchExists := err == nil

	var args []string
	if branchExists {
		// Local branch exists: attach worktree to existing branch (force in case already checked out)
		args = []string{"worktree", "add", "--force", wtPath, branch}
	} else if m.RemoteBranchExists(branch) {
		// Remote branch exists: fetch and create worktree from remote tracking branch
		if fetchErr := m.FetchBranch(branch); fetchErr != nil {
			// Fetch failed: fallback to creating new local branch
			args = []string{"worktree", "add", "-b", branch, wtPath}
		} else {
			// Fetch succeeded: create worktree from fetched branch
			args = []string{"worktree", "add", wtPath, branch}
		}
	} else {
		// No local or remote branch: create new branch
		args = []string{"worktree", "add", "-b", branch, wtPath}
	}

	if _, err := m.CmdRunner.Run(gitCmd, args...); err != nil {
		return fmt.Errorf("git worktree add failed: %w", err)
	}
	return nil
}

// Remove removes a git worktree.
// Before removal, it deinitializes any git submodules to prevent
// "working trees containing submodules cannot be moved or removed" errors.
// On failure, it retries after a short delay to handle Windows file-lock issues.
func (m *Manager) Remove(branch string, force bool) error {
	wtPath := m.Path(branch)
	gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")

	// Step 1: Deinit submodules if present (R1)
	m.deinitSubmodules(wtPath)

	// Step 2: Try git worktree remove
	args := []string{"worktree", "remove", wtPath}
	if force {
		args = []string{"worktree", "remove", "-f", wtPath}
	}

	_, err := m.CmdRunner.RunWithOpts(cmdexec.ToleratedOpt(), gitCmd, args...)

	// Step 3: Retry on failure (R3)
	if err != nil {
		for range removeMaxRetries {
			time.Sleep(removeRetryDelay)
			_, err = m.CmdRunner.RunWithOpts(cmdexec.ToleratedOpt(), gitCmd, args...)
			if err == nil {
				break
			}
		}
	}

	if err != nil {
		return fmt.Errorf("git worktree remove failed: %w", err)
	}

	// Ensure directory is fully removed (git may leave empty directory)
	if _, err := os.Stat(wtPath); err == nil {
		os.RemoveAll(wtPath)
	}
	return nil
}

// deinitSubmodules deinitializes git submodules in the worktree directory.
// This is a prerequisite for git worktree remove when submodules are present.
// Failures are tolerated (logged at WARN level) since the worktree may
// not have initialized submodules.
func (m *Manager) deinitSubmodules(wtPath string) {
	gitmodulesPath := filepath.Join(wtPath, ".gitmodules")
	if _, err := os.Stat(gitmodulesPath); os.IsNotExist(err) {
		return
	}
	gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")
	opts := cmdexec.RunOption{
		Dir:          wtPath,
		FailLevelSet: true,
		FailLevel:    pkglog.LevelWarn,
		FailLabel:    "SKIP",
		QuietCmd:     false,
	}
	m.CmdRunner.RunWithOpts(opts, gitCmd, "submodule", "deinit", "--all", "-f")
}

// DeleteBranch deletes the local branch.
// Uses -d for merged branches, -D if force is true.
func (m *Manager) DeleteBranch(branch string, force bool) error {
	gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")

	flag := "-d"
	if force {
		flag = "-D"
	}

	if _, err := m.CmdRunner.RunWithOpts(cmdexec.ToleratedOpt(), gitCmd, "branch", flag, branch); err != nil {
		return fmt.Errorf("git branch delete failed: %w", err)
	}
	return nil
}

// FindNestedWorktrees returns child worktree branch names found under
// the given branch's work/ directory. Only directories with a .git file
// (valid worktrees) are included; ghost directories are excluded.
func (m *Manager) FindNestedWorktrees(branch string) []string {
	childWorkDir := filepath.Join(m.Path(branch), "work")
	entries, err := os.ReadDir(childWorkDir)
	if err != nil {
		return nil
	}

	var result []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		gitPath := filepath.Join(childWorkDir, entry.Name(), ".git")
		if _, statErr := os.Stat(gitPath); statErr == nil {
			result = append(result, entry.Name())
		}
	}
	return result
}

// RemoteBranchExists checks if a branch exists on the remote (origin).
// Uses "git ls-remote --heads origin <branch>" to check.
// Returns true if the remote has a matching ref.
// In dry-run mode, this always returns true since the command succeeds with empty output.
func (m *Manager) RemoteBranchExists(branch string) bool {
	gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")
	output, err := m.CmdRunner.RunWithOpts(cmdexec.CheckOpt(), gitCmd, "ls-remote", "--heads", "origin", branch)
	if err != nil {
		return false
	}
	// In dry-run mode, output is "" and err is nil, so this returns true.
	// In real mode, ls-remote outputs "<hash>\trefs/heads/<branch>" if branch exists.
	// If the branch does not exist, ls-remote succeeds but outputs empty string.
	if !m.CmdRunner.DryRun && strings.TrimSpace(output) == "" {
		return false
	}
	return true
}

// FetchBranch fetches a specific branch from the remote (origin).
func (m *Manager) FetchBranch(branch string) error {
	gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")
	if _, err := m.CmdRunner.Run(gitCmd, "fetch", "origin", branch); err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}
	return nil
}

// Prune runs 'git worktree prune' to clean up stale worktree metadata
// entries that point to non-existent worktree directories.
func (m *Manager) Prune() error {
	gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")
	if _, err := m.CmdRunner.Run(gitCmd, "worktree", "prune"); err != nil {
		return fmt.Errorf("git worktree prune failed: %w", err)
	}
	return nil
}
