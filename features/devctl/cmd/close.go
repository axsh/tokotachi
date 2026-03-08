package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/devctl/internal/action"
	"github.com/axsh/tokotachi/features/devctl/internal/report"
	"github.com/axsh/tokotachi/features/devctl/internal/resolve"
	"github.com/axsh/tokotachi/features/devctl/internal/worktree"
)

var closeFlagForce bool

var closeCmd = &cobra.Command{
	Use:   "close <branch> [feature]",
	Short: "Close the development environment",
	Long:  "Stop container (if feature specified), remove worktree and delete branch.",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runClose,
}

func init() {
	closeCmd.Flags().BoolVar(&closeFlagForce, "force", false, "Force delete even if branch is not merged")
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
		projectName = "devctl"
	}

	wm := &worktree.Manager{CmdRunner: ctx.CmdRunner, RepoRoot: ctx.RepoRoot}

	if err := ctx.ActionRunner.Close(action.CloseOptions{
		Feature:     ctx.Feature,
		Branch:      ctx.Branch,
		Force:       closeFlagForce,
		RepoRoot:    ctx.RepoRoot,
		ProjectName: projectName,
	}, wm); err != nil {
		ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Close", Success: false})
		ctx.Report.OverallResult = "FAILED"
		return fmt.Errorf("close failed: %w", err)
	}
	ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Close", Success: true})
	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
