package action

// Down stops and removes the development container.
func (r *Runner) Down(containerName string) error {
	r.Logger.Info("Stopping container %s...", containerName)
	if err := r.DockerRunTolerated("stop", containerName); err != nil {
		r.Logger.Warn("Stop failed (may already be stopped): %v", err)
	}

	r.Logger.Info("Removing container %s...", containerName)
	if err := r.DockerRunTolerated("rm", containerName); err != nil {
		return err
	}

	r.Logger.Info("Container %s removed successfully", containerName)
	return nil
}
