package worktree

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/axsh/tokotachi/internal/cmdexec"
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

	// Check if branch already exists
	_, err := m.CmdRunner.RunWithOpts(cmdexec.CheckOpt(), gitCmd, "rev-parse", "--verify", branch)
	branchExists := err == nil

	var args []string
	if branchExists {
		// Branch exists: attach worktree to existing branch (force in case already checked out)
		args = []string{"worktree", "add", "--force", wtPath, branch}
	} else {
		// Branch does not exist: create new branch
		args = []string{"worktree", "add", "-b", branch, wtPath}
	}

	if _, err := m.CmdRunner.Run(gitCmd, args...); err != nil {
		return fmt.Errorf("git worktree add failed: %w", err)
	}
	return nil
}

// Remove removes a git worktree.
func (m *Manager) Remove(branch string, force bool) error {
	wtPath := m.Path(branch)
	gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")

	args := []string{"worktree", "remove", wtPath}
	if force {
		args = []string{"worktree", "remove", "-f", wtPath}
	}

	if _, err := m.CmdRunner.RunWithOpts(cmdexec.ToleratedOpt(), gitCmd, args...); err != nil {
		return fmt.Errorf("git worktree remove failed: %w", err)
	}

	// Ensure directory is fully removed (git may leave empty directory)
	if _, err := os.Stat(wtPath); err == nil {
		os.RemoveAll(wtPath)
	}
	return nil
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

// Prune runs 'git worktree prune' to clean up stale worktree metadata
// entries that point to non-existent worktree directories.
func (m *Manager) Prune() error {
	gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")
	if _, err := m.CmdRunner.Run(gitCmd, "worktree", "prune"); err != nil {
		return fmt.Errorf("git worktree prune failed: %w", err)
	}
	return nil
}
