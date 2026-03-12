package action

import (
	"fmt"
	"io"

	"github.com/axsh/tokotachi/features/tt/internal/resolve"
	"github.com/axsh/tokotachi/features/tt/internal/state"
	"github.com/axsh/tokotachi/features/tt/internal/worktree"
)

// CloseOptions holds parameters for the close action (Syntax Sugar).
type CloseOptions struct {
	Feature     string // empty = close all features + delete
	Branch      string
	Force       bool
	RepoRoot    string
	ProjectName string    // for resolving container names
	Depth       int       // max recursion depth for nested worktrees (default: 10)
	Yes         bool      // skip [y/N] confirmation prompt
	Verbose     bool      // show all pending changes without truncation
	Stdin       io.Reader // input source for confirmation prompt
}

// Close performs the close sequence (Syntax Sugar).
// With feature: stop that feature's container, remove from state,
//
//	then delete worktree/branch if all containers are stopped.
//
// Without feature: stop all feature containers, then delete worktree/branch.
func (r *Runner) Close(opts CloseOptions, wm *worktree.Manager) error {
	statePath := state.StatePath(opts.RepoRoot, opts.Branch)

	if opts.Feature != "" {
		// Feature-specific close: stop only this container
		containerName := resolve.ContainerName(opts.ProjectName, opts.Feature)
		if containerName != "" {
			containerState := r.Status(containerName, wm.Path(opts.Branch))
			if containerState == StateContainerRunning || containerState == StateContainerStopped {
				r.Logger.Info("Stopping container %s...", containerName)
				if err := r.Down(containerName); err != nil {
					r.Logger.Warn("Container down failed (may already be removed): %v", err)
				}
			}
		}

		// Remove feature entry from state
		sf, err := state.Load(statePath)
		if err == nil {
			sf.RemoveFeature(opts.Feature)

			if !sf.HasActiveFeatures() && len(sf.Features) == 0 {
				// All features removed: remove state file first, then delegate to Delete
				if rmErr := state.Remove(statePath); rmErr != nil {
					r.Logger.Warn("State file remove failed: %v", rmErr)
				}
				r.Logger.Info("All features closed, deleting worktree and branch...")
				// Check for pending changes before deletion
				worktreePath := wm.Path(opts.Branch)
				if !r.checkPendingChangesAndConfirm(opts, worktreePath) {
					r.Logger.Info("Aborted.")
					return nil
				}
				deleteOpts := DeleteOptions{
					Branch:      opts.Branch,
					Force:       opts.Force,
					RepoRoot:    opts.RepoRoot,
					ProjectName: opts.ProjectName,
					Depth:       opts.Depth,
					Yes:         opts.Yes,
					Stdin:       opts.Stdin,
				}
				return r.Delete(deleteOpts, wm)
			}
			// Other features remain: save updated state
			if saveErr := state.Save(statePath, sf); saveErr != nil {
				r.Logger.Warn("Failed to save state file: %v", saveErr)
			}
		}

		r.Logger.Info("Close completed for feature %s on branch %s", opts.Feature, opts.Branch)
		return nil
	}

	// No feature: close all features + delete
	sf, loadErr := state.Load(statePath)
	failCount := 0

	if loadErr == nil {
		// Stop all active feature containers
		for _, featureName := range sf.ActiveFeatureNames() {
			containerName := resolve.ContainerName(opts.ProjectName, featureName)
			if containerName == "" {
				continue
			}
			containerState := r.Status(containerName, wm.Path(opts.Branch))
			if containerState == StateContainerRunning || containerState == StateContainerStopped {
				r.Logger.Info("Stopping container %s...", containerName)
				if err := r.Down(containerName); err != nil {
					r.Logger.Warn("Container down failed for %s: %v", containerName, err)
					failCount++
				}
			}
		}
	}

	// Only delete if all containers stopped successfully
	if failCount > 0 {
		r.Logger.Warn("Skipping worktree removal: %d container(s) failed to stop", failCount)
		return fmt.Errorf("close failed: %d container(s) failed to stop", failCount)
	}

	// Remove state file before calling Delete to avoid safety guard conflict
	if rmErr := state.Remove(statePath); rmErr != nil {
		r.Logger.Warn("State file remove failed: %v", rmErr)
	}

	// Check for pending changes before deletion
	worktreePath := wm.Path(opts.Branch)
	if !r.checkPendingChangesAndConfirm(opts, worktreePath) {
		r.Logger.Info("Aborted.")
		return nil
	}

	// Delegate cleanup to Delete
	deleteOpts := DeleteOptions{
		Branch:      opts.Branch,
		Force:       opts.Force,
		RepoRoot:    opts.RepoRoot,
		ProjectName: opts.ProjectName,
		Depth:       opts.Depth,
		Yes:         opts.Yes,
		Stdin:       opts.Stdin,
	}
	return r.Delete(deleteOpts, wm)
}
