package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/internal/report"
	"github.com/axsh/tokotachi/pkg/worktree"
)

var createCmd = &cobra.Command{
	Use:   "create <branch>",
	Short: "Create a branch and worktree",
	Long:  "Create a new git branch and set up a worktree for development.",
	Args:  cobra.ExactArgs(1),
	RunE:  runCreate,
}

func runCreate(cmd *cobra.Command, args []string) error {
	ctx, err := InitContext(args)
	if err != nil {
		return err
	}
	defer finalizeReport(ctx)

	wm := &worktree.Manager{CmdRunner: ctx.CmdRunner, RepoRoot: ctx.RepoRoot}

	if wm.Exists(ctx.Branch) {
		ctx.Logger.Info("Worktree already exists for branch %s", ctx.Branch)
		ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Worktree creation", Success: true})
		ctx.Report.OverallResult = "SUCCESS"
		return nil
	}

	ctx.Logger.Info("Creating worktree for branch %s...", ctx.Branch)
	if err := wm.Create(ctx.Branch); err != nil {
		ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Worktree creation", Success: false})
		ctx.Report.OverallResult = "FAILED"
		return fmt.Errorf("worktree creation failed: %w", err)
	}

	ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Worktree creation", Success: true})
	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
