package worktree

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/axsh/tokotachi/features/devctl/internal/cmdexec"
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

// Exists checks if the worktree directory exists.
func (m *Manager) Exists(branch string) bool {
	info, err := os.Stat(m.Path(branch))
	return err == nil && info.IsDir()
}

// Create creates a new git worktree.
// If the branch already exists, uses it without -b flag.
// Uses --force to handle branches already checked out in other worktrees.
func (m *Manager) Create(branch string) error {
	wtPath := m.Path(branch)
	gitCmd := cmdexec.ResolveCommand("DEVCTL_CMD_GIT", "git")

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
	gitCmd := cmdexec.ResolveCommand("DEVCTL_CMD_GIT", "git")

	args := []string{"worktree", "remove", wtPath}
	if force {
		args = []string{"worktree", "remove", "-f", wtPath}
	}

	if _, err := m.CmdRunner.RunWithOpts(cmdexec.ToleratedOpt(), gitCmd, args...); err != nil {
		return fmt.Errorf("git worktree remove failed: %w", err)
	}
	return nil
}

// DeleteBranch deletes the local branch.
// Uses -d for merged branches, -D if force is true.
func (m *Manager) DeleteBranch(branch string, force bool) error {
	gitCmd := cmdexec.ResolveCommand("DEVCTL_CMD_GIT", "git")

	flag := "-d"
	if force {
		flag = "-D"
	}

	if _, err := m.CmdRunner.RunWithOpts(cmdexec.ToleratedOpt(), gitCmd, "branch", flag, branch); err != nil {
		return fmt.Errorf("git branch delete failed: %w", err)
	}
	return nil
}
