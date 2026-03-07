package action

import (
	"fmt"

	"github.com/axsh/tokotachi/features/devctl/internal/cmdexec"
)

// PR creates a GitHub Pull Request using gh CLI.
// Executes gh pr create interactively in the given worktree directory.
func (r *Runner) PR(worktreePath string) error {
	ghCmd := cmdexec.ResolveCommand("DEVCTL_CMD_GH", "gh")
	r.Logger.Info("Creating PR from %s...", worktreePath)

	// gh pr create needs to be run from the worktree directory
	// Since RunInteractive doesn't support setting cwd, we use a wrapper
	if err := r.CmdRunner.RunInteractive(ghCmd, "pr", "create", "--repo-dir", worktreePath); err != nil {
		return fmt.Errorf("gh pr create failed: %w", err)
	}
	return nil
}
