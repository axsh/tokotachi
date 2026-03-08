package action

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

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
	ProjectName string    // for resolving container names
	Depth       int       // max recursion depth for nested worktrees (default: 10)
	Yes         bool      // skip [y/N] confirmation prompt
	Stdin       io.Reader // input source for confirmation prompt
}

// Close performs the close sequence.
// With feature: stop that feature's container, remove from state.
// Without feature: stop all feature containers, remove worktree + branch + state.
// Nested worktrees are detected and closed recursively (depth-limited).
func (r *Runner) Close(opts CloseOptions, wm *worktree.Manager) error {
	// Phase 1: Detect nested worktrees (if depth allows)
	var nested []string
	hasDepthWarning := false
	if opts.Depth > 0 {
		nested = wm.FindNestedWorktrees(opts.Branch)
		if len(nested) > 0 {
			r.Logger.Info("Detected %d nested worktree(s) under %s: %v", len(nested), opts.Branch, nested)

			// Check if any children have further nesting beyond depth limit
			if opts.Depth == 1 {
				for _, child := range nested {
					grandchildren := wm.FindNestedWorktrees(child)
					if len(grandchildren) > 0 {
						hasDepthWarning = true
						break
					}
				}
			}
		}
	} else {
		// At depth limit, check if there are children we cannot reach
		possibleNested := wm.FindNestedWorktrees(opts.Branch)
		if len(possibleNested) > 0 {
			r.Logger.Warn("Depth limit reached: %d nested worktree(s) under %s will NOT be closed: %v",
				len(possibleNested), opts.Branch, possibleNested)
		}
	}

	// Phase 2: Confirmation prompt (skip for recursive child calls where Yes=true)
	if !opts.Yes {
		// Display preview
		r.Logger.Info("Close preview for branch: %s", opts.Branch)
		if len(nested) > 0 {
			r.Logger.Info("  Nested worktrees to close first: %v", nested)
		}
		if hasDepthWarning {
			r.Logger.Warn("  Depth limit (%d) may leave deeper nested worktrees behind.", opts.Depth)
		}

		// Ask for confirmation
		if opts.Stdin == nil {
			r.Logger.Info("Aborted (no input source for confirmation).")
			return nil
		}
		fmt.Fprint(os.Stderr, "Proceed? [y/N]: ")
		scanner := bufio.NewScanner(opts.Stdin)
		if scanner.Scan() {
			response := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if response != "y" && response != "yes" {
				r.Logger.Info("Aborted.")
				return nil
			}
		} else {
			r.Logger.Info("Aborted.")
			return nil
		}
	}

	// Phase 3: Recursively close nested worktrees (children first)
	if len(nested) > 0 && opts.Depth > 0 {
		for _, childBranch := range nested {
			r.Logger.Info("Recursively closing nested worktree: %s", childBranch)
			childOpts := CloseOptions{
				Branch:      childBranch,
				Force:       opts.Force,
				RepoRoot:    opts.RepoRoot,
				ProjectName: opts.ProjectName,
				Depth:       opts.Depth - 1,
				Yes:         true, // Already confirmed by parent
				Stdin:       nil,
			}
			if err := r.Close(childOpts, wm); err != nil {
				r.Logger.Warn("Failed to close nested worktree %s: %v", childBranch, err)
			}
		}
	}

	// Phase 4: Execute the close for this branch
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
		r.pruneIfForce(opts, wm)
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
	r.pruneIfForce(opts, wm)
	return nil
}

// pruneIfForce runs git worktree prune when --force is set.
func (r *Runner) pruneIfForce(opts CloseOptions, wm *worktree.Manager) {
	if opts.Force {
		r.Logger.Info("Pruning stale worktree metadata...")
		if err := wm.Prune(); err != nil {
			r.Logger.Warn("Worktree prune failed: %v", err)
		}
	}
}
