// Package tokotachi provides a high-level API for managing development environments.
//
// This package mirrors the functionality of the `tt` CLI tool,
// allowing programmatic control over worktrees, containers, and scaffolding.
//
// Usage:
//
//	client := tokotachi.NewClient("/path/to/repo")
//	err := client.Create("my-branch", tokotachi.CreateOptions{})
//	err = client.Up("my-branch", "my-feature", tokotachi.UpOptions{})
//	err = client.Scaffold("go", "web-api", tokotachi.ScaffoldOptions{})
package tokotachi

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/axsh/tokotachi/internal/cmdexec"
	"github.com/axsh/tokotachi/internal/log"
	"github.com/axsh/tokotachi/pkg/action"
	"github.com/axsh/tokotachi/pkg/detect"
	"github.com/axsh/tokotachi/pkg/editor"
	"github.com/axsh/tokotachi/pkg/matrix"
	"github.com/axsh/tokotachi/pkg/plan"
	"github.com/axsh/tokotachi/pkg/resolve"
	"github.com/axsh/tokotachi/pkg/scaffold"
	"github.com/axsh/tokotachi/pkg/state"
	"github.com/axsh/tokotachi/pkg/worktree"
)

// Client provides high-level operations for managing development environments.
// It wraps the lower-level packages in pkg/ to provide simple, command-level functions.
type Client struct {
	// RepoRoot is the root path of the repository.
	RepoRoot string

	// Verbose enables debug logging.
	Verbose bool

	// DryRun enables dry-run mode (actions are logged but not executed).
	DryRun bool

	// Stdout is the writer for standard output. Defaults to os.Stdout if nil.
	Stdout io.Writer

	// Stderr is the writer for standard error. Defaults to os.Stderr if nil.
	Stderr io.Writer

	// Stdin is the reader for standard input. Defaults to os.Stdin if nil.
	Stdin io.Reader
}

// NewClient creates a new Client with the given repository root.
func NewClient(repoRoot string) *Client {
	return &Client{
		RepoRoot: repoRoot,
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		Stdin:    os.Stdin,
	}
}

// newContext builds internal context objects from Client settings.
func (c *Client) newContext() (*log.Logger, *cmdexec.Runner, *action.Runner) {
	stderr := c.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	logger := log.New(stderr, c.Verbose)
	rec := cmdexec.NewRecorder()
	runner := &cmdexec.Runner{Logger: logger, DryRun: c.DryRun, Recorder: rec}
	actionRunner := &action.Runner{
		Logger:    logger,
		DryRun:    c.DryRun,
		CmdRunner: runner,
	}
	return logger, runner, actionRunner
}

// resolveProjectName loads the project name from .devrc.yaml, defaulting to "tt".
func (c *Client) resolveProjectName() string {
	globalCfg, _ := resolve.LoadGlobalConfig(c.RepoRoot)
	if globalCfg.ProjectName != "" {
		return globalCfg.ProjectName
	}
	return "tt"
}

// reservedBranchNames that cannot be used.
var reservedBranchNames = []string{"main", "master"}

// validateBranch checks if the branch name is reserved.
func validateBranch(branch string) error {
	for _, name := range reservedBranchNames {
		if branch == name {
			return fmt.Errorf("%q is a reserved branch name and cannot be used", branch)
		}
	}
	return nil
}

// CreateOptions configures the Create operation.
type CreateOptions struct{}

// Create creates a new git branch and worktree.
func (c *Client) Create(branch string, _ CreateOptions) error {
	if err := validateBranch(branch); err != nil {
		return err
	}

	logger, _, _ := c.newContext()
	wm := &worktree.Manager{CmdRunner: &cmdexec.Runner{Logger: logger, DryRun: c.DryRun, Recorder: cmdexec.NewRecorder()}, RepoRoot: c.RepoRoot}

	if wm.Exists(branch) {
		logger.Info("Worktree already exists for branch %s", branch)
		return nil
	}

	logger.Info("Creating worktree for branch %s...", branch)
	if err := wm.Create(branch); err != nil {
		return fmt.Errorf("worktree creation failed: %w", err)
	}
	return nil
}

// UpOptions configures the Up operation.
type UpOptions struct {
	// SSH enables SSH mode for the container.
	SSH bool

	// Rebuild forces rebuilding the container image.
	Rebuild bool

	// NoBuild skips image building.
	NoBuild bool
}

// Up starts the development container for the given branch and feature.
func (c *Client) Up(branch, feature string, opts UpOptions) error {
	if err := validateBranch(branch); err != nil {
		return err
	}
	if feature == "" {
		return fmt.Errorf("feature is required for 'up' operation")
	}

	logger, _, actionRunner := c.newContext()
	projectName := c.resolveProjectName()

	containerName := resolve.ContainerName(projectName, feature)
	imageName := resolve.ImageName(projectName, feature)

	rec := cmdexec.NewRecorder()
	runner := &cmdexec.Runner{Logger: logger, DryRun: c.DryRun, Recorder: rec}
	wm := &worktree.Manager{CmdRunner: runner, RepoRoot: c.RepoRoot}

	if !wm.Exists(branch) {
		return fmt.Errorf("worktree not found for branch %s. Use Create() first", branch)
	}

	// Build execution plan
	currentOS := detect.CurrentOS()
	globalCfg2, _ := resolve.LoadGlobalConfig(c.RepoRoot)
	containerMode := matrix.ContainerMode(globalCfg2.DefaultContainerMode)

	p := plan.Build(plan.Input{
		Feature:       feature,
		OS:            currentOS,
		ContainerMode: containerMode,
		Up:            true,
		SSH:           opts.SSH,
		Rebuild:       opts.Rebuild,
		NoBuild:       opts.NoBuild,
	})
	_ = p

	// Resolve worktree path
	worktreePath, err := resolve.Worktree(c.RepoRoot, branch)
	if err != nil {
		if c.DryRun {
			worktreePath = wm.Path(branch)
		} else {
			return fmt.Errorf("worktree resolution failed: %w", err)
		}
	}

	// Load devcontainer.json
	dcCfg, _ := resolve.LoadDevcontainerConfig(c.RepoRoot, feature, branch)

	upOpts := action.UpOptions{
		ContainerName: containerName,
		ImageName:     imageName,
		WorktreePath:  worktreePath,
		FeaturePath:   filepath.Join(c.RepoRoot, "features", feature),
		Rebuild:       opts.Rebuild,
		NoBuild:       opts.NoBuild,
		SSHMode:       opts.SSH,
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

	// Detect git worktree configuration
	gitInfo, gitErr := resolve.DetectGitWorktree(worktreePath)
	if gitErr == nil && gitInfo.IsWorktree {
		upOpts.GitWorktree = &gitInfo
		overrideFile, oErr := resolve.CreateContainerGitFile(os.TempDir())
		if oErr == nil {
			upOpts.GitOverrideFile = overrideFile
		}
	}

	if err := actionRunner.Up(upOpts); err != nil {
		return fmt.Errorf("up failed: %w", err)
	}

	// Save state
	statePath := state.StatePath(c.RepoRoot, branch)
	sf, _ := state.Load(statePath)
	if sf.Branch == "" {
		sf.Branch = branch
		sf.CreatedAt = time.Now()
	}
	sf.SetFeature(feature, state.FeatureState{
		Status:    state.StatusActive,
		StartedAt: time.Now(),
		Connectivity: state.Connectivity{
			Docker: state.DockerConnectivity{
				Enabled:       true,
				ContainerName: containerName,
				Devcontainer:  !dcCfg.IsEmpty(),
			},
			SSH: state.SSHConnectivity{Enabled: opts.SSH},
		},
	})
	if sf.CodeStatus == nil {
		sf.CodeStatus = &state.CodeStatus{
			Status: state.CodeStatusLocal,
		}
	}
	_ = state.Save(statePath, sf)

	return nil
}

// DownOptions configures the Down operation.
type DownOptions struct{}

// Down stops the development container for the given branch and feature.
func (c *Client) Down(branch, feature string, _ DownOptions) error {
	if err := validateBranch(branch); err != nil {
		return err
	}
	if feature == "" {
		return fmt.Errorf("feature is required for 'down' operation")
	}

	_, _, actionRunner := c.newContext()
	projectName := c.resolveProjectName()
	containerName := resolve.ContainerName(projectName, feature)

	if err := actionRunner.Down(containerName); err != nil {
		return fmt.Errorf("down failed: %w", err)
	}

	// Update state
	statePath := state.StatePath(c.RepoRoot, branch)
	if sf, err := state.Load(statePath); err == nil {
		if err := sf.UpdateFeatureStatus(feature, state.StatusStopped); err == nil {
			_ = state.Save(statePath, sf)
		}
	}

	return nil
}

// OpenOptions configures the Open operation.
type OpenOptions struct {
	// Editor specifies the editor to use (code|cursor|ag|claude).
	Editor string
}

// Open creates a worktree, starts the container (if feature specified),
// and opens the editor. Equivalent to: create -> up -> editor.
func (c *Client) Open(branch, feature string, opts OpenOptions) error {
	if err := validateBranch(branch); err != nil {
		return err
	}

	logger, _, actionRunner := c.newContext()
	projectName := c.resolveProjectName()

	rec := cmdexec.NewRecorder()
	runner := &cmdexec.Runner{Logger: logger, DryRun: c.DryRun, Recorder: rec}
	wm := &worktree.Manager{CmdRunner: runner, RepoRoot: c.RepoRoot}

	// Step 1: Create worktree if not exists
	if !wm.Exists(branch) {
		logger.Info("Worktree not found, creating %s...", wm.Path(branch))
		if err := wm.Create(branch); err != nil {
			return fmt.Errorf("worktree creation failed: %w", err)
		}
	}

	// Step 2: Start container if feature is specified
	if feature != "" {
		containerName := resolve.ContainerName(projectName, feature)
		imageName := resolve.ImageName(projectName, feature)

		worktreePath, wpErr := resolve.Worktree(c.RepoRoot, branch)
		if wpErr != nil {
			worktreePath = wm.Path(branch)
		}

		containerState := actionRunner.Status(containerName, worktreePath)
		if containerState != action.StateContainerRunning {
			logger.Info("Starting container...")

			upOpts := action.UpOptions{
				ContainerName: containerName,
				ImageName:     imageName,
				WorktreePath:  worktreePath,
				FeaturePath:   filepath.Join(c.RepoRoot, "features", feature),
			}

			dcCfg, _ := resolve.LoadDevcontainerConfig(c.RepoRoot, feature, branch)
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

			gitInfo, gitErr := resolve.DetectGitWorktree(worktreePath)
			if gitErr == nil && gitInfo.IsWorktree {
				upOpts.GitWorktree = &gitInfo
				overrideFile, oErr := resolve.CreateContainerGitFile(os.TempDir())
				if oErr == nil {
					upOpts.GitOverrideFile = overrideFile
				}
			}

			if err := actionRunner.Up(upOpts); err != nil {
				return fmt.Errorf("up failed: %w", err)
			}

			// Save state
			statePath := state.StatePath(c.RepoRoot, branch)
			sf, _ := state.Load(statePath)
			if sf.Branch == "" {
				sf.Branch = branch
				sf.CreatedAt = time.Now()
			}
			sf.SetFeature(feature, state.FeatureState{
				Status:    state.StatusActive,
				StartedAt: time.Now(),
				Connectivity: state.Connectivity{
					Docker: state.DockerConnectivity{
						Enabled:       true,
						ContainerName: containerName,
						Devcontainer:  !dcCfg.IsEmpty(),
					},
				},
			})
			if sf.CodeStatus == nil {
				sf.CodeStatus = &state.CodeStatus{
					Status: state.CodeStatusLocal,
				}
			}
			_ = state.Save(statePath, sf)
		}
	}

	// Step 3: Open editor
	worktreePath, err := resolve.Worktree(c.RepoRoot, branch)
	if err != nil {
		if c.DryRun {
			worktreePath = wm.Path(branch)
		} else {
			return fmt.Errorf("worktree resolution failed: %w", err)
		}
	}

	currentOS := detect.CurrentOS()
	editorName := detect.Editor(opts.Editor)
	if editorName == "" {
		editorName = detect.EditorCursor
	}
	globalCfg3, _ := resolve.LoadGlobalConfig(c.RepoRoot)
	containerMode := matrix.ContainerMode(globalCfg3.DefaultContainerMode)

	p := plan.Build(plan.Input{
		Feature:       feature,
		OS:            currentOS,
		Editor:        editorName,
		ContainerMode: containerMode,
		EditorOpen:    true,
	})

	launcher, err := editor.NewLauncher(editorName)
	if err != nil {
		return fmt.Errorf("editor launcher creation failed: %w", err)
	}

	var containerName string
	if feature != "" {
		containerName = resolve.ContainerName(projectName, feature)
	}
	tryDevcontainer := p.TryDevcontainerAttach && feature != ""

	if _, err := actionRunner.Open(launcher, editor.LaunchOptions{
		WorktreePath:    worktreePath,
		ContainerName:   containerName,
		NewWindow:       true,
		TryDevcontainer: tryDevcontainer,
		Logger:          logger,
		DryRun:          c.DryRun,
		CmdRunner:       runner,
	}); err != nil {
		return fmt.Errorf("open failed: %w", err)
	}

	return nil
}

// CloseOptions configures the Close operation.
type CloseOptions struct {
	// Yes skips confirmation prompts.
	Yes bool

	// Force forces branch deletion even if not merged.
	Force bool

	// Depth is the maximum depth for recursive worktree close.
	Depth int

	// Verbose shows all pending changes without truncation.
	Verbose bool
}

// Close stops containers and deletes the worktree.
func (c *Client) Close(branch string, opts CloseOptions) error {
	if err := validateBranch(branch); err != nil {
		return err
	}

	logger, _, actionRunner := c.newContext()
	projectName := c.resolveProjectName()

	rec := cmdexec.NewRecorder()
	runner := &cmdexec.Runner{Logger: logger, DryRun: c.DryRun, Recorder: rec}
	wm := &worktree.Manager{CmdRunner: runner, RepoRoot: c.RepoRoot}

	stdin := c.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}

	depth := opts.Depth
	if depth == 0 {
		depth = 10
	}

	if err := actionRunner.Close(action.CloseOptions{
		Branch:      branch,
		Force:       opts.Force,
		RepoRoot:    c.RepoRoot,
		ProjectName: projectName,
		Depth:       depth,
		Yes:         opts.Yes,
		Verbose:     opts.Verbose,
		Stdin:       stdin,
	}, wm); err != nil {
		return fmt.Errorf("close failed: %w", err)
	}
	return nil
}

// ScaffoldOptions configures the Scaffold operation.
type ScaffoldOptions struct {
	// RepoURL overrides the default template repository URL.
	RepoURL string

	// DryRun shows planned actions without executing.
	DryRun bool

	// Yes skips confirmation prompts.
	Yes bool

	// Lang specifies locale for template localization (e.g. "ja", "en").
	Lang string

	// Values provides key=value overrides for template options.
	Values []string

	// UseDefaults uses default values for non-required options.
	UseDefaults bool

	// SkipDeps skips dependency resolution.
	SkipDeps bool

	// Force forces re-download of all scaffolds.
	Force bool
}

// Scaffold generates project structure from templates.
func (c *Client) Scaffold(category, name string, opts ScaffoldOptions) error {
	stderr := c.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	stdout := c.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stdin := c.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}

	logger := log.New(stderr, c.Verbose)

	overrides, err := scaffold.ParseOptionOverrides(opts.Values)
	if err != nil {
		return err
	}

	var pattern []string
	if category != "" && name != "" {
		pattern = []string{category, name}
	} else if category != "" {
		pattern = []string{category}
	}

	dryRun := opts.DryRun || c.DryRun

	runOpts := scaffold.RunOptions{
		Pattern:         pattern,
		RepoURL:         opts.RepoURL,
		RepoRoot:        c.RepoRoot,
		DryRun:          dryRun,
		Yes:             opts.Yes,
		Lang:            opts.Lang,
		Logger:          logger,
		Stdout:          stdout,
		Stdin:           stdin,
		OptionOverrides: overrides,
		UseDefaults:     opts.UseDefaults,
		SkipDeps:        opts.SkipDeps,
		Force:           opts.Force,
	}

	p, err := scaffold.Run(runOpts)
	if err != nil {
		return err
	}

	if p == nil {
		return nil
	}

	if len(p.Warnings) > 0 {
		return fmt.Errorf("cannot proceed due to conflicts")
	}

	if dryRun {
		scaffold.PrintPlan(p, stdout)
		return nil
	}

	if !opts.Yes {
		scaffold.PrintPlan(p, stdout)
	}

	return scaffold.Apply(p, runOpts)
}

// StatusResult holds the result of a Status query.
type StatusResult struct {
	// Branch is the branch name.
	Branch string

	// WorktreeExists indicates if the worktree exists.
	WorktreeExists bool

	// WorktreePath is the path to the worktree directory.
	WorktreePath string

	// Features holds the feature states if available.
	Features map[string]state.FeatureState
}

// StatusOptions configures the Status operation.
type StatusOptions struct{}

// Status returns the worktree and container status for the given branch.
func (c *Client) Status(branch string, _ StatusOptions) (*StatusResult, error) {
	if err := validateBranch(branch); err != nil {
		return nil, err
	}

	logger, _, _ := c.newContext()
	rec := cmdexec.NewRecorder()
	runner := &cmdexec.Runner{Logger: logger, DryRun: c.DryRun, Recorder: rec}
	wm := &worktree.Manager{CmdRunner: runner, RepoRoot: c.RepoRoot}

	result := &StatusResult{
		Branch:         branch,
		WorktreeExists: wm.Exists(branch),
		WorktreePath:   wm.Path(branch),
	}

	statePath := state.StatePath(c.RepoRoot, branch)
	if sf, err := state.Load(statePath); err == nil {
		result.Features = sf.Features
	}

	return result, nil
}

// ListEntry represents a single worktree entry in the listing.
type ListEntry struct {
	// Branch is the branch name.
	Branch string

	// Path is the worktree path.
	Path string

	// Bare indicates if this is the bare repository entry.
	Bare bool
}

// ListOptions configures the List operation.
type ListOptions struct{}

// List returns all worktree branches.
func (c *Client) List(_ ListOptions) ([]ListEntry, error) {
	logger, _, _ := c.newContext()
	rec := cmdexec.NewRecorder()
	runner := &cmdexec.Runner{Logger: logger, DryRun: c.DryRun, Recorder: rec}

	gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")
	output, err := runner.Run(gitCmd, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Parse porcelain output
	var entries []ListEntry
	var current ListEntry
	for _, line := range splitLines(output) {
		switch {
		case len(line) > 9 && line[:9] == "worktree ":
			if current.Path != "" {
				entries = append(entries, current)
			}
			current = ListEntry{Path: line[9:]}
		case len(line) > 7 && line[:7] == "branch ":
			// Extract branch name from "refs/heads/xxx"
			ref := line[7:]
			if len(ref) > 11 && ref[:11] == "refs/heads/" {
				current.Branch = ref[11:]
			} else {
				current.Branch = ref
			}
		case line == "bare":
			current.Bare = true
		}
	}
	if current.Path != "" {
		entries = append(entries, current)
	}

	return entries, nil
}

// splitLines splits a string into lines.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := range len(s) {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
