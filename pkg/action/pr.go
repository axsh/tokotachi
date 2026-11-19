package action

import (
	"fmt"

	"github.com/axsh/tokotachi/internal/forge"
)

// PR creates a Pull Request (GitHub) or Merge Request (GitLab).
func (r *Runner) PR(repoRoot, worktreePath string) error {
	r.Logger.Info("Creating change request from %s...", worktreePath)

	client, host, err := forge.NewClientForRepoOpts(forge.ClientForRepoOpts{
		RepoRoot:    repoRoot,
		Runner:      r.CmdRunner,
		CLIOverride: r.ForgeOverride,
	})
	if err != nil {
		return fmt.Errorf("forge client creation failed: %w", err)
	}
	r.Logger.Info("Detected forge host: %s", host)
	r.Logger.Debug("forge client ready host=%s forgeOverride=%q worktree=%s", host, r.ForgeOverride, worktreePath)

	if err := client.CreateChangeRequest(worktreePath); err != nil {
		return fmt.Errorf("change request create failed: %w", err)
	}
	return nil
}
