package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/devctl/internal/action"
	"github.com/axsh/tokotachi/features/devctl/internal/editor"
	"github.com/axsh/tokotachi/features/devctl/internal/plan"
	"github.com/axsh/tokotachi/features/devctl/internal/report"
	"github.com/axsh/tokotachi/features/devctl/internal/resolve"
	"github.com/axsh/tokotachi/features/devctl/internal/state"
	"github.com/axsh/tokotachi/features/devctl/internal/worktree"
)

var (
	openFlagEditor string
	openFlagAttach bool
	openFlagUp     bool
)

var openCmd = &cobra.Command{
	Use:   "open <branch> [feature]",
	Short: "Open the editor",
	Long:  "Open the editor for the given branch. Use --up to start the container if not running. Use --attach to reconnect to a running container.",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runOpen,
}

func init() {
	openCmd.Flags().StringVar(&openFlagEditor, "editor", "", "Editor to use (code|cursor|ag|claude)")
	openCmd.Flags().BoolVar(&openFlagAttach, "attach", false, "Attempt DevContainer attach to running container")
	openCmd.Flags().BoolVar(&openFlagUp, "up", false, "Start the container if not running before opening editor")
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

	// Container name only when feature is specified
	var containerName string
	if ctx.HasFeature() {
		containerName = resolve.ContainerName(projectName, ctx.Feature)
	}

	p := plan.Build(plan.Input{
		Feature:       ctx.Feature,
		OS:            currentOS,
		Editor:        ed,
		ContainerMode: containerMode,
		EditorOpen:    true,
		Attach:        openFlagAttach,
	})

	wm := &worktree.Manager{CmdRunner: ctx.CmdRunner, RepoRoot: ctx.RepoRoot}

	// --up flag: ensure worktree and container are ready
	if openFlagUp {
		// Create worktree if not exists
		if !wm.Exists(ctx.Feature, ctx.Branch) {
			ctx.Logger.Info("Worktree not found, creating %s...", wm.Path(ctx.Feature, ctx.Branch))
			if err := wm.Create(ctx.Feature, ctx.Branch); err != nil {
				return fmt.Errorf("worktree creation failed: %w", err)
			}
		}

		// Start container if feature is specified and container is not running
		if ctx.HasFeature() {
			worktreePath, err := resolve.Worktree(ctx.RepoRoot, ctx.Feature, ctx.Branch)
			if err != nil {
				worktreePath = wm.Path(ctx.Feature, ctx.Branch)
			}

			containerState := ctx.ActionRunner.Status(containerName, worktreePath)
			if containerState != action.StateContainerRunning {
				ctx.Logger.Info("Container not running, starting up...")

				imageName := resolve.ImageName(projectName, ctx.Feature)

				// Detect git worktree configuration
				var gitInfo resolve.GitWorktreeInfo
				var gitErr error
				gitInfo, gitErr = resolve.DetectGitWorktree(worktreePath)
				if gitErr != nil {
					ctx.Logger.Warn("Git worktree detection failed: %v", gitErr)
				}

				// Load devcontainer.json
				dcCfg, _ := resolve.LoadDevcontainerConfig(ctx.RepoRoot, ctx.Feature, ctx.Branch)

				upOpts := action.UpOptions{
					ContainerName: containerName,
					ImageName:     imageName,
					WorktreePath:  worktreePath,
					FeaturePath:   filepath.Join(ctx.RepoRoot, "features", ctx.Feature),
				}

				if !dcCfg.IsEmpty() {
					if dcCfg.HasDockerfile() {
						configDir := dcCfg.ConfigDir()
						upOpts.DockerfilePath = filepath.Join(configDir, dcCfg.Build.Dockerfile)
						if dcCfg.Build.Context != "" {
							upOpts.BuildContext = filepath.Join(configDir, dcCfg.Build.Context)
						} else {
							upOpts.BuildContext = configDir
						}
					}
					if dcCfg.Image != "" && !dcCfg.HasDockerfile() {
						upOpts.ImageName = dcCfg.Image
						upOpts.NoBuild = true
					}
					upOpts.WorkspaceFolder = dcCfg.WorkspaceFolder
					upOpts.Mounts = dcCfg.Mounts
					upOpts.ContainerEnv = dcCfg.ContainerEnv
					upOpts.RemoteUser = dcCfg.RemoteUser
				}

				if gitErr == nil && gitInfo.IsWorktree {
					upOpts.GitWorktree = &gitInfo
					// Create temp .git override file for container mount
					overrideFile, oErr := resolve.CreateContainerGitFile(os.TempDir())
					if oErr != nil {
						ctx.Logger.Warn("Failed to create git override file: %v", oErr)
					} else {
						upOpts.GitOverrideFile = overrideFile
					}
				}

				if err := ctx.ActionRunner.Up(upOpts); err != nil {
					return fmt.Errorf("up failed: %w", err)
				}

				// Save state
				statePath := state.StatePath(ctx.RepoRoot, ctx.Feature, ctx.Branch)
				_ = state.Save(statePath, state.StateFile{
					Feature:       ctx.Feature,
					Branch:        ctx.Branch,
					CreatedAt:     time.Now(),
					ContainerMode: string(containerMode),
					Editor:        string(ed),
					Status:        state.StatusActive,
				})
			}
		}
	}

	worktreePath, err := resolve.Worktree(ctx.RepoRoot, ctx.Feature, ctx.Branch)
	if err != nil {
		return fmt.Errorf("worktree resolution failed: %w", err)
	}

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
		return fmt.Errorf("open failed: %w", err)
	}
	ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Editor open", Success: true})
	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
