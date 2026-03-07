package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/escape-dev/devctl/internal/report"
	"github.com/escape-dev/devctl/internal/resolve"
)

var shellCmd = &cobra.Command{
	Use:   "shell <feature> [branch]",
	Short: "Open a shell in the container",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runShell,
}

func runShell(cmd *cobra.Command, args []string) error {
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

	if err := ctx.ActionRunner.Shell(containerName); err != nil {
		ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Shell", Success: false})
		ctx.Report.OverallResult = "FAILED"
		return fmt.Errorf("shell failed: %w", err)
	}
	ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Shell", Success: true})
	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
