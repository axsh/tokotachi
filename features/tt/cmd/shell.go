package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/internal/report"
	"github.com/axsh/tokotachi/pkg/resolve"
)

var shellCmd = &cobra.Command{
	Use:   "shell <branch> <feature>",
	Short: "Open a shell in the development container",
	Long:  "Open an interactive shell in the running container. Requires feature argument.",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runShell,
}

func init() {}

func runShell(cmd *cobra.Command, args []string) error {
	ctx, err := InitContext(args)
	if err != nil {
		return err
	}
	defer finalizeReport(ctx)

	if !ctx.HasFeature() {
		return fmt.Errorf("feature is required for 'shell' command (container operation)")
	}

	projectName := "tt"
	if projectName == "" {
		projectName = "tt"
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
