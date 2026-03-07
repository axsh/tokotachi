package editor

import (
	"fmt"
	"os"
	"os/exec"
)

const (
	envKeyClaude     = "DEVCTL_CMD_CLAUDE"
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

	run := exec.Command(cmd)
	run.Dir = opts.WorktreePath
	run.Stdin = os.Stdin
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	if err := run.Start(); err != nil {
		return LaunchResult{}, fmt.Errorf("failed to start Claude Code: %w", err)
	}
	opts.Logger.Info("Claude Code started (pid=%d)", run.Process.Pid)
	return LaunchResult{Method: "cli", EditorCmd: cmd}, nil
}
