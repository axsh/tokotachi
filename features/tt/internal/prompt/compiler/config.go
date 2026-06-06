package compiler

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
)

// LoadConfig reads project.yaml and returns a ProjectConfig
func LoadConfig(path string) (*manifest.ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("project.yaml not found: %s: %w", path, err)
	}

	var cfg manifest.ProjectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse project.yaml: %s: %w", path, err)
	}

	if cfg.Version < 1 {
		return nil, fmt.Errorf("project.yaml: version must be >= 1, got %d", cfg.Version)
	}
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("project.yaml: project_id is required")
	}

	return &cfg, nil
}

// ResolveProjectRoot computes the project root from the project.yaml path
// Assumes project.yaml is at prompts/manifest/project.yaml, so root is 2 levels up
func ResolveProjectRoot(projectYAMLPath string) (string, error) {
	absPath, err := filepath.Abs(projectYAMLPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}
	// prompts/manifest/project.yaml -> prompts/manifest -> prompts -> root
	root := filepath.Dir(filepath.Dir(filepath.Dir(absPath)))
	return root, nil
}
