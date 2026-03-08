package action

import (
	"fmt"
	"strings"
)

// FeatureState represents the state of a feature environment.
type FeatureState string

const (
	StateNotFound         FeatureState = "NOT_FOUND"
	StateWorktreeOnly     FeatureState = "WORKTREE_ONLY"
	StateContainerRunning FeatureState = "CONTAINER_RUNNING"
	StateContainerStopped FeatureState = "CONTAINER_STOPPED"
)

// Status checks the state of a feature's container.
func (r *Runner) Status(containerName, worktreePath string) FeatureState {
	out, err := r.DockerRunOutputCheck("inspect", "--format", "{{.State.Running}}", containerName)
	if err != nil {
		if worktreePath != "" {
			return StateWorktreeOnly
		}
		return StateNotFound
	}
	if strings.TrimSpace(out) == "true" {
		return StateContainerRunning
	}
	return StateContainerStopped
}

// PrintStatus displays the feature status to the user.
func (r *Runner) PrintStatus(feature, containerName, worktreePath string) {
	state := r.Status(containerName, worktreePath)
	r.Logger.Info("Feature: %s", feature)
	r.Logger.Info("Container: %s", containerName)
	r.Logger.Info("Worktree: %s", worktreePath)
	r.Logger.Info("State: %s", state)
	switch state {
	case StateContainerRunning:
		fmt.Println("✅ Container is running")
	case StateContainerStopped:
		fmt.Println("⏸  Container is stopped")
	case StateWorktreeOnly:
		fmt.Println("📁 Worktree exists, no container")
	case StateNotFound:
		fmt.Println("❌ Not found")
	}
}
