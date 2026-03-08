package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/editor"
	"github.com/axsh/tokotachi/features/tt/internal/plan"
	"github.com/axsh/tokotachi/features/tt/internal/report"
	"github.com/axsh/tokotachi/features/tt/internal/resolve"
)

var (
	editorFlagEditor string
	editorFlagAttach bool
)

var editorCmd = &cobra.Command{
	Use:   "editor <branch> [feature]",
	Short: "Open the editor",
	Long:  "Open the editor for the given branch. Use --attach to reconnect to a running container.",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runEditor,
}

func init() {
	editorCmd.Flags().StringVar(&editorFlagEditor, "editor", "", "Editor to use (code|cursor|ag|claude)")
	editorCmd.Flags().BoolVar(&editorFlagAttach, "attach", false, "Attempt DevContainer attach to running container")
}

func runEditor(cmd *cobra.Command, args []string) error {
	ctx, err := InitContext(args)
	if err != nil {
		return err
	}
	defer finalizeReport(ctx)

	_, ed, containerMode, err := ctx.ResolveEnvironment(editorFlagEditor)
	if err != nil {
		return err
	}

	globalCfg, _ := resolve.LoadGlobalConfig(ctx.RepoRoot)
	projectName := globalCfg.ProjectName
	if projectName == "" {
		projectName = "tt"
	}

	// Container name only when feature is specified
	var containerName string
	if ctx.HasFeature() {
		containerName = resolve.ContainerName(projectName, ctx.Feature)
	}

	worktreePath, err := resolve.Worktree(ctx.RepoRoot, ctx.Branch)
	if err != nil {
		return fmt.Errorf("worktree resolution failed: %w", err)
	}

	p := plan.Build(plan.Input{
		Feature:       ctx.Feature,
		OS:            "",
		Editor:        ed,
		ContainerMode: containerMode,
		EditorOpen:    true,
		Attach:        editorFlagAttach,
	})

	launcher, err := editor.NewLauncher(ed)
	if err != nil {
		return fmt.Errorf("editor launcher creation failed: %w", err)
	}

	// When no feature, skip devcontainer attach
	tryDevcontainer := p.TryDevcontainerAttach && ctx.HasFeature()
	if _, err := ctx.ActionRunner.Open(launcher, editor.LaunchOptions{
		WorktreePath:    worktreePath,
		ContainerName:   containerName,
		NewWindow:       true,
		TryDevcontainer: tryDevcontainer,
		Logger:          ctx.Logger,
		DryRun:          ctx.DryRun,
		CmdRunner:       ctx.CmdRunner,
	}); err != nil {
		ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Editor open", Success: false})
		ctx.Report.OverallResult = "FAILED"
		return fmt.Errorf("editor open failed: %w", err)
	}

	ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Editor open", Success: true})
	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
