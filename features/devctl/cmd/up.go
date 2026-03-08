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
	upFlagEditor  string
	upFlagSSH     bool
	upFlagRebuild bool
	upFlagNoBuild bool
)

var upCmd = &cobra.Command{
	Use:   "up <branch> [feature]",
	Short: "Start the development container",
	Long:  "Start the container for the given branch and feature. If feature is omitted, only creates worktree without container.",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runUp,
}

func init() {
	upCmd.Flags().StringVar(&upFlagEditor, "editor", "", "Editor to use (code|cursor|ag|claude). Also opens the editor.")
	upCmd.Flags().BoolVar(&upFlagSSH, "ssh", false, "Enable SSH mode")
	upCmd.Flags().BoolVar(&upFlagRebuild, "rebuild", false, "Rebuild the container image")
	upCmd.Flags().BoolVar(&upFlagNoBuild, "no-build", false, "Skip image build")
}

func runUp(cmd *cobra.Command, args []string) error {
	ctx, err := InitContext(args)
	if err != nil {
		return err
	}
	defer finalizeReport(ctx)

	currentOS, ed, containerMode, err := ctx.ResolveEnvironment(upFlagEditor)
	if err != nil {
		return err
	}

	// Resolve identifiers
	globalCfg, _ := resolve.LoadGlobalConfig(ctx.RepoRoot)
	projectName := globalCfg.ProjectName
	if projectName == "" {
		projectName = "devctl"
	}

	// Container/image names only when feature is specified
	var containerName, imageName string
	if ctx.HasFeature() {
		containerName = resolve.ContainerName(projectName, ctx.Feature)
		imageName = resolve.ImageName(projectName, ctx.Feature)
	}

	// Auto-create worktree if not exists
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

	// Build execution plan
	editorOpen := upFlagEditor != ""
	p := plan.Build(plan.Input{
		Feature:       ctx.Feature,
		OS:            currentOS,
		Editor:        ed,
		ContainerMode: containerMode,
		Up:            true,
		EditorOpen:    editorOpen,
		SSH:           upFlagSSH,
		Rebuild:       upFlagRebuild,
		NoBuild:       upFlagNoBuild,
	})

	ctx.Logger.Debug("Plan: up=%v editor=%v devcontainer=%v", p.ShouldStartContainer, p.ShouldOpenEditor, p.TryDevcontainerAttach)

	// Resolve worktree path
	worktreePath, err := resolve.Worktree(ctx.RepoRoot, ctx.Branch)
	if err != nil {
		if ctx.DryRun {
			// In dry-run mode, worktree may not exist yet; use computed path
			worktreePath = wm.Path(ctx.Branch)
			ctx.Logger.Debug("Worktree not found (dry-run), using computed path: %s", worktreePath)
		} else {
			return fmt.Errorf("worktree resolution failed: %w", err)
		}
	}

	// Load or create state file
	statePath := state.StatePath(ctx.RepoRoot, ctx.Branch)
	sf, _ := state.Load(statePath)
	if sf.Branch == "" {
		sf.Branch = ctx.Branch
		sf.CreatedAt = time.Now()
	}

	// --- Container operations: only when feature is specified ---
	if ctx.HasFeature() {
		// Detect git worktree configuration
		gitInfo, gitErr := resolve.DetectGitWorktree(worktreePath)
		if gitErr != nil {
			ctx.Logger.Warn("Git worktree detection failed: %v", gitErr)
		} else if gitInfo.IsWorktree {
			ctx.Logger.Debug("Git worktree detected: mainGitDir=%s, worktreeGitDir=%s",
				gitInfo.MainGitDir, gitInfo.WorktreeGitDir)
		}

		// Load devcontainer.json
		dcCfg, _ := resolve.LoadDevcontainerConfig(ctx.RepoRoot, ctx.Feature, ctx.Branch)

		// Build UpOptions from DevcontainerConfig
		upOpts := action.UpOptions{
			ContainerName: containerName,
			ImageName:     imageName,
			WorktreePath:  worktreePath,
			FeaturePath:   filepath.Join(ctx.RepoRoot, "features", ctx.Feature),
			Rebuild:       p.Rebuild,
			NoBuild:       p.NoBuild,
			SSHMode:       p.SSHMode,
		}

		if !dcCfg.IsEmpty() {
			ctx.Logger.Debug("DevcontainerConfig loaded: name=%s", dcCfg.Name)

			// Resolve Dockerfile path and build context
			if dcCfg.HasDockerfile() {
				configDir := dcCfg.ConfigDir()
				upOpts.DockerfilePath = filepath.Join(configDir, dcCfg.Build.Dockerfile)
				if dcCfg.Build.Context != "" {
					upOpts.BuildContext = filepath.Join(configDir, dcCfg.Build.Context)
				} else {
					upOpts.BuildContext = configDir
				}
			}

			// Use image directly if no build config
			if dcCfg.Image != "" && !dcCfg.HasDockerfile() {
				upOpts.ImageName = dcCfg.Image
				upOpts.NoBuild = true
			}

			upOpts.WorkspaceFolder = dcCfg.WorkspaceFolder
			upOpts.Mounts = dcCfg.Mounts
			upOpts.ContainerEnv = dcCfg.ContainerEnv
			upOpts.RemoteUser = dcCfg.RemoteUser
		}

		// Set git worktree info if detected
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

		// Execute: up
		if err := ctx.ActionRunner.Up(upOpts); err != nil {
			ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Container up", Success: false})
			ctx.Report.OverallResult = "FAILED"
			return fmt.Errorf("up failed: %w", err)
		}
		ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Container up", Success: true})

		// Save feature state
		sf.SetFeature(ctx.Feature, state.FeatureState{
			Status:    state.StatusActive,
			StartedAt: time.Now(),
			Connectivity: state.Connectivity{
				Docker: state.DockerConnectivity{
					Enabled:       true,
					ContainerName: containerName,
					Devcontainer:  !dcCfg.IsEmpty(),
				},
				SSH: state.SSHConnectivity{Enabled: upFlagSSH},
			},
		})
	} else {
		ctx.Logger.Info("No feature specified — skipping container operations")
	}

	// Save state file
	if err := state.Save(statePath, sf); err != nil {
		ctx.Logger.Warn("Failed to save state file: %v", err)
	}

	// Execute: open editor if --editor was specified
	if p.ShouldOpenEditor {
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
	}

	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
