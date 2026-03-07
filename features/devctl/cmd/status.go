package cmd

import (
	"github.com/spf13/cobra"

	"github.com/escape-dev/devctl/internal/resolve"
)

var statusCmd = &cobra.Command{
	Use:   "status <feature> [branch]",
	Short: "Show feature status",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
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
	containerName := resolve.ContainerName(projectName, ctx.Feature)

	worktreePath, _ := resolve.Worktree(ctx.RepoRoot, ctx.Feature, ctx.Branch)
	ctx.ActionRunner.PrintStatus(ctx.Feature, containerName, worktreePath)
	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
