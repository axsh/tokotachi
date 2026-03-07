package action

import (
	"fmt"

	"github.com/escape-dev/devctl/internal/state"
	"github.com/escape-dev/devctl/internal/worktree"
)

// CloseOptions holds parameters for the close action.
type CloseOptions struct {
	ContainerName string
	Feature       string
	Branch        string
	Force         bool
	RepoRoot      string
}

// Close performs the full close sequence:
// 1. Down container (if running)
// 2. Remove worktree
// 3. Delete branch
// 4. Remove state file
func (r *Runner) Close(opts CloseOptions, wm *worktree.Manager) error {
	// Step 1: Down container if running
	containerState := r.Status(opts.ContainerName, wm.Path(opts.Feature, opts.Branch))
	if containerState == StateContainerRunning || containerState == StateContainerStopped {
		r.Logger.Info("Stopping container before close...")
		if err := r.Down(opts.ContainerName); err != nil {
			r.Logger.Warn("Container down failed (may already be removed): %v", err)
		}
	}

	// Step 2: Remove worktree
	if wm.Exists(opts.Feature, opts.Branch) {
		r.Logger.Info("Removing worktree work/%s/%s...", opts.Feature, opts.Branch)
		if err := wm.Remove(opts.Feature, opts.Branch, opts.Force); err != nil {
			return fmt.Errorf("worktree remove failed: %w", err)
		}
	}

	// Step 3: Delete branch
	r.Logger.Info("Deleting branch %s...", opts.Branch)
	if err := wm.DeleteBranch(opts.Branch, opts.Force); err != nil {
		r.Logger.Warn("Branch delete failed: %v", err)
	}

	// Step 4: Remove state file
	statePath := state.StatePath(opts.RepoRoot, opts.Feature, opts.Branch)
	if err := state.Remove(statePath); err != nil {
		r.Logger.Warn("State file remove failed: %v", err)
	}

	r.Logger.Info("Close completed for %s/%s", opts.Feature, opts.Branch)
	return nil
}
