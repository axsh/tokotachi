package action

import (
	"os"
	"os/exec"
)

// Shell opens an interactive shell in the container.
func (r *Runner) Shell(containerName string) error {
	r.Logger.Info("Opening shell in %s...", containerName)
	if r.DryRun {
		r.Logger.Info("[DRY-RUN] docker exec -it %s bash", containerName)
		return nil
	}
	cmd := exec.Command("docker", "exec", "-it", containerName, "bash")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
