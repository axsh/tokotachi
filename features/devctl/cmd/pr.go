package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/escape-dev/devctl/internal/report"
	"github.com/escape-dev/devctl/internal/resolve"
)

var prCmd = &cobra.Command{
	Use:   "pr <feature> [branch]",
	Short: "Create a GitHub Pull Request",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPR,
}

func runPR(cmd *cobra.Command, args []string) error {
	ctx, err := InitContext(args)
	if err != nil {
		return err
	}
	defer finalizeReport(ctx)

	worktreePath, err := resolve.Worktree(ctx.RepoRoot, ctx.Feature, ctx.Branch)
	if err != nil {
		return fmt.Errorf("worktree resolution failed: %w", err)
	}

	if err := ctx.ActionRunner.PR(worktreePath); err != nil {
		ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "PR create", Success: false})
		ctx.Report.OverallResult = "FAILED"
		return fmt.Errorf("pr failed: %w", err)
	}
	ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "PR create", Success: true})
	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
