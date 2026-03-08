package action

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/axsh/tokotachi/features/tt/internal/state"
	"github.com/axsh/tokotachi/features/tt/internal/worktree"
)

// DeleteOptions holds parameters for the delete action.
type DeleteOptions struct {
	Branch      string
	Force       bool
	RepoRoot    string
	ProjectName string
	Depth       int       // max recursion depth for nested worktrees (default: 10)
	Yes         bool      // skip [y/N] confirmation prompt
	Stdin       io.Reader // input source for confirmation prompt
}

// Delete removes worktree, deletes branch, and cleans up state file.
// Returns error if any active containers are found (safety guard).
// Nested worktrees are detected and deleted recursively (depth-limited).
func (r *Runner) Delete(opts DeleteOptions, wm *worktree.Manager) error {
	// Safety guard: reject deletion if active containers exist
	statePath := state.StatePath(opts.RepoRoot, opts.Branch)
	sf, loadErr := state.Load(statePath)
	if loadErr == nil && sf.HasActiveFeatures() {
		return fmt.Errorf("cannot delete branch %s: active container(s) found. Stop them first with 'tt down'", opts.Branch)
	}

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
			r.Logger.Warn("Depth limit reached: %d nested worktree(s) under %s will NOT be deleted: %v",
				len(possibleNested), opts.Branch, possibleNested)
		}
	}

	// Phase 2: Confirmation prompt (skip for recursive child calls where Yes=true)
	if !opts.Yes {
		// Display preview
		r.Logger.Info("Delete preview for branch: %s", opts.Branch)
		if len(nested) > 0 {
			r.Logger.Info("  Nested worktrees to delete first: %v", nested)
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

	// Phase 3: Recursively delete nested worktrees (children first)
	if len(nested) > 0 && opts.Depth > 0 {
		for _, childBranch := range nested {
			r.Logger.Info("Recursively deleting nested worktree: %s", childBranch)
			childOpts := DeleteOptions{
				Branch:      childBranch,
				Force:       opts.Force,
				RepoRoot:    opts.RepoRoot,
				ProjectName: opts.ProjectName,
				Depth:       opts.Depth - 1,
				Yes:         true, // Already confirmed by parent
				Stdin:       nil,
			}
			if err := r.Delete(childOpts, wm); err != nil {
				r.Logger.Warn("Failed to delete nested worktree %s: %v", childBranch, err)
			}
		}
	}

	// Phase 4: Remove worktree, branch, and state file
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

	r.Logger.Info("Deleting branch %s...", opts.Branch)
	if err := wm.DeleteBranch(opts.Branch, opts.Force); err != nil {
		r.Logger.Warn("Branch delete failed: %v", err)
	}

	if err := state.Remove(statePath); err != nil {
		r.Logger.Warn("State file remove failed: %v", err)
	}

	r.Logger.Info("Delete completed for branch %s", opts.Branch)
	if opts.Force {
		r.Logger.Info("Pruning stale worktree metadata...")
		if err := wm.Prune(); err != nil {
			r.Logger.Warn("Worktree prune failed: %v", err)
		}
	}
	return nil
}
