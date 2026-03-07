package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/escape-dev/devctl/internal/editor"
	"github.com/escape-dev/devctl/internal/plan"
	"github.com/escape-dev/devctl/internal/report"
	"github.com/escape-dev/devctl/internal/resolve"
)

var (
	openFlagEditor string
	openFlagAttach bool
)

var openCmd = &cobra.Command{
	Use:   "open <feature> [branch]",
	Short: "Open the editor",
	Long:  "Open the editor for the given feature. Use --attach to reconnect to a running container.",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runOpen,
}

func init() {
	openCmd.Flags().StringVar(&openFlagEditor, "editor", "", "Editor to use (code|cursor|ag|claude)")
	openCmd.Flags().BoolVar(&openFlagAttach, "attach", false, "Attempt DevContainer attach to running container")
}

func runOpen(cmd *cobra.Command, args []string) error {
	ctx, err := InitContext(args)
	if err != nil {
		return err
	}
	defer finalizeReport(ctx)

	currentOS, ed, containerMode, err := ctx.ResolveEnvironment(openFlagEditor)
	if err != nil {
		return err
	}

	globalCfg, _ := resolve.LoadGlobalConfig(ctx.RepoRoot)
	projectName := globalCfg.ProjectName
	if projectName == "" {
		projectName = "devctl"
	}
	containerName := resolve.ContainerName(projectName, ctx.Feature)

	p := plan.Build(plan.Input{
		Feature:       ctx.Feature,
		OS:            currentOS,
		Editor:        ed,
		ContainerMode: containerMode,
		EditorOpen:    true,
		Attach:        openFlagAttach,
	})

	worktreePath, err := resolve.Worktree(ctx.RepoRoot, ctx.Feature, ctx.Branch)
	if err != nil {
		return fmt.Errorf("worktree resolution failed: %w", err)
	}

	launcher, err := editor.NewLauncher(ed)
	if err != nil {
		return fmt.Errorf("editor launcher creation failed: %w", err)
	}

	if _, err := ctx.ActionRunner.Open(launcher, editor.LaunchOptions{
		WorktreePath:    worktreePath,
		ContainerName:   containerName,
		NewWindow:       true,
		TryDevcontainer: p.TryDevcontainerAttach,
		Logger:          ctx.Logger,
		DryRun:          ctx.DryRun,
		CmdRunner:       ctx.CmdRunner,
	}); err != nil {
		ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Editor open", Success: false})
		ctx.Report.OverallResult = "FAILED"
		return fmt.Errorf("open failed: %w", err)
	}
	ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Editor open", Success: true})
	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
