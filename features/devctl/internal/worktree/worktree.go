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

// Path returns the worktree directory path.
// With feature: work/<branch>/features/<feature>
// Without feature (feature=""): work/<branch>/all
func (m *Manager) Path(feature, branch string) string {
	if feature == "" {
		return filepath.Join(m.RepoRoot, "work", branch, "all")
	}
	return filepath.Join(m.RepoRoot, "work", branch, "features", feature)
}

// Exists checks if the worktree directory exists.
func (m *Manager) Exists(feature, branch string) bool {
	info, err := os.Stat(m.Path(feature, branch))
	return err == nil && info.IsDir()
}

// Create creates a new git worktree.
// If the branch already exists, uses it without -b flag.
// Uses --force to handle branches already checked out in other worktrees.
func (m *Manager) Create(feature, branch string) error {
	wtPath := m.Path(feature, branch)
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
func (m *Manager) Remove(feature, branch string, force bool) error {
	wtPath := m.Path(feature, branch)
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

// List returns all feature worktree entries for a branch by scanning work/<branch>/features/.
func (m *Manager) List(branch string) ([]WorktreeInfo, error) {
	featuresDir := filepath.Join(m.RepoRoot, "work", branch, "features")
	entries, err := os.ReadDir(featuresDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read features directory: %w", err)
	}

	var result []WorktreeInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		result = append(result, WorktreeInfo{
			Feature: e.Name(),
			Branch:  branch,
			Path:    filepath.Join(featuresDir, e.Name()),
		})
	}
	return result, nil
}
