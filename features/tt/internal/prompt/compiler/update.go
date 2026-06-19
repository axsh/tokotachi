package compiler

import (
	"fmt"

	"github.com/axsh/tokotachi/pkg/resolve"
)

// UpdateOptions holds options for the update pipeline.
type UpdateOptions struct {
	ProjectPath string
	Target      string // default: "all"
	Force       bool
	DryRun      bool
}

// UpdateResult holds the output of the update pipeline.
type UpdateResult struct {
	TargetResults map[string]*TargetUpdateResult
}

// TargetUpdateResult holds the result for a single target.
type TargetUpdateResult struct {
	Skipped      bool
	Reason       string // e.g., "no changes detected"
	DeployResult *DeployResult
}

// Update executes the update pipeline for the given targets.
// It resolves the target, and runs deploy for each target (always runs, no change check).
func Update(opts UpdateOptions) (*UpdateResult, error) {
	target := opts.Target
	if target == "" {
		target = "all"
	}

	targets, err := resolve.ResolveTargets(target)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve targets: %w", err)
	}

	result := &UpdateResult{
		TargetResults: make(map[string]*TargetUpdateResult),
	}

	for _, t := range targets {
		tr := &TargetUpdateResult{}

		// Run deploy
		deployResult, err := Deploy(DeployOptions{
			ProjectPath: opts.ProjectPath,
			Target:      t,
			Force:       opts.Force,
			DryRun:      opts.DryRun,
		})
		if err != nil {
			return nil, fmt.Errorf("deploy failed for target %s: %w", t, err)
		}
		tr.DeployResult = deployResult
		tr.Skipped = false
		tr.Reason = ""

		result.TargetResults[t] = tr
	}

	return result, nil
}

