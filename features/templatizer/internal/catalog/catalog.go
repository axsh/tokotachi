package catalog

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValueSpec defines validation rules for a template parameter value.
type ValueSpec struct {
	Type   string      `yaml:"type,omitempty"`   // "string" or "number"
	Length *LengthSpec `yaml:"length,omitempty"`
	Format *FormatSpec `yaml:"format,omitempty"`
	Range  *RangeSpec  `yaml:"range,omitempty"`
	Enum   []string    `yaml:"enum,omitempty"`
}

// LengthSpec defines length constraints for parameter values.
type LengthSpec struct {
	MaxBytes  *int `yaml:"max_bytes,omitempty"`
	MaxChars  *int `yaml:"max_chars,omitempty"`
	MaxDigits *int `yaml:"max_digits,omitempty"`
}

// FormatSpec defines format constraints using regular expressions.
type FormatSpec struct {
	Pattern string `yaml:"pattern,omitempty"`
}

// RangeSpec defines numeric range constraints (JSONSchema style).
type RangeSpec struct {
	Minimum          *float64 `yaml:"minimum,omitempty"`
	Maximum          *float64 `yaml:"maximum,omitempty"`
	ExclusiveMinimum *float64 `yaml:"exclusive_minimum,omitempty"`
	ExclusiveMaximum *float64 `yaml:"exclusive_maximum,omitempty"`
}

// DefaultValueSpec returns the default ValueSpec for auto-added parameters.
// Type: "string", MaxBytes: 256.
func DefaultValueSpec() *ValueSpec {
	maxBytes := 256
	return &ValueSpec{
		Type: "string",
		Length: &LengthSpec{
			MaxBytes: &maxBytes,
		},
	}
}

// TemplateParam represents a single template conversion parameter.
type TemplateParam struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description"`
	Required    bool       `yaml:"required"`
	Default     string     `yaml:"default,omitempty"`
	OldValue    string     `yaml:"old_value"`
	ValueSpec   *ValueSpec `yaml:"value_spec,omitempty"`
}

// DependencyRef represents a reference to a dependency scaffold.
type DependencyRef struct {
	Category string `yaml:"category"`
	Name     string `yaml:"name"`
}

// Scaffold represents a single scaffold entry from catalog.yaml.
type Scaffold struct {
	Name           string          `yaml:"name"`
	Category       string          `yaml:"category"`
	Description    string          `yaml:"description"`
	DependsOn      []DependencyRef `yaml:"depends_on,omitempty"`
	TemplateRef    string          `yaml:"template_ref"`
	OriginalRef    string          `yaml:"original_ref"`
	PlacementRef   string          `yaml:"placement_ref,omitempty"`
	TemplateParams []TemplateParam `yaml:"template_params,omitempty"`
}

// Catalog represents the top-level catalog.yaml structure.
type Catalog struct {
	Version         string     `yaml:"version"`
	DefaultScaffold string     `yaml:"default_scaffold,omitempty"`
	Scaffolds       []Scaffold `yaml:"scaffolds"`
}

// ParseCatalog parses the YAML bytes and returns a Catalog.
// Returns error if the YAML is invalid.
func ParseCatalog(data []byte) (*Catalog, error) {
	var cat Catalog
	if err := yaml.Unmarshal(data, &cat); err != nil {
		return nil, fmt.Errorf("failed to parse catalog YAML: %w", err)
	}
	return &cat, nil
}

// LoadCatalog reads a catalog.yaml file and returns a Catalog.
// Returns error if the file cannot be read or parsed.
func LoadCatalog(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read catalog file %s: %w", path, err)
	}
	return ParseCatalog(data)
}

// scaffoldKey returns a unique key string for a scaffold identified by category and name.
func scaffoldKey(category, name string) string {
	return category + "/" + name
}

// FindScaffold finds a scaffold by category and name.
// Returns the scaffold and true if found, nil and false otherwise.
func (c *Catalog) FindScaffold(category, name string) (*Scaffold, bool) {
	for i := range c.Scaffolds {
		if c.Scaffolds[i].Category == category && c.Scaffolds[i].Name == name {
			return &c.Scaffolds[i], true
		}
	}
	return nil, false
}

// ValidateDependencies checks the referential integrity and circular dependencies
// of all depends_on references in the catalog.
func (c *Catalog) ValidateDependencies() error {
	// Build a lookup map for quick scaffold resolution.
	scaffoldMap := make(map[string]bool, len(c.Scaffolds))
	for _, s := range c.Scaffolds {
		scaffoldMap[scaffoldKey(s.Category, s.Name)] = true
	}

	// Check that all depends_on references point to existing scaffolds.
	for _, s := range c.Scaffolds {
		for _, dep := range s.DependsOn {
			depKey := scaffoldKey(dep.Category, dep.Name)
			if !scaffoldMap[depKey] {
				return fmt.Errorf(
					"scaffold %q depends on %q which was not found in the catalog",
					scaffoldKey(s.Category, s.Name), depKey,
				)
			}
		}
	}

	// Detect circular dependencies using DFS with two-color marking.
	visited := make(map[string]bool, len(c.Scaffolds))
	inStack := make(map[string]bool, len(c.Scaffolds))

	var detectCycle func(key string) error
	detectCycle = func(key string) error {
		if inStack[key] {
			return fmt.Errorf("circular dependency detected involving %q", key)
		}
		if visited[key] {
			return nil
		}
		inStack[key] = true

		s, ok := c.FindScaffold(
			strings.SplitN(key, "/", 2)[0],
			strings.SplitN(key, "/", 2)[1],
		)
		if ok {
			for _, dep := range s.DependsOn {
				if err := detectCycle(scaffoldKey(dep.Category, dep.Name)); err != nil {
					return err
				}
			}
		}

		inStack[key] = false
		visited[key] = true
		return nil
	}

	for _, s := range c.Scaffolds {
		key := scaffoldKey(s.Category, s.Name)
		if err := detectCycle(key); err != nil {
			return err
		}
	}

	return nil
}

// ResolveDependencyChain resolves the full dependency graph for the specified scaffold
// using topological sort (DFS post-order). Returns scaffolds in dependency-first order
// (root to leaf). Duplicate scaffolds are included only once.
func (c *Catalog) ResolveDependencyChain(category, name string) ([]Scaffold, error) {
	if _, ok := c.FindScaffold(category, name); !ok {
		return nil, fmt.Errorf("scaffold %q not found", scaffoldKey(category, name))
	}

	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	var result []Scaffold

	var dfs func(key string) error
	dfs = func(key string) error {
		if inStack[key] {
			return fmt.Errorf("circular dependency detected involving %q", key)
		}
		if visited[key] {
			return nil
		}
		inStack[key] = true

		parts := strings.SplitN(key, "/", 2)
		s, _ := c.FindScaffold(parts[0], parts[1])

		for _, dep := range s.DependsOn {
			if err := dfs(scaffoldKey(dep.Category, dep.Name)); err != nil {
				return err
			}
		}

		inStack[key] = false
		visited[key] = true
		result = append(result, *s)
		return nil
	}

	if err := dfs(scaffoldKey(category, name)); err != nil {
		return nil, err
	}

	return result, nil
}

// ScaffoldHash returns a 4-character base-36 hash string for the given
// category and name. Uses FNV-1a 32-bit with modulo 36^4 (1,679,616).
func ScaffoldHash(category, name string) string {
	h := fnv.New32a()
	h.Write([]byte(category + "/" + name))
	v := h.Sum32() % 1679616 // 36^4
	s := strconv.FormatUint(uint64(v), 36)
	return fmt.Sprintf("%04s", s)
}

// ScaffoldShardPath returns the relative file path for a shard file
// based on the given 4-character hash.
// Format: catalog/scaffolds/{h[0]}/{h[1]}/{h[2]}/{h[3]}.yaml
func ScaffoldShardPath(hash string) string {
	return fmt.Sprintf("catalog/scaffolds/%c/%c/%c/%c.yaml",
		hash[0], hash[1], hash[2], hash[3])
}

// ShardFile represents a single shard YAML file containing one or more scaffolds.
type ShardFile struct {
	Scaffolds []Scaffold `yaml:"scaffolds"`
}

// MinimalCatalog represents the minimized catalog.yaml after sharding.
type MinimalCatalog struct {
	Version         string `yaml:"version"`
	DefaultScaffold string `yaml:"default_scaffold"`
	UpdatedAt       string `yaml:"updated_at"`
}

// Placement represents the placement rules for a scaffold.
type Placement struct {
	BaseDir        string          `yaml:"base_dir"`
	ConflictPolicy string          `yaml:"conflict_policy"`
	TemplateConfig *TemplateConfig `yaml:"template_config,omitempty"`
	FileMappings   []interface{}   `yaml:"file_mappings,omitempty"`
	PostActions    *PostActions    `yaml:"post_actions,omitempty"`
}

// TemplateConfig represents template processing configuration.
type TemplateConfig struct {
	TemplateExtension string `yaml:"template_extension"`
	StripExtension    bool   `yaml:"strip_extension"`
}

// PostActions represents post-processing actions after scaffold application.
type PostActions struct {
	GitignoreEntries []string         `yaml:"gitignore_entries,omitempty"`
	FilePermissions  []FilePermission `yaml:"file_permissions,omitempty"`
}

// FilePermission represents a file permission rule.
type FilePermission struct {
	Pattern    string `yaml:"pattern"`
	Executable bool   `yaml:"executable"`
}

// ScaffoldDefinition represents a single scaffold.yaml input file.
// This is the developer-facing format placed in originals/.
type ScaffoldDefinition struct {
	Name           string          `yaml:"name"`
	Category       string          `yaml:"category"`
	Description    string          `yaml:"description"`
	DependsOn      []DependencyRef `yaml:"depends_on,omitempty"`
	OriginalRef    string          `yaml:"original_ref"`
	Placement      *Placement      `yaml:"placement,omitempty"`
	TemplateParams []TemplateParam `yaml:"template_params,omitempty"`
}

// ParseScaffoldDefinition parses a single scaffold.yaml file.
func ParseScaffoldDefinition(data []byte) (*ScaffoldDefinition, error) {
	var def ScaffoldDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse scaffold definition: %w", err)
	}
	return &def, nil
}

// ScanScaffoldDefinitions walks the originals directory and loads
// all scaffold.yaml files, returning them as ScaffoldDefinition slice.
func ScanScaffoldDefinitions(originalsDir string) ([]ScaffoldDefinition, error) {
	var defs []ScaffoldDefinition
	err := filepath.WalkDir(originalsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "scaffold.yaml" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		def, err := ParseScaffoldDefinition(data)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}
		defs = append(defs, *def)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan scaffold definitions: %w", err)
	}
	return defs, nil
}

// CatalogIndex represents the index catalog.yaml file.
// Format: scaffolds -> category -> name -> shard path
type CatalogIndex struct {
	Scaffolds map[string]map[string]string `yaml:"scaffolds"`
}

// MetaCatalog represents the meta.yaml file.
type MetaCatalog struct {
	Version         string `yaml:"version"`
	DefaultScaffold string `yaml:"default_scaffold"`
	UpdatedAt       string `yaml:"updated_at"`
}

// BuildCatalogIndex builds a CatalogIndex from scaffolds.
func BuildCatalogIndex(scaffolds []Scaffold) *CatalogIndex {
	index := &CatalogIndex{Scaffolds: make(map[string]map[string]string)}
	for _, s := range scaffolds {
		h := ScaffoldHash(s.Category, s.Name)
		path := ScaffoldShardPath(h)
		if index.Scaffolds[s.Category] == nil {
			index.Scaffolds[s.Category] = make(map[string]string)
		}
		index.Scaffolds[s.Category][s.Name] = path
	}
	return index
}
