package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/devctl/internal/report"
	"github.com/axsh/tokotachi/features/devctl/internal/resolve"
	"github.com/axsh/tokotachi/features/devctl/internal/state"
)

var downCmd = &cobra.Command{
	Use:   "down <branch> <feature>",
	Short: "Stop the development container",
	Long:  "Stop and remove the container for the given feature. Requires feature argument.",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runDown,
}

func init() {}

func runDown(cmd *cobra.Command, args []string) error {
	ctx, err := InitContext(args)
	if err != nil {
		return err
	}
	defer finalizeReport(ctx)

	if !ctx.HasFeature() {
		return fmt.Errorf("feature is required for 'down' command (container operation)")
	}

	globalCfg, _ := resolve.LoadGlobalConfig(ctx.RepoRoot)
	projectName := globalCfg.ProjectName
	if projectName == "" {
		projectName = "devctl"
	}
	containerName := resolve.ContainerName(projectName, ctx.Feature)

	if err := ctx.ActionRunner.Down(containerName); err != nil {
		ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Container down", Success: false})
		ctx.Report.OverallResult = "FAILED"
		return fmt.Errorf("down failed: %w", err)
	}
	ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Container down", Success: true})

	// Update state file to stopped
	statePath := state.StatePath(ctx.RepoRoot, ctx.Feature, ctx.Branch)
	if s, err := state.Load(statePath); err == nil {
		s.Status = state.StatusStopped
		if err := state.Save(statePath, s); err != nil {
			ctx.Logger.Warn("Failed to update state file: %v", err)
		}
	}

	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
