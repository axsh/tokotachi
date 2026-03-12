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
	Name           string       `yaml:"name"`
	Category       string       `yaml:"category"`
	Description    string       `yaml:"description"`
	TemplateRef    string       `yaml:"template_ref"`
	OriginalRef    string       `yaml:"original_ref"`
	DependsOn      []Dependency `yaml:"depends_on"`
	TemplateParams []Option     `yaml:"template_params"`
	// Legacy fields (old catalog format)
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

// --- New catalog index format support ---

// CatalogIndex represents the new index-style catalog.yaml.
// The top-level scaffolds field is a nested map: category -> name -> ref path.
type CatalogIndex struct {
	Scaffolds map[string]map[string]string `yaml:"scaffolds"`
}

// IndexRef represents a resolved reference from the catalog index.
type IndexRef struct {
	Category string
	Name     string
	Ref      string // Path to the individual scaffold YAML file
}

// Dependency represents a scaffold dependency.
type Dependency struct {
	Category string `yaml:"category"`
	Name     string `yaml:"name"`
}

// ScaffoldDetail represents the wrapper for individual scaffold YAML files.
type ScaffoldDetail struct {
	Scaffolds []ScaffoldEntry `yaml:"scaffolds"`
}

// ParseCatalogIndex parses the new index-style catalog.yaml into a CatalogIndex.
func ParseCatalogIndex(data []byte) (*CatalogIndex, error) {
	var idx CatalogIndex
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("failed to parse catalog index: %w", err)
	}
	if len(idx.Scaffolds) == 0 {
		return nil, fmt.Errorf("catalog index has no scaffolds")
	}
	return &idx, nil
}

// ParseScaffoldDetail parses an individual scaffold YAML file.
func ParseScaffoldDetail(data []byte) ([]ScaffoldEntry, error) {
	var detail ScaffoldDetail
	if err := yaml.Unmarshal(data, &detail); err != nil {
		return nil, fmt.Errorf("failed to parse scaffold detail: %w", err)
	}
	return detail.Scaffolds, nil
}

// ResolveFromIndex resolves command arguments to matching index refs.
//
// Resolution rules:
//   - nil/empty pattern: return the "root"/"default" entry
//   - 1 arg: try exact name match across all categories, then category match
//   - 2 args: match by category (arg[0]) and name (arg[1])
func (idx *CatalogIndex) ResolveFromIndex(pattern []string) ([]IndexRef, error) {
	if len(pattern) == 0 {
		return idx.resolveIndexDefault()
	}
	if len(pattern) == 1 {
		return idx.resolveIndexSingle(pattern[0])
	}
	return idx.resolveIndexCategoryAndName(pattern[0], pattern[1])
}

// resolveIndexDefault returns the default scaffold (root/default).
func (idx *CatalogIndex) resolveIndexDefault() ([]IndexRef, error) {
	if rootEntries, ok := idx.Scaffolds["root"]; ok {
		if ref, ok := rootEntries["default"]; ok {
			return []IndexRef{{Category: "root", Name: "default", Ref: ref}}, nil
		}
	}
	return nil, fmt.Errorf("no default scaffold found (expected root/default in catalog index)")
}

// resolveIndexSingle tries to match by name first, then by category.
func (idx *CatalogIndex) resolveIndexSingle(nameOrCategory string) ([]IndexRef, error) {
	// Try exact name match across all categories
	for category, entries := range idx.Scaffolds {
		if ref, ok := entries[nameOrCategory]; ok {
			return []IndexRef{{Category: category, Name: nameOrCategory, Ref: ref}}, nil
		}
	}

	// Try category match
	if entries, ok := idx.Scaffolds[nameOrCategory]; ok {
		var refs []IndexRef
		for name, ref := range entries {
			refs = append(refs, IndexRef{Category: nameOrCategory, Name: name, Ref: ref})
		}
		return refs, nil
	}

	return nil, fmt.Errorf("no scaffold found matching %q", nameOrCategory)
}

// resolveIndexCategoryAndName finds an entry matching both category and name.
func (idx *CatalogIndex) resolveIndexCategoryAndName(category, name string) ([]IndexRef, error) {
	if entries, ok := idx.Scaffolds[category]; ok {
		if ref, ok := entries[name]; ok {
			return []IndexRef{{Category: category, Name: name, Ref: ref}}, nil
		}
	}
	return nil, fmt.Errorf("no scaffold found matching category=%q name=%q", category, name)
}
