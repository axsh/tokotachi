package editor

import (
	"fmt"
)

const (
	envKeyAG     = "TT_CMD_AG"
	defaultCmdAG = "antigravity"
)

// AG implements Launcher for the Antigravity editor.
type AG struct{}

// Name returns the editor identifier.
func (a *AG) Name() string { return "ag" }

// Launch opens Antigravity with the local worktree.
// Dev Container attach is never attempted (L4: unsupported).
func (a *AG) Launch(opts LaunchOptions) (LaunchResult, error) {
	cmd := ResolveCommand(envKeyAG, defaultCmdAG)

	if opts.DryRun {
		opts.Logger.Info("[DRY-RUN] %s %s", cmd, opts.WorktreePath)
		return LaunchResult{Method: "local", EditorCmd: cmd}, nil
	}

	if err := opts.CmdRunner.RunInteractive(cmd, opts.WorktreePath); err != nil {
		return LaunchResult{}, fmt.Errorf("failed to open Antigravity: %w", err)
	}
	return LaunchResult{Method: "local", EditorCmd: cmd}, nil
}
