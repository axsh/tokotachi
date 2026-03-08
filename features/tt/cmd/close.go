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

var closeFlagForce bool
var closeFlagDepth int
var closeFlagYes bool

var closeCmd = &cobra.Command{
	Use:   "close <branch> [feature]",
	Short: "Stop containers and delete worktree (syntax sugar)",
	Long:  "Syntax sugar: runs down → delete in sequence. If feature specified, stops only that container.",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runClose,
}

func init() {
	closeCmd.Flags().BoolVar(&closeFlagForce, "force", false, "Force delete even if branch is not merged")
	closeCmd.Flags().IntVar(&closeFlagDepth, "depth", 10, "Maximum depth for recursive nested worktree close")
	closeCmd.Flags().BoolVar(&closeFlagYes, "yes", false, "Skip [y/N] confirmation and execute immediately")
}

func runClose(cmd *cobra.Command, args []string) error {
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

	if err := ctx.ActionRunner.Close(action.CloseOptions{
		Feature:     ctx.Feature,
		Branch:      ctx.Branch,
		Force:       closeFlagForce,
		RepoRoot:    ctx.RepoRoot,
		ProjectName: projectName,
		Depth:       closeFlagDepth,
		Yes:         closeFlagYes,
		Stdin:       os.Stdin,
	}, wm); err != nil {
		ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Close", Success: false})
		ctx.Report.OverallResult = "FAILED"
		return fmt.Errorf("close failed: %w", err)
	}
	ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Close", Success: true})
	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
