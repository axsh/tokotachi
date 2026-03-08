package editor

import (
	"fmt"
)

const (
	envKeyCursor     = "DEVCTL_CMD_CURSOR"
	defaultCmdCursor = "cursor"
)

// Cursor implements Launcher for the Cursor editor.
type Cursor struct{}

// Name returns the editor identifier.
func (c *Cursor) Name() string { return "cursor" }

// Launch opens Cursor. Same logic as VSCode but using the "cursor" command.
func (c *Cursor) Launch(opts LaunchOptions) (LaunchResult, error) {
	cmd := ResolveCommand(envKeyCursor, defaultCmdCursor)

	if opts.DryRun {
		method := "local"
		if opts.TryDevcontainer {
			method = "devcontainer (dry-run)"
		}
		opts.Logger.Info("[DRY-RUN] %s %s (method: %s)", cmd, opts.WorktreePath, method)
		return LaunchResult{Method: method, EditorCmd: cmd}, nil
	}

	if opts.TryDevcontainer && opts.ContainerName != "" {
		opts.Logger.Info("Attempting Dev Container attach for %s...", opts.ContainerName)
		uri := DevcontainerURI(opts.ContainerName, "")
		if err := opts.CmdRunner.RunInteractive(cmd, "--folder-uri", uri); err == nil {
			opts.Logger.Info("Dev Container attach succeeded")
			return LaunchResult{Method: "devcontainer", EditorCmd: cmd}, nil
		}
		opts.Logger.Warn("Dev Container attach failed, falling back to local open")
	}

	args := []string{opts.WorktreePath}
	if opts.NewWindow {
		args = append([]string{"--new-window"}, args...)
	}
	if err := opts.CmdRunner.RunInteractive(cmd, args...); err != nil {
		return LaunchResult{}, fmt.Errorf("failed to open Cursor: %w", err)
	}
	fallback := opts.TryDevcontainer
	return LaunchResult{Method: "local", Fallback: fallback, EditorCmd: cmd}, nil
}
