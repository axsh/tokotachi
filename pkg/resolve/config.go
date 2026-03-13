package resolve

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FeatureConfig represents feature-level dev settings.
type FeatureConfig struct {
	Dev struct {
		EditorDefault string `yaml:"editor_default"`
		SSHSupported  bool   `yaml:"ssh_supported"`
		Shell         string `yaml:"shell"`
	} `yaml:"dev"`
}

// LoadFeatureConfig loads feature.yaml from the feature directory.
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
