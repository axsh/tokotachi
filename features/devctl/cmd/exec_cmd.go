package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/devctl/internal/report"
	"github.com/axsh/tokotachi/features/devctl/internal/resolve"
)

var execCmd = &cobra.Command{
	Use:   "exec <feature> [branch] -- <command...>",
	Short: "Execute a command in the container",
	Long:  "Execute a command inside the running container. Use -- to separate devctl args from the command.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runExec,
}

func runExec(cmd *cobra.Command, args []string) error {
	// Split args: before "--" is feature/branch, after is the command
	feature := args[0]
	var branch string
	var execArgs []string

	dashIdx := cmd.ArgsLenAtDash()
	if dashIdx >= 0 {
		// args before -- are feature [branch]
		beforeDash := args[:dashIdx]
		execArgs = args[dashIdx:]
		feature = beforeDash[0]
		if len(beforeDash) >= 2 {
			branch = beforeDash[1]
		}
	} else {
		// No --, treat all remaining as feature [branch]
		if len(args) >= 2 {
			branch = args[1]
		}
	}
	if branch == "" {
		branch = feature
	}

	ctxArgs := []string{feature, branch}
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
		projectName = "devctl"
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
