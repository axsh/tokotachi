package action

import (
	"github.com/axsh/tokotachi/features/devctl/internal/cmdexec"
	"github.com/axsh/tokotachi/features/devctl/internal/log"
)

// Runner executes container and editor actions via cmdexec.
type Runner struct {
	Logger    *log.Logger
	DryRun    bool
	CmdRunner *cmdexec.Runner
}

// DockerRun executes "docker <args...>" interactively.
func (r *Runner) DockerRun(args ...string) error {
	return r.CmdRunner.RunInteractive("docker", args...)
}

// DockerRunOutput executes "docker <args...>" and returns stdout.
func (r *Runner) DockerRunOutput(args ...string) (string, error) {
	return r.CmdRunner.Run("docker", args...)
}

// DockerRunCheck executes a Docker command for condition checking.
// Failures are logged at DEBUG level with [SKIP] label (not ERROR).
func (r *Runner) DockerRunCheck(args ...string) error {
	return r.CmdRunner.RunInteractiveWithOpts(cmdexec.CheckOpt(), "docker", args...)
}

// DockerRunOutputCheck is like DockerRunOutput but for condition checks.
// Failures are logged at DEBUG level with [SKIP] label (not ERROR).
func (r *Runner) DockerRunOutputCheck(args ...string) (string, error) {
	return r.CmdRunner.RunWithOpts(cmdexec.CheckOpt(), "docker", args...)
}

// DockerRunTolerated executes a Docker command where failure is acceptable.
// Failures are logged at WARN level.
func (r *Runner) DockerRunTolerated(args ...string) error {
	return r.CmdRunner.RunInteractiveWithOpts(cmdexec.ToleratedOpt(), "docker", args...)
}
