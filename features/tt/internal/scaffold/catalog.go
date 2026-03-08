package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Catalog represents the top-level catalog structure (Segment 1).
type Catalog struct {
	Version         string          `yaml:"version"`
	DefaultScaffold string          `yaml:"default_scaffold"`
	Scaffolds       []ScaffoldEntry `yaml:"scaffolds"`
}

// ScaffoldEntry represents a single scaffold template entry in the catalog.
type ScaffoldEntry struct {
	Name         string       `yaml:"name"`
	Category     string       `yaml:"category"`
	Description  string       `yaml:"description"`
	TemplateRef  string       `yaml:"template_ref"`
	PlacementRef string       `yaml:"placement_ref"`
	Requirements Requirements `yaml:"requirements"`
	Options      []Option     `yaml:"options"`
}

// Requirements defines prerequisites for applying a scaffold template.
type Requirements struct {
	Directories []string `yaml:"directories"`
	Files       []string `yaml:"files"`
}

// Option defines a template variable that can be specified via CLI flags.
type Option struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Default     string `yaml:"default"`
}

// ParseCatalog parses YAML bytes into a Catalog struct.
func ParseCatalog(data []byte) (*Catalog, error) {
	var catalog Catalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse catalog: %w", err)
	}
	return &catalog, nil
}

// ResolvePattern resolves command arguments to matching scaffold entries.
//
// Resolution rules:
//   - nil/empty pattern: use default_scaffold to find the entry
//   - 1 arg: try exact match by name first, then by category
//   - 2 args: match by category (arg[0]) and name (arg[1])
func (c *Catalog) ResolvePattern(pattern []string) ([]ScaffoldEntry, error) {
	if len(pattern) == 0 {
		return c.resolveDefault()
	}

	if len(pattern) == 1 {
		return c.resolveSingle(pattern[0])
	}

	return c.resolveCategoryAndName(pattern[0], pattern[1])
}

// resolveDefault finds the scaffold marked as default_scaffold.
func (c *Catalog) resolveDefault() ([]ScaffoldEntry, error) {
	if c.DefaultScaffold == "" {
		return nil, fmt.Errorf("no default scaffold configured (default_scaffold is empty)")
	}

	for _, entry := range c.Scaffolds {
		if entry.Name == c.DefaultScaffold {
			return []ScaffoldEntry{entry}, nil
		}
	}

	return nil, fmt.Errorf("default scaffold %q not found in catalog", c.DefaultScaffold)
}

// resolveSingle tries to match by name first, then by category.
func (c *Catalog) resolveSingle(nameOrCategory string) ([]ScaffoldEntry, error) {
	// Try exact name match first
	for _, entry := range c.Scaffolds {
		if entry.Name == nameOrCategory {
			return []ScaffoldEntry{entry}, nil
		}
	}

	// Try category match
	var matches []ScaffoldEntry
	for _, entry := range c.Scaffolds {
		if entry.Category == nameOrCategory {
			matches = append(matches, entry)
		}
	}

	if len(matches) > 0 {
		return matches, nil
	}

	return nil, fmt.Errorf("no scaffold found matching %q", nameOrCategory)
}

// resolveCategoryAndName finds an entry matching both category and name.
func (c *Catalog) resolveCategoryAndName(category, name string) ([]ScaffoldEntry, error) {
	for _, entry := range c.Scaffolds {
		if entry.Category == category && entry.Name == name {
			return []ScaffoldEntry{entry}, nil
		}
	}

	return nil, fmt.Errorf("no scaffold found matching category=%q name=%q", category, name)
}

// CheckRequirements verifies that all prerequisites are met in the given repo root.
// Returns an error with details and hints if any requirement is missing.
func CheckRequirements(reqs Requirements, repoRoot string) error {
	var missing []string

	for _, dir := range reqs.Directories {
		path := filepath.Join(repoRoot, dir)
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			missing = append(missing, fmt.Sprintf("directory: %s", dir))
		}
	}

	for _, file := range reqs.Files {
		path := filepath.Join(repoRoot, file)
		if _, err := os.Stat(path); err != nil {
			missing = append(missing, fmt.Sprintf("file: %s", file))
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("prerequisites not met:\n  Missing %s\n  Hint: Run \"tt scaffold\" first to create the base project structure",
			strings.Join(missing, "\n  Missing "))
	}

	return nil
}

// ListScaffolds returns all scaffold entries in the catalog.
func (c *Catalog) ListScaffolds() []ScaffoldEntry {
	return c.Scaffolds
}
