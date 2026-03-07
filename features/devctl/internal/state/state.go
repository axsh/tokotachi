package state

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Status represents the worktree lifecycle status.
type Status string

const (
	StatusActive  Status = "active"
	StatusStopped Status = "stopped"
	StatusClosed  Status = "closed"
)

// StateFile represents the worktree state YAML file.
type StateFile struct {
	Feature       string    `yaml:"feature"`
	Branch        string    `yaml:"branch"`
	CreatedAt     time.Time `yaml:"created_at"`
	ContainerMode string    `yaml:"container_mode"`
	Editor        string    `yaml:"editor"`
	Status        Status    `yaml:"status"`
}

// StatePath returns the state file path: work/<feature>/<branch>.state.yaml
func StatePath(repoRoot, feature, branch string) string {
	return filepath.Join(repoRoot, "work", feature, branch+".state.yaml")
}

// Load reads a state file from disk.
func Load(path string) (StateFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return StateFile{}, fmt.Errorf("failed to read state file: %w", err)
	}
	var s StateFile
	if err := yaml.Unmarshal(data, &s); err != nil {
		return StateFile{}, fmt.Errorf("failed to parse state file: %w", err)
	}
	return s, nil
}

// Save writes a state file to disk.
func Save(path string, s StateFile) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to marshal state file: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}
	return nil
}

// Remove deletes a state file. Returns nil if the file does not exist.
func Remove(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove state file: %w", err)
	}
	return nil
}
