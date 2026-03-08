package editor

import (
	"fmt"
)

const (
	envKeyClaude     = "TT_CMD_CLAUDE"
	defaultCmdClaude = "claude"
)

// Claude implements Launcher for Claude Code (CLI/agent).
type Claude struct{}

// Name returns the editor identifier.
func (c *Claude) Name() string { return "claude" }

// Launch starts Claude Code with the target worktree as working directory.
// Dev Container attach is not applicable.
func (c *Claude) Launch(opts LaunchOptions) (LaunchResult, error) {
	cmd := ResolveCommand(envKeyClaude, defaultCmdClaude)

	if opts.DryRun {
		opts.Logger.Info("[DRY-RUN] %s (cwd: %s)", cmd, opts.WorktreePath)
		return LaunchResult{Method: "cli", EditorCmd: cmd}, nil
	}

	// Claude Code is started with cwd, so we use RunInteractive
	// but we need to set dir. For now, use the worktree path as argument.
	if err := opts.CmdRunner.RunInteractive(cmd, opts.WorktreePath); err != nil {
		return LaunchResult{}, fmt.Errorf("failed to start Claude Code: %w", err)
	}
	return LaunchResult{Method: "cli", EditorCmd: cmd}, nil
}
