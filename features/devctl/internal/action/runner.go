package action

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/escape-dev/devctl/internal/log"
)

// Runner executes Docker commands.
type Runner struct {
	Logger *log.Logger
	DryRun bool
}

// DockerRun executes "docker <args...>".
// In dry-run mode, it only logs the command.
func (r *Runner) DockerRun(args ...string) error {
	r.Logger.Debug("docker %v", args)
	if r.DryRun {
		r.Logger.Info("[DRY-RUN] docker %v", args)
		return nil
	}
	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker %v: %w", args, err)
	}
	return nil
}

// DockerRunOutput executes "docker <args...>" and returns stdout.
func (r *Runner) DockerRunOutput(args ...string) (string, error) {
	r.Logger.Debug("docker %v", args)
	cmd := exec.Command("docker", args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker %v: %w", args, err)
	}
	return string(out), nil
}
