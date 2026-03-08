package action

import (
	"os"

	"github.com/axsh/tokotachi/features/devctl/internal/resolve"
	"github.com/axsh/tokotachi/features/devctl/internal/state"
	"github.com/axsh/tokotachi/features/devctl/internal/worktree"
)

// CloseOptions holds parameters for the close action.
type CloseOptions struct {
	Feature     string // empty = close all features + worktree
	Branch      string
	Force       bool
	RepoRoot    string
	ProjectName string // for resolving container names
}

// Close performs the close sequence.
// With feature: stop that feature's container, remove from state.
// Without feature: stop all feature containers, remove worktree + branch + state.
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

			if len(sf.Features) == 0 {
				// Last feature closed: clean up worktree, branch, and state file
				if wm.Exists(opts.Branch) {
					r.Logger.Info("Removing worktree work/%s...", opts.Branch)
					if rmErr := wm.Remove(opts.Branch, opts.Force); rmErr != nil {
						r.Logger.Warn("Worktree remove failed: %v", rmErr)
						wtPath := wm.Path(opts.Branch)
						if dirErr := os.RemoveAll(wtPath); dirErr != nil {
							r.Logger.Warn("Directory cleanup also failed: %v", dirErr)
						} else {
							r.Logger.Info("Cleaned up worktree directory directly")
						}
					}
				}

				r.Logger.Info("Deleting branch %s...", opts.Branch)
				if brErr := wm.DeleteBranch(opts.Branch, opts.Force); brErr != nil {
					r.Logger.Warn("Branch delete failed: %v", brErr)
				}

				if rmErr := state.Remove(statePath); rmErr != nil {
					r.Logger.Warn("State file remove failed: %v", rmErr)
				}
			} else {
				// Other features remain: save updated state
				if saveErr := state.Save(statePath, sf); saveErr != nil {
					r.Logger.Warn("Failed to save state file: %v", saveErr)
				}
			}
		}

		r.Logger.Info("Close completed for feature %s on branch %s", opts.Feature, opts.Branch)
		return nil
	}

	// No feature: close all features + worktree
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

	// Only remove worktree if all containers stopped successfully
	if failCount > 0 {
		r.Logger.Warn("Skipping worktree removal: %d container(s) failed to stop", failCount)
		return nil
	}

	// Remove worktree
	if wm.Exists(opts.Branch) {
		r.Logger.Info("Removing worktree work/%s...", opts.Branch)
		if err := wm.Remove(opts.Branch, opts.Force); err != nil {
			r.Logger.Warn("Worktree remove failed: %v", err)
			// Fallback: remove directory directly
			wtPath := wm.Path(opts.Branch)
			if removeErr := os.RemoveAll(wtPath); removeErr != nil {
				r.Logger.Warn("Directory cleanup also failed: %v", removeErr)
			} else {
				r.Logger.Info("Cleaned up worktree directory directly")
			}
		}
	}

	// Delete branch
	r.Logger.Info("Deleting branch %s...", opts.Branch)
	if err := wm.DeleteBranch(opts.Branch, opts.Force); err != nil {
		r.Logger.Warn("Branch delete failed: %v", err)
	}

	// Remove state file
	if err := state.Remove(statePath); err != nil {
		r.Logger.Warn("State file remove failed: %v", err)
	}

	r.Logger.Info("Close completed for branch %s", opts.Branch)
	return nil
}
