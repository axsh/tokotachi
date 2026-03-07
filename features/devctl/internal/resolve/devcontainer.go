package resolve

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// DevcontainerConfig represents the minimal subset of devcontainer.json.
type DevcontainerConfig struct {
	Image           string            `json:"image"`
	Build           DevcontainerBuild `json:"build"`
	WorkspaceFolder string            `json:"workspaceFolder"`
	ContainerEnv    map[string]string `json:"containerEnv"`
}

// DevcontainerBuild represents the "build" field of devcontainer.json.
type DevcontainerBuild struct {
	Dockerfile string `json:"dockerfile"`
	Context    string `json:"context"`
}

// IsEmpty returns true if no configuration was loaded.
func (c DevcontainerConfig) IsEmpty() bool {
	return c.Image == "" && c.Build.Dockerfile == "" && c.WorkspaceFolder == ""
}

// HasDockerfile returns true if a Dockerfile-based build is configured.
func (c DevcontainerConfig) HasDockerfile() bool {
	return c.Build.Dockerfile != ""
}

// LoadDevcontainerConfig loads devcontainer configuration for the given feature.
// Search priority:
//  1. work/<feature>/.devcontainer/devcontainer.json
//  2. work/<feature>/.devcontainer/Dockerfile
//  3. work/<feature>/Dockerfile
func LoadDevcontainerConfig(repoRoot, feature string) (DevcontainerConfig, error) {
	var cfg DevcontainerConfig
	worktree := filepath.Join(repoRoot, "work", feature)

	// Priority 1: devcontainer.json
	jsonPath := filepath.Join(worktree, ".devcontainer", "devcontainer.json")
	data, err := os.ReadFile(jsonPath)
	if err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return cfg, err
		}
		return cfg, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return cfg, err
	}

	// Priority 2: .devcontainer/Dockerfile
	dcDockerfile := filepath.Join(worktree, ".devcontainer", "Dockerfile")
	if _, err := os.Stat(dcDockerfile); err == nil {
		cfg.Build.Dockerfile = dcDockerfile
		cfg.WorkspaceFolder = "/workspace"
		return cfg, nil
	}

	// Priority 3: Dockerfile in worktree root
	rootDockerfile := filepath.Join(worktree, "Dockerfile")
	if _, err := os.Stat(rootDockerfile); err == nil {
		cfg.Build.Dockerfile = rootDockerfile
		cfg.WorkspaceFolder = "/workspace"
		return cfg, nil
	}

	return cfg, nil
}
