package action

// UpOptions holds parameters for the up action.
type UpOptions struct {
	ContainerName string
	ImageName     string
	WorktreePath  string
	Rebuild       bool
	NoBuild       bool
	SSHMode       bool
	Env           map[string]string
}

// Up starts the development container.
func (r *Runner) Up(opts UpOptions) error {
	// Step 1: Check if container already exists and is running
	state := r.Status(opts.ContainerName, opts.WorktreePath)
	if state == StateContainerRunning {
		r.Logger.Info("Container %s is already running", opts.ContainerName)
		return nil
	}

	// Step 2: Build image if needed
	if !opts.NoBuild {
		if opts.Rebuild || !r.imageExists(opts.ImageName) {
			r.Logger.Info("Building image %s...", opts.ImageName)
			if err := r.buildImage(opts); err != nil {
				return err
			}
		}
	}

	// Step 3: Remove stopped container if exists
	if state == StateContainerStopped {
		r.Logger.Info("Removing stopped container %s...", opts.ContainerName)
		_ = r.DockerRun("rm", opts.ContainerName)
	}

	// Step 4: Run container
	args := []string{
		"run", "-d",
		"--name", opts.ContainerName,
		"-v", opts.WorktreePath + ":/workspace",
		"-w", "/workspace",
	}

	// Add environment variables
	for k, v := range opts.Env {
		args = append(args, "-e", k+"="+v)
	}

	// SSH mode
	if opts.SSHMode {
		args = append(args, "-e", "ENABLE_SSH=1")
	}

	args = append(args, opts.ImageName)

	r.Logger.Info("Starting container %s...", opts.ContainerName)
	if err := r.DockerRun(args...); err != nil {
		return err
	}

	r.Logger.Info("Container %s started successfully", opts.ContainerName)
	return nil
}

func (r *Runner) imageExists(imageName string) bool {
	_, err := r.DockerRunOutput("image", "inspect", imageName)
	return err == nil
}

func (r *Runner) buildImage(opts UpOptions) error {
	return r.DockerRun("build", "-t", opts.ImageName, opts.WorktreePath)
}
