package action

import (
	"fmt"

	"github.com/axsh/tokotachi/features/devctl/internal/github"
)

// PR creates a GitHub Pull Request using github.Client.
// The underlying implementation uses gh CLI via the shared github package.
func (r *Runner) PR(worktreePath string) error {
	r.Logger.Info("Creating PR from %s...", worktreePath)

	client, err := github.NewClient("", github.WithCmdRunner(r.CmdRunner))
	if err != nil {
		return fmt.Errorf("github client creation failed: %w", err)
	}

	if err := client.CreatePR(worktreePath); err != nil {
		return fmt.Errorf("gh pr create failed: %w", err)
	}
	return nil
}
