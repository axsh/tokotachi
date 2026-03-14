package state

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// DockerConnectivity holds Docker container connection information.
type DockerConnectivity struct {
	Enabled       bool   `yaml:"enabled"`
	ContainerName string `yaml:"container_name"`
	Devcontainer  bool   `yaml:"devcontainer"`
}

// SSHConnectivity holds SSH connection information.
type SSHConnectivity struct {
	Enabled  bool   `yaml:"enabled"`
	Endpoint string `yaml:"endpoint,omitempty"`
}

// Connectivity groups all connection methods for a feature.
type Connectivity struct {
	Docker DockerConnectivity `yaml:"docker"`
	SSH    SSHConnectivity    `yaml:"ssh"`
}

// FeatureState represents the state of a single feature within a branch.
type FeatureState struct {
	Status       Status       `yaml:"status"`
	StartedAt    time.Time    `yaml:"started_at"`
	Connectivity Connectivity `yaml:"connectivity"`
}

// CodeStatusType represents the code hosting status.
type CodeStatusType string

const (
	CodeStatusLocal   CodeStatusType = "local"
	CodeStatusHosted  CodeStatusType = "hosted"
	CodeStatusPR      CodeStatusType = "pr"
	CodeStatusDeleted CodeStatusType = "deleted"
)

// CodeStatus holds the code hosting service status for a branch.
type CodeStatus struct {
	Status        CodeStatusType `yaml:"status"`
	PRCreatedAt   *time.Time     `yaml:"pr_created_at,omitempty"`
	LastCheckedAt *time.Time     `yaml:"last_checked_at,omitempty"`
}

// StateFile represents the branch-level state YAML file.
// Features map holds per-feature state entries.
type StateFile struct {
	Branch     string                  `yaml:"branch"`
	BaseBranch string                  `yaml:"base_branch,omitempty"`
	CreatedAt  time.Time               `yaml:"created_at"`
	Features   map[string]FeatureState `yaml:"features,omitempty"`
	CodeStatus *CodeStatus             `yaml:"code_status,omitempty"`
}

// StatePath returns the state file path: work/<branch>.state.yaml
func StatePath(repoRoot, branch string) string {
	return filepath.Join(repoRoot, "work", branch+".state.yaml")
}

// SetFeature adds or overwrites a feature entry in the state file.
func (s *StateFile) SetFeature(feature string, fs FeatureState) {
	if s.Features == nil {
		s.Features = make(map[string]FeatureState)
	}
	s.Features[feature] = fs
}

// UpdateFeatureStatus updates only the Status field of an existing feature,
// preserving Connectivity and other fields.
// Returns an error if the feature is not found.
func (s *StateFile) UpdateFeatureStatus(feature string, status Status) error {
	fs, ok := s.Features[feature]
	if !ok {
		return fmt.Errorf("feature %q not found in state", feature)
	}
	fs.Status = status
	s.Features[feature] = fs
	return nil
}

// RemoveFeature deletes a feature entry from the state file.
func (s *StateFile) RemoveFeature(feature string) {
	delete(s.Features, feature)
}

// HasActiveFeatures returns true if at least one feature has active status.
func (s *StateFile) HasActiveFeatures() bool {
	for _, fs := range s.Features {
		if fs.Status == StatusActive {
			return true
		}
	}
	return false
}

// ActiveFeatureNames returns feature names with active status.
func (s *StateFile) ActiveFeatureNames() []string {
	var names []string
	for name, fs := range s.Features {
		if fs.Status == StatusActive {
			names = append(names, name)
		}
	}
	return names
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

// ScanStateFiles finds all state files under work/ directory.
// Returns a map of branch name -> StateFile.
func ScanStateFiles(repoRoot string) (map[string]StateFile, error) {
	pattern := filepath.Join(repoRoot, "work", "*.state.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob state files: %w", err)
	}

	result := make(map[string]StateFile, len(matches))
	for _, path := range matches {
		branch := strings.TrimSuffix(filepath.Base(path), ".state.yaml")
		sf, err := Load(path)
		if err != nil {
			// Skip corrupted files instead of failing entirely
			continue
		}
		result[branch] = sf
	}
	return result, nil
}
