package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/report"
	"github.com/axsh/tokotachi/features/tt/internal/resolve"
	"github.com/axsh/tokotachi/features/tt/internal/state"
)

var prCmd = &cobra.Command{
	Use:   "pr <branch> [feature]",
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

	worktreePath, err := resolve.Worktree(ctx.RepoRoot, ctx.Branch)
	if err != nil {
		return fmt.Errorf("worktree resolution failed: %w", err)
	}

	if err := ctx.ActionRunner.PR(worktreePath); err != nil {
		ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "PR create", Success: false})
		ctx.Report.OverallResult = "FAILED"
		return fmt.Errorf("pr failed: %w", err)
	}

	// Update CodeStatus to PR after successful PR creation
	statePath := state.StatePath(ctx.RepoRoot, ctx.Branch)
	sf, loadErr := state.Load(statePath)
	if loadErr == nil {
		now := time.Now()
		sf.CodeStatus = &state.CodeStatus{
			Status:        state.CodeStatusPR,
			PRCreatedAt:   &now,
			LastCheckedAt: &now,
		}
		if saveErr := state.Save(statePath, sf); saveErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update code status: %v\n", saveErr)
		}
	}

	ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "PR create", Success: true})
	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
