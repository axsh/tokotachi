package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/escape-dev/devctl/internal/action"
	"github.com/escape-dev/devctl/internal/detect"
	"github.com/escape-dev/devctl/internal/editor"
	"github.com/escape-dev/devctl/internal/log"
	"github.com/escape-dev/devctl/internal/matrix"
	"github.com/escape-dev/devctl/internal/plan"
	"github.com/escape-dev/devctl/internal/resolve"
)

var (
	flagUp      bool
	flagOpen    bool
	flagDown    bool
	flagStatus  bool
	flagShell   bool
	flagExec    []string
	flagEditor  string
	flagSSH     bool
	flagVerbose bool
	flagDryRun  bool
	flagForce   bool
	flagRebuild bool
	flagNoBuild bool
)

var rootCmd = &cobra.Command{
	Use:   "devctl <feature> [flags]",
	Short: "Development environment orchestrator",
	Long:  "devctl manages feature-level development environments with matrix-driven control.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runRoot,
}

func init() {
	rootCmd.Flags().BoolVar(&flagUp, "up", false, "Start the container")
	rootCmd.Flags().BoolVar(&flagOpen, "open", false, "Open the editor")
	rootCmd.Flags().BoolVar(&flagDown, "down", false, "Stop and remove the container")
	rootCmd.Flags().BoolVar(&flagStatus, "status", false, "Show feature status")
	rootCmd.Flags().BoolVar(&flagShell, "shell", false, "Open a shell in the container")
	rootCmd.Flags().StringSliceVar(&flagExec, "exec", nil, "Execute a command in the container")
	rootCmd.Flags().StringVar(&flagEditor, "editor", "", "Editor to use (code|cursor|ag|claude)")
	rootCmd.Flags().BoolVar(&flagSSH, "ssh", false, "Enable SSH mode")
	rootCmd.Flags().BoolVar(&flagVerbose, "verbose", false, "Show debug logs")
	rootCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Show planned actions without executing")
	rootCmd.Flags().BoolVar(&flagForce, "force", false, "Skip confirmation prompts")
	rootCmd.Flags().BoolVar(&flagRebuild, "rebuild", false, "Rebuild the container image")
	rootCmd.Flags().BoolVar(&flagNoBuild, "no-build", false, "Skip image build")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func runRoot(cmd *cobra.Command, args []string) error {
	feature := args[0]
	logger := log.New(os.Stderr, flagVerbose)
	repoRoot := "." // TODO: detect git root in future

	// Step 1: Detect environment
	currentOS := detect.CurrentOS()
	logger.Debug("OS=%s", currentOS)

	// Step 2: Load configuration
	globalCfg, err := resolve.LoadGlobalConfig(repoRoot)
	if err != nil {
		logger.Warn("Failed to load .devrc.yaml: %v", err)
	}
	featureCfg, err := resolve.LoadFeatureConfig(repoRoot, feature)
	if err != nil {
		logger.Warn("Failed to load feature.yaml: %v", err)
	}

	// Step 3: Resolve editor
	ed, err := detect.ResolveEditor(
		flagEditor,
		os.Getenv(detect.EnvKeyEditor),
		featureCfg.Dev.EditorDefault,
		globalCfg.DefaultEditor,
	)
	if err != nil {
		return fmt.Errorf("editor resolution failed: %w", err)
	}
	logger.Debug("Editor=%s", ed)

	// Step 4: Validate action
	if !flagUp && !flagOpen && !flagDown && !flagStatus && !flagShell && len(flagExec) == 0 {
		return fmt.Errorf("no action specified; use --up, --open, --down, --status, --shell, or --exec")
	}

	// Step 5: Resolve identifiers
	containerMode := matrix.ContainerMode(globalCfg.DefaultContainerMode)
	projectName := globalCfg.ProjectName
	if projectName == "" {
		projectName = "devctl"
	}
	containerName := resolve.ContainerName(projectName, feature)
	imageName := resolve.ImageName(projectName, feature)

	// Step 6: Build execution plan
	p := plan.Build(plan.Input{
		Feature:       feature,
		OS:            currentOS,
		Editor:        ed,
		ContainerMode: containerMode,
		Up:            flagUp,
		Open:          flagOpen,
		Down:          flagDown,
		Status:        flagStatus,
		Shell:         flagShell,
		Exec:          flagExec,
		SSH:           flagSSH,
		Rebuild:       flagRebuild,
		NoBuild:       flagNoBuild,
	})

	// Decision log
	logger.Debug("ContainerMode=%s CompatLevel=%s", containerMode, p.CompatLevel)
	logger.Debug("Plan: up=%v open=%v down=%v status=%v shell=%v exec=%v ssh=%v devcontainer=%v",
		p.ShouldStartContainer, p.ShouldOpenEditor, p.ShouldStopContainer,
		p.ShouldShowStatus, p.ShouldOpenShell, p.ExecCommand, p.SSHMode, p.TryDevcontainerAttach)

	// Step 7: Execute plan
	runner := &action.Runner{Logger: logger, DryRun: flagDryRun}

	// Resolve worktree path (needed for most actions)
	worktreePath, err := resolve.Worktree(repoRoot, feature)
	if err != nil && (p.ShouldStartContainer || p.ShouldOpenEditor || p.ShouldOpenShell) {
		return fmt.Errorf("worktree resolution failed: %w", err)
	}

	// Execute: up
	if p.ShouldStartContainer {
		if err := runner.Up(action.UpOptions{
			ContainerName: containerName,
			ImageName:     imageName,
			WorktreePath:  worktreePath,
			Rebuild:       p.Rebuild,
			NoBuild:       p.NoBuild,
			SSHMode:       p.SSHMode,
		}); err != nil {
			return fmt.Errorf("up failed: %w", err)
		}
	}

	// Execute: open
	if p.ShouldOpenEditor {
		launcher, err := editor.NewLauncher(ed)
		if err != nil {
			return fmt.Errorf("editor launcher creation failed: %w", err)
		}
		if _, err := runner.Open(launcher, editor.LaunchOptions{
			WorktreePath:    worktreePath,
			ContainerName:   containerName,
			NewWindow:       true,
			TryDevcontainer: p.TryDevcontainerAttach,
			Logger:          logger,
			DryRun:          flagDryRun,
		}); err != nil {
			return fmt.Errorf("open failed: %w", err)
		}
	}

	// Execute: down
	if p.ShouldStopContainer {
		if err := runner.Down(containerName); err != nil {
			return fmt.Errorf("down failed: %w", err)
		}
	}

	// Execute: status
	if p.ShouldShowStatus {
		runner.PrintStatus(feature, containerName, worktreePath)
	}

	// Execute: shell
	if p.ShouldOpenShell {
		if err := runner.Shell(containerName); err != nil {
			return fmt.Errorf("shell failed: %w", err)
		}
	}

	// Execute: exec
	if len(p.ExecCommand) > 0 {
		if err := runner.Exec(containerName, p.ExecCommand); err != nil {
			return fmt.Errorf("exec failed: %w", err)
		}
	}

	return nil
}
