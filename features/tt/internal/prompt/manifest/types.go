package manifest

import "fmt"

// ValidKinds contains the list of valid entity kinds
var ValidKinds = map[string]bool{
	"policy":     true,
	"procedure":  true,
	"capability": true,
	"guard":      true,
	"worker":     true,
	"bundle":     true,
	"target":     true,
	"skip":       true,
}

// Entity represents an entity read from a manifest YAML file
type Entity struct {
	APIVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	ID         string         `yaml:"id"`
	Title      string         `yaml:"title"`
	FilePath   string         `yaml:"-"`
	Raw        map[string]any `yaml:"-"`
}

// ValidateKind validates whether the kind field has a valid value
func (e *Entity) ValidateKind() error {
	if e.Kind == "" {
		return fmt.Errorf("kind is empty")
	}
	if !ValidKinds[e.Kind] {
		return fmt.Errorf("unknown kind: %s", e.Kind)
	}
	return nil
}

// MemoryDoc represents a memory document read from frontmatter-annotated Markdown
type MemoryDoc struct {
	ID           string   `yaml:"id"`
	Kind         string   `yaml:"kind"`
	Title        string   `yaml:"title"`
	Status       string   `yaml:"status"`
	Topics       []string `yaml:"topics"`
	Triggers     []string `yaml:"triggers"`
	DependsOn    []string `yaml:"depends_on"`
	Evidence     []string `yaml:"evidence"`
	LastReviewed string   `yaml:"last_reviewed"`
	FilePath     string   `yaml:"-"`
}

// ValidStatuses are valid status values for memory documents
var ValidStatuses = map[string]bool{
	"current":      true,
	"target":       true,
	"transitional": true,
	"question":     true,
	"deprecated":   true,
}

// ValidateStatus validates the status field
func (d *MemoryDoc) ValidateStatus() error {
	if d.Status == "" {
		return fmt.Errorf("status is empty")
	}
	if !ValidStatuses[d.Status] {
		return fmt.Errorf("unknown status: %s", d.Status)
	}
	return nil
}

// ProjectConfig represents the configuration from project.yaml
type ProjectConfig struct {
	Version   int               `yaml:"version"`
	ProjectID string            `yaml:"project_id"`
	Sources   map[string]string `yaml:"sources"`
	Outputs   OutputConfig      `yaml:"outputs"`
	Defaults  DefaultConfig     `yaml:"defaults"`
}

// OutputConfig holds output path configuration
type OutputConfig struct {
	ResolvedManifest  string `yaml:"resolved_manifest"`
}

// DefaultConfig holds default configuration values
type DefaultConfig struct {
	Language        string `yaml:"language"`
	GeneratedBanner bool   `yaml:"generated_banner"`
	BuildDir        string `yaml:"build_dir"`
}

// ValidationError represents a validation error
type ValidationError struct {
	File    string
	Line    int
	Message string
}

// Error returns the error message for a ValidationError
func (e ValidationError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d: ERROR: %s", e.File, e.Line, e.Message)
	}
	return fmt.Sprintf("%s: ERROR: %s", e.File, e.Message)
}
