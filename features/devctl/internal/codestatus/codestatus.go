package codestatus

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/axsh/tokotachi/features/devctl/internal/state"
)

// Checker resolves the code hosting status for branches.
type Checker struct {
	GitCmd   string        // git command path
	GhCmd    string        // gh command path
	RepoRoot string        // repository root
	Timeout  time.Duration // timeout for external commands
}

// BranchStatus holds the resolved status for a single branch.
type BranchStatus struct {
	Status      state.CodeStatusType
	PRCreatedAt *time.Time
}

// prInfo mirrors the JSON output from `gh pr list`.
type prInfo struct {
	Number    int       `json:"number"`
	CreatedAt time.Time `json:"createdAt"`
}

// Resolve checks the current code hosting status for a branch.
// It uses git ls-remote to check if the branch exists on the remote,
// and gh pr list to check for open PRs.
func (c *Checker) Resolve(ctx context.Context, branch string, prevStatus state.CodeStatusType) (BranchStatus, error) {
	// Step 1: Check if branch exists on remote
	remoteExists, err := c.hasRemoteBranch(ctx, branch)
	if err != nil {
		return BranchStatus{}, fmt.Errorf("failed to check remote branch: %w", err)
	}

	if !remoteExists {
		// Branch not on remote
		if prevStatus == state.CodeStatusHosted || prevStatus == state.CodeStatusPR {
			return BranchStatus{Status: state.CodeStatusDeleted}, nil
		}
		return BranchStatus{Status: state.CodeStatusLocal}, nil
	}

	// Step 2: Check for open PRs
	pr, err := c.findPR(ctx, branch)
	if err != nil {
		// If gh fails, fall back to hosted
		return BranchStatus{Status: state.CodeStatusHosted}, nil
	}

	if pr != nil {
		return BranchStatus{
			Status:      state.CodeStatusPR,
			PRCreatedAt: &pr.CreatedAt,
		}, nil
	}

	return BranchStatus{Status: state.CodeStatusHosted}, nil
}

// hasRemoteBranch checks if a branch exists on the remote using git ls-remote.
func (c *Checker) hasRemoteBranch(ctx context.Context, branch string) (bool, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, c.GitCmd, "ls-remote", "--heads", "origin", branch)
	cmd.Dir = c.RepoRoot
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

// findPR checks for an open PR with the given head branch.
func (c *Checker) findPR(ctx context.Context, branch string) (*prInfo, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, c.GhCmd, "pr", "list",
		"--head", branch,
		"--json", "number,createdAt",
		"--limit", "1",
	)
	cmd.Dir = c.RepoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	output := strings.TrimSpace(string(out))
	if output == "" || output == "[]" {
		return nil, nil
	}

	var prs []prInfo
	if err := json.Unmarshal([]byte(output), &prs); err != nil {
		return nil, err
	}
	if len(prs) == 0 {
		return nil, nil
	}
	return &prs[0], nil
}

// UpdateAll updates code status for all given branches in state files.
// Errors from individual branches are logged but do not stop processing.
func (c *Checker) UpdateAll(ctx context.Context, branches []string) error {
	now := time.Now()
	var lastErr error

	for _, branch := range branches {
		statePath := state.StatePath(c.RepoRoot, branch)
		sf, err := state.Load(statePath)
		if err != nil {
			lastErr = fmt.Errorf("failed to load state for %s: %w", branch, err)
			continue
		}

		// Determine previous status
		var prevStatus state.CodeStatusType
		if sf.CodeStatus != nil {
			prevStatus = sf.CodeStatus.Status
		}

		bs, err := c.Resolve(ctx, branch, prevStatus)
		if err != nil {
			lastErr = fmt.Errorf("failed to resolve status for %s: %w", branch, err)
			continue
		}

		sf.CodeStatus = &state.CodeStatus{
			Status:        bs.Status,
			PRCreatedAt:   bs.PRCreatedAt,
			LastCheckedAt: &now,
		}

		if err := state.Save(statePath, sf); err != nil {
			lastErr = fmt.Errorf("failed to save state for %s: %w", branch, err)
			continue
		}
	}

	return lastErr
}
