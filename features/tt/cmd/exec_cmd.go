package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/report"
	"github.com/axsh/tokotachi/features/tt/internal/resolve"
)

var execCmd = &cobra.Command{
	Use:   "exec <branch> <feature> -- <command...>",
	Short: "Execute a command in the container",
	Long:  "Execute a command inside the running container. Requires feature argument. Use -- to separate tt args from the command.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runExec,
}

func runExec(cmd *cobra.Command, args []string) error {
	// Split args: before "--" is branch [feature], after is the command
	var branch, feature string
	var execArgs []string

	dashIdx := cmd.ArgsLenAtDash()
	if dashIdx >= 0 {
		// args before -- are branch [feature]
		beforeDash := args[:dashIdx]
		execArgs = args[dashIdx:]
		branch = beforeDash[0]
		if len(beforeDash) >= 2 {
			feature = beforeDash[1]
		}
	} else {
		// No --, treat all remaining as branch [feature]
		branch = args[0]
		if len(args) >= 2 {
			feature = args[1]
		}
	}

	if feature == "" {
		return fmt.Errorf("feature is required for 'exec' command (container operation)")
	}

	ctxArgs := []string{branch, feature}
	ctx, err := InitContext(ctxArgs)
	if err != nil {
		return err
	}
	defer finalizeReport(ctx)

	if len(execArgs) == 0 {
		return fmt.Errorf("no command specified after --")
	}

	globalCfg, _ := resolve.LoadGlobalConfig(ctx.RepoRoot)
	projectName := globalCfg.ProjectName
	if projectName == "" {
		projectName = "tt"
	}
	containerName := resolve.ContainerName(projectName, ctx.Feature)

	if err := ctx.ActionRunner.Exec(containerName, execArgs); err != nil {
		ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Exec", Success: false})
		ctx.Report.OverallResult = "FAILED"
		return fmt.Errorf("exec failed: %w", err)
	}
	ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Exec", Success: true})
	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
