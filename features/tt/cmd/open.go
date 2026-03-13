package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/internal/report"
	"github.com/axsh/tokotachi/pkg/action"
	"github.com/axsh/tokotachi/pkg/editor"
	"github.com/axsh/tokotachi/pkg/plan"
	"github.com/axsh/tokotachi/pkg/resolve"
	"github.com/axsh/tokotachi/pkg/state"
	"github.com/axsh/tokotachi/pkg/worktree"
)

var (
	openFlagEditor string
)

var openCmd = &cobra.Command{
	Use:   "open <branch> [feature]",
	Short: "Create worktree, start container, and open editor",
	Long: "Syntax sugar: runs create → up → editor in sequence. " +
		"If feature is omitted, skips container start.",
	Args: cobra.RangeArgs(1, 2),
	RunE: runOpen,
}

func init() {
	openCmd.Flags().StringVar(&openFlagEditor, "editor", "", "Editor to use (code|cursor|ag|claude)")
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

	projectName := "tt"
	if projectName == "" {
		projectName = "tt"
	}

	// Step 1: Create worktree if not exists
	wm := &worktree.Manager{CmdRunner: ctx.CmdRunner, RepoRoot: ctx.RepoRoot}
	if !wm.Exists(ctx.Branch) {
		ctx.Logger.Info("Worktree not found, creating %s...", wm.Path(ctx.Branch))
		if err := wm.Create(ctx.Branch); err != nil {
			ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Worktree creation", Success: false})
			ctx.Report.OverallResult = "FAILED"
			return fmt.Errorf("worktree creation failed: %w", err)
		}
		ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Worktree creation", Success: true})
	}

	// Step 2: Start container if feature is specified
	var containerName string
	if ctx.HasFeature() {
		containerName = resolve.ContainerName(projectName, ctx.Feature)
		imageName := resolve.ImageName(projectName, ctx.Feature)

		worktreePath, wpErr := resolve.Worktree(ctx.RepoRoot, ctx.Branch)
		if wpErr != nil {
			worktreePath = wm.Path(ctx.Branch)
		}

		containerState := ctx.ActionRunner.Status(containerName, worktreePath)
		if containerState != action.StateContainerRunning {
			ctx.Logger.Info("Starting container...")

			// Detect git worktree configuration
			gitInfo, gitErr := resolve.DetectGitWorktree(worktreePath)
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
				overrideFile, oErr := resolve.CreateContainerGitFile(os.TempDir())
				if oErr != nil {
					ctx.Logger.Warn("Failed to create git override file: %v", oErr)
				} else {
					upOpts.GitOverrideFile = overrideFile
				}
			}

			if err := ctx.ActionRunner.Up(upOpts); err != nil {
				ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Container up", Success: false})
				ctx.Report.OverallResult = "FAILED"
				return fmt.Errorf("up failed: %w", err)
			}
			ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Container up", Success: true})

			// Save state
			statePath := state.StatePath(ctx.RepoRoot, ctx.Branch)
			sf, _ := state.Load(statePath)
			if sf.Branch == "" {
				sf.Branch = ctx.Branch
				sf.CreatedAt = time.Now()
			}
			sf.SetFeature(ctx.Feature, state.FeatureState{
				Status:    state.StatusActive,
				StartedAt: time.Now(),
				Connectivity: state.Connectivity{
					Docker: state.DockerConnectivity{
						Enabled:       true,
						ContainerName: containerName,
						Devcontainer:  !dcCfg.IsEmpty(),
					},
					SSH: state.SSHConnectivity{Enabled: false},
				},
			})
			// Initialize CodeStatus if not yet set
			if sf.CodeStatus == nil {
				sf.CodeStatus = &state.CodeStatus{
					Status: state.CodeStatusLocal,
				}
			}
			_ = state.Save(statePath, sf)
		}
	}

	// Step 3: Open editor
	worktreePath, err := resolve.Worktree(ctx.RepoRoot, ctx.Branch)
	if err != nil {
		if ctx.DryRun {
			worktreePath = wm.Path(ctx.Branch)
		} else {
			return fmt.Errorf("worktree resolution failed: %w", err)
		}
	}

	p := plan.Build(plan.Input{
		Feature:       ctx.Feature,
		OS:            currentOS,
		Editor:        ed,
		ContainerMode: containerMode,
		EditorOpen:    true,
	})

	launcher, err := editor.NewLauncher(ed)
	if err != nil {
		return fmt.Errorf("editor launcher creation failed: %w", err)
	}

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
