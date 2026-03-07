package resolve

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// DevcontainerConfig represents the parsed devcontainer.json.
type DevcontainerConfig struct {
	Name            string            `json:"name"`
	Image           string            `json:"image"`
	Build           DevcontainerBuild `json:"build"`
	WorkspaceFolder string            `json:"workspaceFolder"`
	ContainerEnv    map[string]string `json:"containerEnv"`
	Mounts          []string          `json:"mounts"`
	RemoteUser      string            `json:"remoteUser"`
	configDir       string            // directory containing devcontainer.json
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

// ConfigDir returns the directory containing devcontainer.json.
// Used to resolve relative paths in build.dockerfile and build.context.
func (c DevcontainerConfig) ConfigDir() string {
	return c.configDir
}

// loadFromJSON reads and parses a devcontainer.json file.
// Sets configDir to the directory containing the file.
func loadFromJSON(jsonPath string) (DevcontainerConfig, error) {
	var cfg DevcontainerConfig
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	cfg.configDir = filepath.Dir(jsonPath)
	return cfg, nil
}

// LoadDevcontainerConfig loads devcontainer configuration.
// Search priority:
//  1. features/<feature>/.devcontainer/devcontainer.json
//  2. work/<feature>/<branch>/.devcontainer/devcontainer.json
//  3. work/<feature>/.devcontainer/devcontainer.json (backward compat)
//  4. work/<feature>/<branch>/.devcontainer/Dockerfile (fallback)
//  5. work/<feature>/<branch>/Dockerfile (fallback)
func LoadDevcontainerConfig(repoRoot, feature, branch string) (DevcontainerConfig, error) {
	// Priority 1: features/<feature>/.devcontainer/devcontainer.json
	featureJSON := filepath.Join(repoRoot, "features", feature, ".devcontainer", "devcontainer.json")
	if cfg, err := loadFromJSON(featureJSON); err == nil {
		return cfg, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return DevcontainerConfig{}, err
	}

	// Priority 2: work/<feature>/<branch>/.devcontainer/devcontainer.json
	branchJSON := filepath.Join(repoRoot, "work", feature, branch, ".devcontainer", "devcontainer.json")
	if cfg, err := loadFromJSON(branchJSON); err == nil {
		return cfg, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return DevcontainerConfig{}, err
	}

	// Priority 3: work/<feature>/.devcontainer/devcontainer.json (backward compat)
	featureOnlyJSON := filepath.Join(repoRoot, "work", feature, ".devcontainer", "devcontainer.json")
	if cfg, err := loadFromJSON(featureOnlyJSON); err == nil {
		return cfg, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return DevcontainerConfig{}, err
	}

	// Fallback: look for Dockerfile directly
	var cfg DevcontainerConfig
	worktree := filepath.Join(repoRoot, "work", feature, branch)

	// Priority 4: .devcontainer/Dockerfile
	dcDockerfile := filepath.Join(worktree, ".devcontainer", "Dockerfile")
	if _, err := os.Stat(dcDockerfile); err == nil {
		cfg.Build.Dockerfile = dcDockerfile
		cfg.WorkspaceFolder = "/workspace"
		cfg.configDir = filepath.Dir(dcDockerfile)
		return cfg, nil
	}

	// Priority 5: Dockerfile in worktree root
	rootDockerfile := filepath.Join(worktree, "Dockerfile")
	if _, err := os.Stat(rootDockerfile); err == nil {
		cfg.Build.Dockerfile = rootDockerfile
		cfg.WorkspaceFolder = "/workspace"
		cfg.configDir = worktree
		return cfg, nil
	}

	return cfg, nil
}
