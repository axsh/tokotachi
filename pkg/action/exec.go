package action

import (
	"os"
	"os/exec"
)

// Exec runs a command in the container and returns its exit code.
func (r *Runner) Exec(containerName string, command []string) error {
	r.Logger.Info("Executing in %s: %v", containerName, command)
	if r.DryRun {
		r.Logger.Info("[DRY-RUN] docker exec %s %v", containerName, command)
		return nil
	}
	args := append([]string{"exec", containerName}, command...)
	cmd := exec.Command("docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
