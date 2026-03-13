package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCatalog(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Catalog
		wantErr bool
	}{
		{
			name: "valid catalog with default_scaffold",
			input: `
version: "1.0.0"
default_scaffold: "default"
scaffolds:
  - name: "default"
    category: "root"
    description: "Standard project structure"
    template_ref: "templates/project-default"
    placement_ref: "placements/default.yaml"
    requirements:
      directories: []
      files: []
  - name: "go-standard"
    category: "features"
    description: "Go standard feature template"
    template_ref: "templates/feature-go"
    placement_ref: "placements/features-go-standard.yaml"
    requirements:
      directories:
        - "features/"
      files: []
    options:
      - name: "Name"
        description: "Feature name"
        required: true
      - name: "GoModule"
        description: "Go module path"
        required: false
        default: "github.com/example/{{.Name}}"
`,
			want: &Catalog{
				Version:         "1.0.0",
				DefaultScaffold: "default",
				Scaffolds: []ScaffoldEntry{
					{
						Name:         "default",
						Category:     "root",
						Description:  "Standard project structure",
						TemplateRef:  "templates/project-default",
						PlacementRef: "placements/default.yaml",
						Requirements: Requirements{Directories: []string{}, Files: []string{}},
					},
					{
						Name:         "go-standard",
						Category:     "features",
						Description:  "Go standard feature template",
						TemplateRef:  "templates/feature-go",
						PlacementRef: "placements/features-go-standard.yaml",
						Requirements: Requirements{Directories: []string{"features/"}, Files: []string{}},
						Options: []Option{
							{Name: "Name", Description: "Feature name", Required: true},
							{Name: "GoModule", Description: "Go module path", Required: false, Default: "github.com/example/{{.Name}}"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid YAML",
			input:   "version: [\ninvalid",
			want:    nil,
			wantErr: true,
		},
		{
			name: "missing default_scaffold field",
			input: `
version: "1.0.0"
scaffolds:
  - name: "default"
    category: "root"
`,
			want: &Catalog{
				Version: "1.0.0",
				Scaffolds: []ScaffoldEntry{
					{Name: "default", Category: "root"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCatalog([]byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want.Version, got.Version)
			assert.Equal(t, tt.want.DefaultScaffold, got.DefaultScaffold)
			assert.Equal(t, len(tt.want.Scaffolds), len(got.Scaffolds))
			for i, wantEntry := range tt.want.Scaffolds {
				assert.Equal(t, wantEntry.Name, got.Scaffolds[i].Name)
				assert.Equal(t, wantEntry.Category, got.Scaffolds[i].Category)
			}
		})
	}
}

func TestResolvePattern_Default(t *testing.T) {
	catalog := &Catalog{
		DefaultScaffold: "default",
		Scaffolds: []ScaffoldEntry{
			{Name: "default", Category: "root", Description: "Default template"},
			{Name: "go-standard", Category: "features", Description: "Go template"},
		},
	}

	entries, err := catalog.ResolvePattern(nil)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "default", entries[0].Name)
}

func TestResolvePattern_ByName(t *testing.T) {
	catalog := &Catalog{
		DefaultScaffold: "default",
		Scaffolds: []ScaffoldEntry{
			{Name: "default", Category: "root"},
			{Name: "go-standard", Category: "features"},
		},
	}

	entries, err := catalog.ResolvePattern([]string{"default"})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "default", entries[0].Name)
}

func TestResolvePattern_ByCategory(t *testing.T) {
	catalog := &Catalog{
		Scaffolds: []ScaffoldEntry{
			{Name: "default", Category: "root"},
			{Name: "go-standard", Category: "features"},
			{Name: "python-standard", Category: "features"},
		},
	}

	entries, err := catalog.ResolvePattern([]string{"features"})
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "go-standard", entries[0].Name)
	assert.Equal(t, "python-standard", entries[1].Name)
}

func TestResolvePattern_ByCategoryAndName(t *testing.T) {
	catalog := &Catalog{
		Scaffolds: []ScaffoldEntry{
			{Name: "default", Category: "root"},
			{Name: "go-standard", Category: "features"},
			{Name: "python-standard", Category: "features"},
		},
	}

	entries, err := catalog.ResolvePattern([]string{"features", "go-standard"})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "go-standard", entries[0].Name)
}

func TestResolvePattern_NotFound(t *testing.T) {
	catalog := &Catalog{
		DefaultScaffold: "default",
		Scaffolds: []ScaffoldEntry{
			{Name: "default", Category: "root"},
		},
	}

	_, err := catalog.ResolvePattern([]string{"nonexistent"})
	assert.Error(t, err)
}

func TestResolvePattern_DefaultNotConfigured(t *testing.T) {
	catalog := &Catalog{
		Scaffolds: []ScaffoldEntry{
			{Name: "default", Category: "root"},
		},
	}

	_, err := catalog.ResolvePattern(nil)
	assert.Error(t, err)
}

func TestCheckRequirements_Satisfied(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "features"), 0o755))

	reqs := Requirements{
		Directories: []string{"features/"},
		Files:       []string{},
	}

	err := CheckRequirements(reqs, tmpDir)
	assert.NoError(t, err)
}

func TestCheckRequirements_MissingDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	reqs := Requirements{
		Directories: []string{"features/"},
		Files:       []string{},
	}

	err := CheckRequirements(reqs, tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "features/")
}

func TestCheckRequirements_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	reqs := Requirements{
		Directories: []string{},
		Files:       []string{".devrc.yaml"},
	}

	err := CheckRequirements(reqs, tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ".devrc.yaml")
}

func TestCheckRequirements_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	reqs := Requirements{
		Directories: []string{},
		Files:       []string{},
	}

	err := CheckRequirements(reqs, tmpDir)
	assert.NoError(t, err)
}

func TestListScaffolds(t *testing.T) {
	catalog := &Catalog{
		Scaffolds: []ScaffoldEntry{
			{Name: "default", Category: "root", Description: "Default template"},
			{Name: "go-standard", Category: "features", Description: "Go template"},
		},
	}

	list := catalog.ListScaffolds()
	assert.Len(t, list, 2)
	assert.Equal(t, "default", list[0].Name)
	assert.Equal(t, "go-standard", list[1].Name)
}
