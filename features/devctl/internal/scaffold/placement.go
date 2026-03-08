package scaffold

import (
	"fmt"
	"os"
	"slices"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Placement represents the placement definition (Segment 3).
type Placement struct {
	Version        string         `yaml:"version"`
	BaseDir        string         `yaml:"base_dir"`
	ConflictPolicy string         `yaml:"conflict_policy"`
	TemplateConfig TemplateConfig `yaml:"template_config"`
	FileMappings   []FileMapping  `yaml:"file_mappings"`
	PostActions    PostActions    `yaml:"post_actions"`
}

// TemplateConfig defines template processing settings.
type TemplateConfig struct {
	TemplateExtension string `yaml:"template_extension"`
	StripExtension    bool   `yaml:"strip_extension"`
}

// FileMapping defines source-to-target filename mapping.
type FileMapping struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}

// PostActions defines post-processing actions.
type PostActions struct {
	GitignoreEntries []string         `yaml:"gitignore_entries"`
	FilePermissions  []FilePermission `yaml:"file_permissions"`
}

// FilePermission defines a file permission rule applied after scaffold placement.
// Either Executable or Mode must be specified. If both are set, Mode takes precedence.
type FilePermission struct {
	Pattern    string `yaml:"pattern"`
	Executable *bool  `yaml:"executable,omitempty"`
	Mode       string `yaml:"mode,omitempty"`
}

// ResolvedMode returns the resolved os.FileMode for this permission rule.
// If Mode is set, it is parsed as an octal string and returned.
// If Executable is true (and Mode is empty), 0o755 is returned.
// Returns an error if neither is specified or Mode is invalid.
func (fp FilePermission) ResolvedMode() (os.FileMode, error) {
	if fp.Mode != "" {
		parsed, err := strconv.ParseUint(fp.Mode, 8, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid mode %q for pattern %q: %w", fp.Mode, fp.Pattern, err)
		}
		return os.FileMode(parsed), nil
	}
	if fp.Executable != nil && *fp.Executable {
		return 0o755, nil
	}
	return 0, fmt.Errorf("file_permissions entry for pattern %q must specify either 'executable' or 'mode'", fp.Pattern)
}

// IsExecutable returns true if the resolved mode has any execute bit set.
func (fp FilePermission) IsExecutable() bool {
	mode, err := fp.ResolvedMode()
	if err != nil {
		return false
	}
	return mode&0o111 != 0
}

// ValidConflictPolicies lists all allowed conflict resolution policies.
var ValidConflictPolicies = []string{"skip", "overwrite", "append", "error"}

// ParsePlacement parses YAML bytes into a Placement struct and validates it.
// If conflict_policy is empty, it defaults to "skip".
func ParsePlacement(data []byte) (*Placement, error) {
	var p Placement
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to parse placement: %w", err)
	}

	if p.ConflictPolicy == "" {
		p.ConflictPolicy = "skip"
	}

	if !slices.Contains(ValidConflictPolicies, p.ConflictPolicy) {
		return nil, fmt.Errorf("invalid conflict_policy %q, must be one of: %v", p.ConflictPolicy, ValidConflictPolicies)
	}

	if err := validateFilePermissions(p.PostActions.FilePermissions); err != nil {
		return nil, err
	}

	return &p, nil
}

// validateFilePermissions checks that each FilePermission entry is valid.
func validateFilePermissions(perms []FilePermission) error {
	for i, fp := range perms {
		if fp.Pattern == "" {
			return fmt.Errorf("file_permissions[%d]: pattern must not be empty", i)
		}
		// Validate that at least one of executable or mode is specified
		if _, err := fp.ResolvedMode(); err != nil {
			return fmt.Errorf("file_permissions[%d]: %w", i, err)
		}
	}
	return nil
}
