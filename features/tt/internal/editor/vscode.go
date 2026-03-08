package editor

import (
	"fmt"
)

const (
	envKeyCode     = "TT_CMD_CODE"
	defaultCmdCode = "code"
)

// VSCode implements Launcher for Visual Studio Code.
type VSCode struct{}

// Name returns the editor identifier.
func (v *VSCode) Name() string { return "code" }

// Launch opens VSCode. If TryDevcontainer is true, attempts Dev Container
// attach via CLI. Falls back to local folder open on failure.
func (v *VSCode) Launch(opts LaunchOptions) (LaunchResult, error) {
	cmd := ResolveCommand(envKeyCode, defaultCmdCode)

	if opts.DryRun {
		method := "local"
		if opts.TryDevcontainer {
			method = "devcontainer (dry-run)"
		}
		opts.Logger.Info("[DRY-RUN] %s %s (method: %s)", cmd, opts.WorktreePath, method)
		return LaunchResult{Method: method, EditorCmd: cmd}, nil
	}

	// Try devcontainer attach if capable
	if opts.TryDevcontainer && opts.ContainerName != "" {
		opts.Logger.Info("Attempting Dev Container attach for %s...", opts.ContainerName)
		uri := DevcontainerURI(opts.ContainerName, "")
		if err := opts.CmdRunner.RunInteractive(cmd, "--folder-uri", uri); err == nil {
			opts.Logger.Info("Dev Container attach succeeded")
			return LaunchResult{Method: "devcontainer", EditorCmd: cmd}, nil
		}
		opts.Logger.Warn("Dev Container attach failed, falling back to local open")
	}

	// Fallback: open local worktree
	args := []string{opts.WorktreePath}
	if opts.NewWindow {
		args = append([]string{"--new-window"}, args...)
	}
	if err := opts.CmdRunner.RunInteractive(cmd, args...); err != nil {
		return LaunchResult{}, fmt.Errorf("failed to open VSCode: %w", err)
	}
	fallback := opts.TryDevcontainer
	return LaunchResult{Method: "local", Fallback: fallback, EditorCmd: cmd}, nil
}
