package resolve

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// GlobalConfig represents .devrc.yaml at the repo root.
type GlobalConfig struct {
	DefaultEditor        string `yaml:"default_editor"`
	ProjectName          string `yaml:"project_name"`
	WorkDir              string `yaml:"work_dir"`
	DefaultContainerMode string `yaml:"default_container_mode"`
}

// FeatureConfig represents feature-level dev settings.
type FeatureConfig struct {
	Dev struct {
		EditorDefault string `yaml:"editor_default"`
		SSHSupported  bool   `yaml:"ssh_supported"`
		Shell         string `yaml:"shell"`
	} `yaml:"dev"`
}

// LoadGlobalConfig loads .devrc.yaml from repoRoot.
// Returns sensible defaults if the file does not exist.
func LoadGlobalConfig(repoRoot string) (GlobalConfig, error) {
	var cfg GlobalConfig
	path := filepath.Join(repoRoot, ".devrc.yaml")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		cfg.DefaultEditor = "cursor"
		cfg.WorkDir = "work"
		cfg.DefaultContainerMode = "docker-local"
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.WorkDir == "" {
		cfg.WorkDir = "work"
	}
	if cfg.DefaultEditor == "" {
		cfg.DefaultEditor = "cursor"
	}
	if cfg.DefaultContainerMode == "" {
		cfg.DefaultContainerMode = "docker-local"
	}
	return cfg, nil
}

// LoadFeatureConfig loads feature.yaml from the feature directory.
// Searches work/<feature>/feature.yaml then features/<feature>/feature.yaml.
func LoadFeatureConfig(repoRoot, feature string) (FeatureConfig, error) {
	var cfg FeatureConfig
	candidates := []string{
		filepath.Join(repoRoot, "work", feature, "feature.yaml"),
		filepath.Join(repoRoot, "features", feature, "feature.yaml"),
	}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return cfg, err
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, err
		}
		return cfg, nil
	}
	return cfg, nil
}
