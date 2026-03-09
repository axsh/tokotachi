package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/action"
	"github.com/axsh/tokotachi/features/tt/internal/plan"
	"github.com/axsh/tokotachi/features/tt/internal/report"
	"github.com/axsh/tokotachi/features/tt/internal/resolve"
	"github.com/axsh/tokotachi/features/tt/internal/state"
	"github.com/axsh/tokotachi/features/tt/internal/worktree"
)

var (
	upFlagSSH     bool
	upFlagRebuild bool
	upFlagNoBuild bool
)

var upCmd = &cobra.Command{
	Use:   "up <branch> <feature>",
	Short: "Start the development container",
	Long:  "Start the container for the given branch and feature. Worktree must already exist (use 'tt create' first).",
	Args:  cobra.ExactArgs(2),
	RunE:  runUp,
}

func init() {
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

	currentOS, _, containerMode, err := ctx.ResolveEnvironment("")
	if err != nil {
		return err
	}

	// Resolve identifiers
	globalCfg, _ := resolve.LoadGlobalConfig(ctx.RepoRoot)
	projectName := globalCfg.ProjectName
	if projectName == "" {
		projectName = "tt"
	}

	containerName := resolve.ContainerName(projectName, ctx.Feature)
	imageName := resolve.ImageName(projectName, ctx.Feature)

	// Worktree must already exist
	wm := &worktree.Manager{CmdRunner: ctx.CmdRunner, RepoRoot: ctx.RepoRoot}
	if !wm.Exists(ctx.Branch) {
		ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Worktree check", Success: false})
		ctx.Report.OverallResult = "FAILED"
		return fmt.Errorf("worktree not found for branch %s. Use 'tt create %s' first", ctx.Branch, ctx.Branch)
	}

	// Build execution plan
	p := plan.Build(plan.Input{
		Feature:       ctx.Feature,
		OS:            currentOS,
		Editor:        "",
		ContainerMode: containerMode,
		Up:            true,
		EditorOpen:    false,
		SSH:           upFlagSSH,
		Rebuild:       upFlagRebuild,
		NoBuild:       upFlagNoBuild,
	})

	ctx.Logger.Debug("Plan: up=%v devcontainer=%v", p.ShouldStartContainer, p.TryDevcontainerAttach)

	// Resolve worktree path
	worktreePath, err := resolve.Worktree(ctx.RepoRoot, ctx.Branch)
	if err != nil {
		if ctx.DryRun {
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

	// Initialize CodeStatus if not yet set (new branch defaults to local)
	if sf.CodeStatus == nil {
		sf.CodeStatus = &state.CodeStatus{
			Status: state.CodeStatusLocal,
		}
	}

	// Save state file
	if err := state.Save(statePath, sf); err != nil {
		ctx.Logger.Warn("Failed to save state file: %v", err)
	}

	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
