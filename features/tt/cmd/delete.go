package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/action"
	"github.com/axsh/tokotachi/features/tt/internal/report"
	"github.com/axsh/tokotachi/features/tt/internal/resolve"
	"github.com/axsh/tokotachi/features/tt/internal/worktree"
)

var (
	deleteFlagForce bool
	deleteFlagDepth int
	deleteFlagYes   bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete <branch>",
	Short: "Delete worktree and branch",
	Long:  "Remove worktree and delete branch. Fails if any Dev Container is still running.",
	Args:  cobra.ExactArgs(1),
	RunE:  runDelete,
}

func init() {
	deleteCmd.Flags().BoolVar(&deleteFlagForce, "force", false, "Force delete even if branch is not merged")
	deleteCmd.Flags().IntVar(&deleteFlagDepth, "depth", 10, "Maximum depth for recursive nested worktree deletion")
	deleteCmd.Flags().BoolVar(&deleteFlagYes, "yes", false, "Skip [y/N] confirmation and execute immediately")
}

func runDelete(cmd *cobra.Command, args []string) error {
	ctx, err := InitContext(args)
	if err != nil {
		return err
	}
	defer finalizeReport(ctx)

	globalCfg, _ := resolve.LoadGlobalConfig(ctx.RepoRoot)
	projectName := globalCfg.ProjectName
	if projectName == "" {
		projectName = "tt"
	}

	wm := &worktree.Manager{CmdRunner: ctx.CmdRunner, RepoRoot: ctx.RepoRoot}

	if err := ctx.ActionRunner.Delete(action.DeleteOptions{
		Branch:      ctx.Branch,
		Force:       deleteFlagForce,
		RepoRoot:    ctx.RepoRoot,
		ProjectName: projectName,
		Depth:       deleteFlagDepth,
		Yes:         deleteFlagYes,
		Stdin:       os.Stdin,
	}, wm); err != nil {
		ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Delete", Success: false})
		ctx.Report.OverallResult = "FAILED"
		return fmt.Errorf("delete failed: %w", err)
	}

	ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Delete", Success: true})
	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
