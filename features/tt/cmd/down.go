package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/internal/report"
	"github.com/axsh/tokotachi/pkg/resolve"
	"github.com/axsh/tokotachi/pkg/state"
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
		projectName = "tt"
	}
	containerName := resolve.ContainerName(projectName, ctx.Feature)

	if err := ctx.ActionRunner.Down(containerName); err != nil {
		ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Container down", Success: false})
		ctx.Report.OverallResult = "FAILED"
		return fmt.Errorf("down failed: %w", err)
	}
	ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Container down", Success: true})

	// Update state file: change feature status to stopped (preserving connectivity)
	statePath := state.StatePath(ctx.RepoRoot, ctx.Branch)
	if sf, err := state.Load(statePath); err == nil {
		if err := sf.UpdateFeatureStatus(ctx.Feature, state.StatusStopped); err != nil {
			ctx.Logger.Warn("Failed to update feature status: %v", err)
		} else {
			if err := state.Save(statePath, sf); err != nil {
				ctx.Logger.Warn("Failed to save state file: %v", err)
			}
		}
	}

	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
