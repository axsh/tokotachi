package scaffold

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePlacement(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Placement
		wantErr bool
	}{
		{
			name: "valid placement with all fields",
			input: `
version: "1.0.0"
base_dir: "."
conflict_policy: "skip"
template_config:
  template_extension: ".tmpl"
  strip_extension: true
file_mappings:
  - source: "dot-gitignore"
    target: ".gitignore"
post_actions:
  gitignore_entries:
    - "work/*"
`,
			want: &Placement{
				Version:        "1.0.0",
				BaseDir:        ".",
				ConflictPolicy: "skip",
				TemplateConfig: TemplateConfig{
					TemplateExtension: ".tmpl",
					StripExtension:    true,
				},
				FileMappings: []FileMapping{
					{Source: "dot-gitignore", Target: ".gitignore"},
				},
				PostActions: PostActions{
					GitignoreEntries: []string{"work/*"},
				},
			},
			wantErr: false,
		},
		{
			name: "default conflict policy when empty",
			input: `
version: "1.0.0"
base_dir: "features/test"
`,
			want: &Placement{
				Version:        "1.0.0",
				BaseDir:        "features/test",
				ConflictPolicy: "skip",
			},
			wantErr: false,
		},
		{
			name: "overwrite conflict policy",
			input: `
version: "1.0.0"
base_dir: "."
conflict_policy: "overwrite"
`,
			want: &Placement{
				Version:        "1.0.0",
				BaseDir:        ".",
				ConflictPolicy: "overwrite",
			},
			wantErr: false,
		},
		{
			name: "error conflict policy",
			input: `
version: "1.0.0"
base_dir: "."
conflict_policy: "error"
`,
			want: &Placement{
				Version:        "1.0.0",
				BaseDir:        ".",
				ConflictPolicy: "error",
			},
			wantErr: false,
		},
		{
			name: "append conflict policy",
			input: `
version: "1.0.0"
base_dir: "."
conflict_policy: "append"
`,
			want: &Placement{
				Version:        "1.0.0",
				BaseDir:        ".",
				ConflictPolicy: "append",
			},
			wantErr: false,
		},
		{
			name: "invalid conflict policy",
			input: `
version: "1.0.0"
base_dir: "."
conflict_policy: "merge"
`,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid YAML",
			input:   "base_dir: [\ninvalid",
			want:    nil,
			wantErr: true,
		},
		{
			name: "valid placement with file_permissions executable",
			input: `
version: "1.0.0"
base_dir: "."
post_actions:
  file_permissions:
    - pattern: "scripts/**/*.sh"
      executable: true
`,
			want: &Placement{
				Version:        "1.0.0",
				BaseDir:        ".",
				ConflictPolicy: "skip",
				PostActions: PostActions{
					FilePermissions: []FilePermission{
						{Pattern: "scripts/**/*.sh", Executable: boolPtr(true)},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid placement with file_permissions mode",
			input: `
version: "1.0.0"
base_dir: "."
post_actions:
  file_permissions:
    - pattern: "secrets/**/*"
      mode: "0600"
`,
			want: &Placement{
				Version:        "1.0.0",
				BaseDir:        ".",
				ConflictPolicy: "skip",
				PostActions: PostActions{
					FilePermissions: []FilePermission{
						{Pattern: "secrets/**/*", Mode: "0600"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "file_permissions with both executable and mode",
			input: `
version: "1.0.0"
base_dir: "."
post_actions:
  file_permissions:
    - pattern: "scripts/*.sh"
      executable: true
      mode: "0700"
`,
			want: &Placement{
				Version:        "1.0.0",
				BaseDir:        ".",
				ConflictPolicy: "skip",
				PostActions: PostActions{
					FilePermissions: []FilePermission{
						{Pattern: "scripts/*.sh", Executable: boolPtr(true), Mode: "0700"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "file_permissions with neither executable nor mode",
			input: `
version: "1.0.0"
base_dir: "."
post_actions:
  file_permissions:
    - pattern: "scripts/*.sh"
`,
			wantErr: true,
		},
		{
			name: "file_permissions with invalid mode",
			input: `
version: "1.0.0"
base_dir: "."
post_actions:
  file_permissions:
    - pattern: "scripts/*.sh"
      mode: "abc"
`,
			wantErr: true,
		},
		{
			name: "file_permissions with empty pattern",
			input: `
version: "1.0.0"
base_dir: "."
post_actions:
  file_permissions:
    - pattern: ""
      executable: true
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePlacement([]byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want.Version, got.Version)
			assert.Equal(t, tt.want.BaseDir, got.BaseDir)
			assert.Equal(t, tt.want.ConflictPolicy, got.ConflictPolicy)
		})
	}
}

func TestFilePermission_ResolvedMode(t *testing.T) {
	tests := []struct {
		name    string
		perm    FilePermission
		want    uint32
		wantErr bool
	}{
		{
			name: "executable true resolves to 0755",
			perm: FilePermission{Pattern: "*.sh", Executable: boolPtr(true)},
			want: 0o755,
		},
		{
			name: "mode 0600",
			perm: FilePermission{Pattern: "*.key", Mode: "0600"},
			want: 0o600,
		},
		{
			name: "mode 0644",
			perm: FilePermission{Pattern: "*.yaml", Mode: "0644"},
			want: 0o644,
		},
		{
			name: "mode takes precedence over executable",
			perm: FilePermission{Pattern: "*.sh", Executable: boolPtr(true), Mode: "0700"},
			want: 0o700,
		},
		{
			name:    "neither executable nor mode",
			perm:    FilePermission{Pattern: "*.sh"},
			wantErr: true,
		},
		{
			name:    "invalid mode string",
			perm:    FilePermission{Pattern: "*.sh", Mode: "abc"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, err := tt.perm.ResolvedMode()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, uint32(mode))
		})
	}
}

func TestFilePermission_IsExecutable(t *testing.T) {
	tests := []struct {
		name string
		perm FilePermission
		want bool
	}{
		{
			name: "mode 0755 is executable",
			perm: FilePermission{Pattern: "*.sh", Mode: "0755"},
			want: true,
		},
		{
			name: "mode 0700 is executable",
			perm: FilePermission{Pattern: "*.sh", Mode: "0700"},
			want: true,
		},
		{
			name: "mode 0644 is not executable",
			perm: FilePermission{Pattern: "*.yaml", Mode: "0644"},
			want: false,
		},
		{
			name: "mode 0600 is not executable",
			perm: FilePermission{Pattern: "*.key", Mode: "0600"},
			want: false,
		},
		{
			name: "executable true is executable",
			perm: FilePermission{Pattern: "*.sh", Executable: boolPtr(true)},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.perm.IsExecutable()
			assert.Equal(t, tt.want, got)
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}
