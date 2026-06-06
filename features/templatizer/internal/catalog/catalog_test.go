package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCatalog(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLen   int
		wantFirst *Scaffold
		wantErr   bool
	}{
		{
			name: "valid catalog with 3 scaffolds",
			input: `version: "1.0.0"
scaffolds:
  - name: "default"
    category: "root"
    description: "Tokotachi - The First of All"
    template_ref: "catalog/templates/root/project-default"
    original_ref: "catalog/originals/root/project-default"
  - name: "axsh-go-standard"
    category: "project"
    description: "AXSH Go Standard Project"
    template_ref: "catalog/templates/axsh/go-standard-project"
    original_ref: "catalog/originals/axsh/go-standard-project"
  - name: "axsh-go-standard"
    category: "feature"
    description: "AXSH Go Standard Feature"
    template_ref: "catalog/templates/axsh/go-standard-feature"
    original_ref: "catalog/originals/axsh/go-standard-feature"
`,
			wantLen: 3,
			wantFirst: &Scaffold{
				Name:        "default",
				Category:    "root",
				Description: "Tokotachi - The First of All",
				TemplateRef: "catalog/templates/root/project-default",
				OriginalRef: "catalog/originals/root/project-default",
			},
			wantErr: false,
		},
		{
			name: "empty scaffolds list",
			input: `version: "1.0.0"
scaffolds: []
`,
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "invalid yaml",
			input:   `{{{invalid yaml`,
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat, err := ParseCatalog([]byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, cat.Scaffolds, tt.wantLen)
			if tt.wantFirst != nil && len(cat.Scaffolds) > 0 {
				assert.Equal(t, tt.wantFirst.Name, cat.Scaffolds[0].Name)
				assert.Equal(t, tt.wantFirst.Category, cat.Scaffolds[0].Category)
				assert.Equal(t, tt.wantFirst.TemplateRef, cat.Scaffolds[0].TemplateRef)
				assert.Equal(t, tt.wantFirst.OriginalRef, cat.Scaffolds[0].OriginalRef)
			}
		})
	}
}

func TestParseCatalogWithTemplateParams(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantParamsLen  int
		wantFirstParam *TemplateParam
	}{
		{
			name: "scaffold with template_params",
			input: `version: "1.0.0"
scaffolds:
  - name: "axsh-go-standard"
    category: "feature"
    description: "AXSH Go Standard Feature"
    template_ref: "catalog/templates/axsh/go-standard-feature"
    original_ref: "catalog/originals/axsh/go-standard-feature"
    template_params:
      - name: "module_path"
        description: "Go module path"
        required: true
        old_value: "github.com/axsh/tokotachi/features/myprog"
      - name: "program_name"
        description: "Program name"
        required: true
        old_value: "myprog"
`,
			wantParamsLen: 2,
			wantFirstParam: &TemplateParam{
				Name:        "module_path",
				Description: "Go module path",
				Required:    true,
				OldValue:    "github.com/axsh/tokotachi/features/myprog",
			},
		},
		{
			name: "scaffold without template_params",
			input: `version: "1.0.0"
scaffolds:
  - name: "default"
    category: "root"
    description: "Default"
    template_ref: "ref"
    original_ref: "ref"
`,
			wantParamsLen:  0,
			wantFirstParam: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat, err := ParseCatalog([]byte(tt.input))
			require.NoError(t, err)
			require.NotEmpty(t, cat.Scaffolds)

			params := cat.Scaffolds[0].TemplateParams
			assert.Len(t, params, tt.wantParamsLen)

			if tt.wantFirstParam != nil && len(params) > 0 {
				assert.Equal(t, tt.wantFirstParam.Name, params[0].Name)
				assert.Equal(t, tt.wantFirstParam.Description, params[0].Description)
				assert.Equal(t, tt.wantFirstParam.Required, params[0].Required)
				assert.Equal(t, tt.wantFirstParam.OldValue, params[0].OldValue)
			}
		})
	}
}

func TestParseCatalogWithValueSpec(t *testing.T) {
	t.Run("string type with length and format", func(t *testing.T) {
		input := `version: "1.0.0"
scaffolds:
  - name: "test"
    category: "feature"
    description: "Test"
    template_ref: "ref"
    original_ref: "ref"
    template_params:
      - name: "module_path"
        description: "Go module path"
        required: true
        value_spec:
          type: string
          length:
            max_bytes: 256
          format:
            pattern: "^[a-zA-Z0-9._/-]+$"
`
		cat, err := ParseCatalog([]byte(input))
		require.NoError(t, err)
		require.NotEmpty(t, cat.Scaffolds)
		require.Len(t, cat.Scaffolds[0].TemplateParams, 1)

		vs := cat.Scaffolds[0].TemplateParams[0].ValueSpec
		require.NotNil(t, vs)
		assert.Equal(t, "string", vs.Type)
		require.NotNil(t, vs.Length)
		require.NotNil(t, vs.Length.MaxBytes)
		assert.Equal(t, 256, *vs.Length.MaxBytes)
		assert.Nil(t, vs.Length.MaxChars)
		require.NotNil(t, vs.Format)
		assert.Equal(t, "^[a-zA-Z0-9._/-]+$", vs.Format.Pattern)
		assert.Nil(t, vs.Range)
		assert.Nil(t, vs.Enum)
	})

	t.Run("number type with range", func(t *testing.T) {
		input := `version: "1.0.0"
scaffolds:
  - name: "test"
    category: "feature"
    description: "Test"
    template_ref: "ref"
    original_ref: "ref"
    template_params:
      - name: "port_number"
        description: "Server port"
        required: false
        default: "8080"
        value_spec:
          type: number
          range:
            minimum: 1
            maximum: 65535
`
		cat, err := ParseCatalog([]byte(input))
		require.NoError(t, err)
		require.NotEmpty(t, cat.Scaffolds)

		vs := cat.Scaffolds[0].TemplateParams[0].ValueSpec
		require.NotNil(t, vs)
		assert.Equal(t, "number", vs.Type)
		assert.Nil(t, vs.Length)
		assert.Nil(t, vs.Format)
		require.NotNil(t, vs.Range)
		require.NotNil(t, vs.Range.Minimum)
		assert.Equal(t, float64(1), *vs.Range.Minimum)
		require.NotNil(t, vs.Range.Maximum)
		assert.Equal(t, float64(65535), *vs.Range.Maximum)
		assert.Nil(t, vs.Range.ExclusiveMinimum)
		assert.Nil(t, vs.Range.ExclusiveMaximum)
	})

	t.Run("enum constraint", func(t *testing.T) {
		input := `version: "1.0.0"
scaffolds:
  - name: "test"
    category: "feature"
    description: "Test"
    template_ref: "ref"
    original_ref: "ref"
    template_params:
      - name: "build_mode"
        description: "Build mode"
        required: true
        value_spec:
          type: string
          enum:
            - "debug"
            - "release"
`
		cat, err := ParseCatalog([]byte(input))
		require.NoError(t, err)
		require.NotEmpty(t, cat.Scaffolds)

		vs := cat.Scaffolds[0].TemplateParams[0].ValueSpec
		require.NotNil(t, vs)
		assert.Equal(t, "string", vs.Type)
		require.NotNil(t, vs.Enum)
		assert.Equal(t, []string{"debug", "release"}, vs.Enum)
	})

	t.Run("without value_spec returns nil", func(t *testing.T) {
		input := `version: "1.0.0"
scaffolds:
  - name: "test"
    category: "feature"
    description: "Test"
    template_ref: "ref"
    original_ref: "ref"
    template_params:
      - name: "module_path"
        description: "Go module path"
        required: true
`
		cat, err := ParseCatalog([]byte(input))
		require.NoError(t, err)
		require.NotEmpty(t, cat.Scaffolds)
		require.Len(t, cat.Scaffolds[0].TemplateParams, 1)
		assert.Nil(t, cat.Scaffolds[0].TemplateParams[0].ValueSpec)
	})

	t.Run("scaffold definition with value_spec", func(t *testing.T) {
		yaml := `
name: "test"
category: "feature"
description: "Test scaffold"
original_ref: "catalog/originals/test"
template_params:
  - name: "feature_name"
    description: "Feature name"
    required: true
    default: "myfunction"
    value_spec:
      type: string
      length:
        max_bytes: 64
        max_chars: 32
      format:
        pattern: "^[a-z][a-z0-9_-]*$"
`
		def, err := ParseScaffoldDefinition([]byte(yaml))
		require.NoError(t, err)
		require.Len(t, def.TemplateParams, 1)

		vs := def.TemplateParams[0].ValueSpec
		require.NotNil(t, vs)
		assert.Equal(t, "string", vs.Type)
		require.NotNil(t, vs.Length)
		require.NotNil(t, vs.Length.MaxBytes)
		assert.Equal(t, 64, *vs.Length.MaxBytes)
		require.NotNil(t, vs.Length.MaxChars)
		assert.Equal(t, 32, *vs.Length.MaxChars)
		require.NotNil(t, vs.Format)
		assert.Equal(t, "^[a-z][a-z0-9_-]*$", vs.Format.Pattern)
	})
}

func TestParseCatalogWithDependsOn(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantDepsLen  int
		wantFirstDep *DependencyRef
	}{
		{
			name: "scaffold with single depends_on",
			input: `version: "1.0.0"
scaffolds:
  - name: "axsh-go-standard"
    category: "project"
    description: "AXSH Go Standard Project"
    depends_on:
      - category: "root"
        name: "default"
    template_ref: "catalog/templates/axsh/go-standard-project"
    original_ref: "catalog/originals/axsh/go-standard-project"
`,
			wantDepsLen: 1,
			wantFirstDep: &DependencyRef{
				Category: "root",
				Name:     "default",
			},
		},
		{
			name: "scaffold with multiple depends_on",
			input: `version: "1.0.0"
scaffolds:
  - name: "feature-x"
    category: "feature"
    description: "Feature X"
    depends_on:
      - category: "project"
        name: "project-a"
      - category: "project"
        name: "project-b"
    template_ref: "ref"
    original_ref: "ref"
`,
			wantDepsLen: 2,
			wantFirstDep: &DependencyRef{
				Category: "project",
				Name:     "project-a",
			},
		},
		{
			name: "scaffold without depends_on",
			input: `version: "1.0.0"
scaffolds:
  - name: "default"
    category: "root"
    description: "Default"
    template_ref: "ref"
    original_ref: "ref"
`,
			wantDepsLen:  0,
			wantFirstDep: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat, err := ParseCatalog([]byte(tt.input))
			require.NoError(t, err)
			require.NotEmpty(t, cat.Scaffolds)

			deps := cat.Scaffolds[0].DependsOn
			assert.Len(t, deps, tt.wantDepsLen)

			if tt.wantFirstDep != nil && len(deps) > 0 {
				assert.Equal(t, tt.wantFirstDep.Category, deps[0].Category)
				assert.Equal(t, tt.wantFirstDep.Name, deps[0].Name)
			}
		})
	}
}

func TestValidateDependencies(t *testing.T) {
	tests := []struct {
		name      string
		scaffolds []Scaffold
		wantErr   bool
		errSubstr string
	}{
		{
			name: "valid dependency chain",
			scaffolds: []Scaffold{
				{Name: "default", Category: "root"},
				{Name: "axsh-go-standard", Category: "project", DependsOn: []DependencyRef{{Category: "root", Name: "default"}}},
				{Name: "axsh-go-standard", Category: "feature", DependsOn: []DependencyRef{{Category: "project", Name: "axsh-go-standard"}}},
			},
			wantErr: false,
		},
		{
			name: "nonexistent dependency",
			scaffolds: []Scaffold{
				{Name: "default", Category: "root"},
				{Name: "test", Category: "feature", DependsOn: []DependencyRef{{Category: "project", Name: "nonexistent"}}},
			},
			wantErr:   true,
			errSubstr: "not found",
		},
		{
			name: "circular dependency",
			scaffolds: []Scaffold{
				{Name: "a", Category: "root", DependsOn: []DependencyRef{{Category: "root", Name: "b"}}},
				{Name: "b", Category: "root", DependsOn: []DependencyRef{{Category: "root", Name: "a"}}},
			},
			wantErr:   true,
			errSubstr: "circular",
		},
		{
			name: "all independent scaffolds",
			scaffolds: []Scaffold{
				{Name: "default", Category: "root"},
				{Name: "other", Category: "root"},
			},
			wantErr: false,
		},
		{
			name: "self reference",
			scaffolds: []Scaffold{
				{Name: "self", Category: "root", DependsOn: []DependencyRef{{Category: "root", Name: "self"}}},
			},
			wantErr:   true,
			errSubstr: "circular",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat := &Catalog{Version: "1.0.0", Scaffolds: tt.scaffolds}
			err := cat.ValidateDependencies()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestResolveDependencyChain(t *testing.T) {
	tests := []struct {
		name      string
		scaffolds []Scaffold
		category  string
		scaffName string
		wantOrder []string // expected order as "category/name"
		wantErr   bool
	}{
		{
			name: "linear chain feature -> project -> root",
			scaffolds: []Scaffold{
				{Name: "default", Category: "root"},
				{Name: "axsh-go-standard", Category: "project", DependsOn: []DependencyRef{{Category: "root", Name: "default"}}},
				{Name: "axsh-go-standard", Category: "feature", DependsOn: []DependencyRef{{Category: "project", Name: "axsh-go-standard"}}},
			},
			category:  "feature",
			scaffName: "axsh-go-standard",
			wantOrder: []string{"root/default", "project/axsh-go-standard", "feature/axsh-go-standard"},
			wantErr:   false,
		},
		{
			name: "no dependencies",
			scaffolds: []Scaffold{
				{Name: "default", Category: "root"},
			},
			category:  "root",
			scaffName: "default",
			wantOrder: []string{"root/default"},
			wantErr:   false,
		},
		{
			name: "diamond dependency",
			scaffolds: []Scaffold{
				{Name: "a", Category: "root"},
				{Name: "b", Category: "project", DependsOn: []DependencyRef{{Category: "root", Name: "a"}}},
				{Name: "c", Category: "project", DependsOn: []DependencyRef{{Category: "root", Name: "a"}}},
				{Name: "d", Category: "feature", DependsOn: []DependencyRef{
					{Category: "project", Name: "b"},
					{Category: "project", Name: "c"},
				}},
			},
			category:  "feature",
			scaffName: "d",
			wantOrder: []string{"root/a", "project/b", "project/c", "feature/d"},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat := &Catalog{Version: "1.0.0", Scaffolds: tt.scaffolds}
			result, err := cat.ResolveDependencyChain(tt.category, tt.scaffName)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			got := make([]string, len(result))
			for i, s := range result {
				got[i] = s.Category + "/" + s.Name
			}
			assert.Equal(t, tt.wantOrder, got)
		})
	}
}

func TestScaffoldHash(t *testing.T) {
	t.Run("idempotent", func(t *testing.T) {
		h1 := ScaffoldHash("root", "default")
		h2 := ScaffoldHash("root", "default")
		assert.Equal(t, h1, h2)
	})

	t.Run("always 4 characters", func(t *testing.T) {
		inputs := []struct{ category, name string }{
			{"root", "default"},
			{"project", "axsh-go-standard"},
			{"feature", "axsh-go-standard"},
			{"feature", "kotoshiro-go-mcp"},
			{"a", "b"},
		}
		for _, in := range inputs {
			h := ScaffoldHash(in.category, in.name)
			assert.Len(t, h, 4, "hash for %s/%s should be 4 chars, got %q", in.category, in.name, h)
		}
	})

	t.Run("charset is 0-9a-z only", func(t *testing.T) {
		h := ScaffoldHash("feature", "axsh-go-standard")
		for _, c := range h {
			valid := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z')
			assert.True(t, valid, "character %c is not in [0-9a-z]", c)
		}
	})

	t.Run("different inputs produce different hashes", func(t *testing.T) {
		h1 := ScaffoldHash("root", "default")
		h2 := ScaffoldHash("project", "axsh-go-standard")
		assert.NotEqual(t, h1, h2)
	})
}

func TestScaffoldShardPath(t *testing.T) {
	tests := []struct {
		name string
		hash string
		want string
	}{
		{
			name: "4 character hash",
			hash: "a3k9",
			want: "catalog/scaffolds/a/3/k/9.yaml",
		},
		{
			name: "all zeros",
			hash: "0000",
			want: "catalog/scaffolds/0/0/0/0.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScaffoldShardPath(tt.hash)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseScaffoldDefinition(t *testing.T) {
	t.Run("with placement", func(t *testing.T) {
		yaml := `
name: "default"
category: "root"
description: "Test scaffold"
original_ref: "catalog/originals/root/project-default"
placement:
  base_dir: "."
  conflict_policy: "skip"
  template_config:
    template_extension: ".tmpl"
    strip_extension: true
  file_mappings: []
  post_actions:
    gitignore_entries:
      - "work/*"
`
		def, err := ParseScaffoldDefinition([]byte(yaml))
		require.NoError(t, err)
		assert.Equal(t, "default", def.Name)
		assert.Equal(t, "root", def.Category)
		require.NotNil(t, def.Placement)
		assert.Equal(t, ".", def.Placement.BaseDir)
		assert.Equal(t, "skip", def.Placement.ConflictPolicy)
		require.NotNil(t, def.Placement.TemplateConfig)
		assert.Equal(t, ".tmpl", def.Placement.TemplateConfig.TemplateExtension)
		assert.True(t, def.Placement.TemplateConfig.StripExtension)
		require.NotNil(t, def.Placement.PostActions)
		assert.Equal(t, []string{"work/*"}, def.Placement.PostActions.GitignoreEntries)
	})

	t.Run("without placement", func(t *testing.T) {
		yaml := `
name: "simple"
category: "test"
description: "No placement"
original_ref: "catalog/originals/test/simple"
`
		def, err := ParseScaffoldDefinition([]byte(yaml))
		require.NoError(t, err)
		assert.Equal(t, "simple", def.Name)
		assert.Nil(t, def.Placement)
	})

	t.Run("full fields with depends_on and template_params", func(t *testing.T) {
		yaml := `
name: "axsh-go-standard"
category: "feature"
description: "AXSH Go Standard Feature"
depends_on:
  - category: "project"
    name: "axsh-go-standard"
original_ref: "catalog/originals/axsh/go-standard-feature"
placement:
  base_dir: "features/myprog"
  conflict_policy: "skip"
  template_config:
    template_extension: ".tmpl"
    strip_extension: true
  post_actions:
    file_permissions:
      - pattern: "scripts/**/*.sh"
        executable: true
template_params:
  - name: "module_path"
    description: "Go module path"
    required: true
    default: "github.com/axsh/tokotachi/features/myprog"
  - name: "program_name"
    description: "Program name"
    required: true
    default: "myprog"
`
		def, err := ParseScaffoldDefinition([]byte(yaml))
		require.NoError(t, err)
		assert.Equal(t, "axsh-go-standard", def.Name)
		assert.Equal(t, "feature", def.Category)
		require.Len(t, def.DependsOn, 1)
		assert.Equal(t, "project", def.DependsOn[0].Category)
		require.NotNil(t, def.Placement)
		require.NotNil(t, def.Placement.PostActions)
		require.Len(t, def.Placement.PostActions.FilePermissions, 1)
		assert.Equal(t, "scripts/**/*.sh", def.Placement.PostActions.FilePermissions[0].Pattern)
		assert.True(t, def.Placement.PostActions.FilePermissions[0].Executable)
		require.Len(t, def.TemplateParams, 2)
	})
}

func TestScanScaffoldDefinitions(t *testing.T) {
	// Create temp directory structure.
	tmpDir := t.TempDir()
	origDir := filepath.Join(tmpDir, "org", "test-scaffold")
	require.NoError(t, os.MkdirAll(origDir, 0o755))

	scaffoldYAML := `
name: "test"
category: "unit"
description: "Test scaffold"
original_ref: "catalog/originals/org/test-scaffold"
`
	require.NoError(t, os.WriteFile(
		filepath.Join(origDir, "scaffold.yaml"), []byte(scaffoldYAML), 0o644,
	))

	// Also create a directory without scaffold.yaml (should be skipped).
	noScaffoldDir := filepath.Join(tmpDir, "org", "no-scaffold")
	require.NoError(t, os.MkdirAll(noScaffoldDir, 0o755))

	defs, err := ScanScaffoldDefinitions(tmpDir)
	require.NoError(t, err)
	require.Len(t, defs, 1)
	assert.Equal(t, "test", defs[0].Name)
	assert.Equal(t, "unit", defs[0].Category)
}

func TestBuildCatalogIndex(t *testing.T) {
	scaffolds := []Scaffold{
		{Name: "default", Category: "root"},
		{Name: "axsh-go-standard", Category: "project"},
		{Name: "axsh-go-standard", Category: "feature"},
		{Name: "kotoshiro-go-mcp", Category: "feature"},
	}

	index := BuildCatalogIndex(scaffolds)

	require.NotNil(t, index)
	require.Contains(t, index.Scaffolds, "root")
	require.Contains(t, index.Scaffolds["root"], "default")
	require.Contains(t, index.Scaffolds, "project")
	require.Contains(t, index.Scaffolds["project"], "axsh-go-standard")
	require.Contains(t, index.Scaffolds, "feature")
	require.Contains(t, index.Scaffolds["feature"], "axsh-go-standard")
	require.Contains(t, index.Scaffolds["feature"], "kotoshiro-go-mcp")

	// Verify paths contain "catalog/scaffolds/" prefix.
	for _, categoryMap := range index.Scaffolds {
		for _, path := range categoryMap {
			assert.Contains(t, path, "catalog/scaffolds/")
			assert.Contains(t, path, ".yaml")
		}
	}
}
