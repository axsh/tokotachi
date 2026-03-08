package action

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/resolve"
)

const containerStartGracePeriod = 2 * time.Second

// UpOptions holds parameters for the up action.
type UpOptions struct {
	ContainerName string
	ImageName     string
	WorktreePath  string
	FeaturePath   string // path to features/<feature>/ for Dockerfile
	Rebuild       bool
	NoBuild       bool
	SSHMode       bool
	Env           map[string]string
	// DevcontainerConfig fields
	WorkspaceFolder string                   // from devcontainer.json (default: "/workspace")
	Mounts          []string                 // from devcontainer.json
	ContainerEnv    map[string]string        // from devcontainer.json
	RemoteUser      string                   // from devcontainer.json
	DockerfilePath  string                   // resolved absolute path to Dockerfile
	BuildContext    string                   // resolved absolute path to build context
	GitWorktree     *resolve.GitWorktreeInfo // nil if not a worktree
	GitOverrideFile string                   // path to temp .git override file (for container mount)
}

// Up starts the development container.
func (r *Runner) Up(opts UpOptions) error {
	// Default workspace folder
	wsFolder := opts.WorkspaceFolder
	if wsFolder == "" {
		wsFolder = "/workspace"
	}

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
		"-v", opts.WorktreePath + ":" + wsFolder,
		"-w", wsFolder,
	}

	// Add git worktree mounts if detected (read-only to avoid corrupting host files)
	if opts.GitWorktree != nil && opts.GitWorktree.IsWorktree {
		r.Logger.Debug("Adding git worktree mounts: mainGitDir=%s, worktreeGitDir=%s",
			opts.GitWorktree.MainGitDir, opts.GitWorktree.WorktreeGitDir)
		args = append(args, "-v", opts.GitWorktree.MainGitDir+":/repo-git:ro")
		args = append(args, "-v", opts.GitWorktree.WorktreeGitDir+":/worktree-git-src:ro")
		// Override-mount the .git file so the host .git is never modified
		if opts.GitOverrideFile != "" {
			args = append(args, "-v", opts.GitOverrideFile+":"+wsFolder+"/.git")
		}
	}

	// Add mounts from devcontainer.json
	for _, m := range opts.Mounts {
		args = append(args, "--mount", m)
	}

	// Add containerEnv from devcontainer.json
	for k, v := range opts.ContainerEnv {
		args = append(args, "-e", k+"="+v)
	}

	// Add CLI environment variables
	for k, v := range opts.Env {
		args = append(args, "-e", k+"="+v)
	}

	// remoteUser
	if opts.RemoteUser != "" {
		args = append(args, "--user", opts.RemoteUser)
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

	// Step 5: Verify container is still running after grace period
	r.Logger.Debug("Waiting %s for container to stabilize...", containerStartGracePeriod)
	time.Sleep(containerStartGracePeriod)

	state = r.Status(opts.ContainerName, "")
	if state != StateContainerRunning {
		// Collect container logs for diagnosis
		logs, _ := r.DockerRunOutput("logs", "--tail", "20", opts.ContainerName)
		return fmt.Errorf(
			"container %s exited immediately after start. "+
				"Ensure the Dockerfile has a CMD that keeps the process running "+
				"(e.g. CMD [\"sleep\", \"infinity\"]).\n"+
				"Container logs:\n%s", opts.ContainerName, logs,
		)
	}

	// Step 6: Setup git worktree paths inside the container
	if opts.GitWorktree != nil && opts.GitWorktree.IsWorktree {
		r.Logger.Info("Setting up git worktree paths inside container...")
		if err := r.setupGitWorktree(opts.ContainerName, opts.GitWorktree, wsFolder); err != nil {
			r.Logger.Warn("Git worktree setup failed (git may not work inside container): %v", err)
		}
	}

	r.Logger.Info("Container %s started successfully", opts.ContainerName)
	return nil
}

// setupGitWorktree creates a writable copy of the read-only mounted worktree metadata
// inside the container, and rewrites the .git file to point to it.
// The original host files remain untouched because mounts are read-only.
func (r *Runner) setupGitWorktree(containerName string, info *resolve.GitWorktreeInfo, wsFolder string) error {
	// Step 1: Copy worktree metadata from read-only mount to writable location
	copyCmd := `cp -a /worktree-git-src /worktree-git`
	if err := r.DockerRun("exec", containerName, "sh", "-c", copyCmd); err != nil {
		return fmt.Errorf("failed to copy worktree metadata: %w", err)
	}

	// Step 2 (removed): .git file rewrite is no longer needed.
	// The .git file is now override-mounted via docker run -v,
	// so the host .git file is never modified.

	// Step 3: Rewrite commondir in the writable copy to point to /repo-git
	commondirCmd := `printf '/repo-git\n' > /worktree-git/commondir`
	if err := r.DockerRun("exec", containerName, "sh", "-c", commondirCmd); err != nil {
		return fmt.Errorf("failed to rewrite commondir: %w", err)
	}

	// Step 4: Rewrite gitdir (reverse reference) to container-internal path
	gitdirCmd := fmt.Sprintf(`printf '%s/.git\n' > /worktree-git/gitdir`, wsFolder)
	if err := r.DockerRun("exec", containerName, "sh", "-c", gitdirCmd); err != nil {
		return fmt.Errorf("failed to rewrite gitdir: %w", err)
	}

	r.Logger.Info("Git worktree paths configured successfully")
	return nil
}

func (r *Runner) imageExists(imageName string) bool {
	_, err := r.DockerRunOutputCheck("image", "inspect", imageName)
	return err == nil
}

func (r *Runner) buildImage(opts UpOptions) error {
	// Use resolved DockerfilePath and BuildContext if available
	if opts.DockerfilePath != "" {
		buildCtx := opts.BuildContext
		if buildCtx == "" {
			buildCtx = filepath.Dir(opts.DockerfilePath)
		}
		return r.DockerRun("build", "-f", opts.DockerfilePath, "-t", opts.ImageName, buildCtx)
	}

	// Fallback: auto-detect Dockerfile
	buildContext := opts.FeaturePath
	if buildContext == "" {
		buildContext = opts.WorktreePath
	}
	// Look for Dockerfile in .devcontainer/ first, then root
	devcontainerDir := filepath.Join(buildContext, ".devcontainer")
	if _, err := os.Stat(filepath.Join(devcontainerDir, "Dockerfile")); err == nil {
		buildContext = devcontainerDir
	}
	return r.DockerRun("build", "-t", opts.ImageName, buildContext)
}
