package compiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantID  string
		wantErr bool
	}{
		{
			name:   "valid project.yaml",
			yaml:   "version: 1\nproject_id: vv5\nsources:\n  policies: \"policies/**/*.yaml\"\noutputs:\n  resolved_manifest: tmp/dist/manifest.resolved.yaml\n  memory_index: prompts/memory/index.md\ndefaults:\n  language: ja\n  generated_banner: true\n  build_dir: tmp/dist/\n",
			wantID: "vv5",
		},
		{
			name:    "missing version",
			yaml:    "project_id: vv5\nsources:\n  policies: \"p\"\n",
			wantErr: true,
		},
		{
			name:    "version zero",
			yaml:    "version: 0\nproject_id: vv5\nsources:\n  policies: \"p\"\n",
			wantErr: true,
		},
		{
			name:    "missing project_id",
			yaml:    "version: 1\nsources:\n  policies: \"p\"\n",
			wantErr: true,
		},
		{
			name:    "invalid yaml",
			yaml:    "{{{{invalid",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write yaml to temp file
			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, "project.yaml")
			if err := os.WriteFile(path, []byte(tt.yaml), 0644); err != nil {
				t.Fatal(err)
			}

			cfg, err := LoadConfig(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && cfg.ProjectID != tt.wantID {
				t.Errorf("LoadConfig() ProjectID = %q, want %q", cfg.ProjectID, tt.wantID)
			}
		})
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/project.yaml")
	if err == nil {
		t.Error("LoadConfig() expected error for nonexistent file")
	}
}

func TestResolveProjectRoot(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantTail string // expected suffix of the resolved root
		wantErr  bool
	}{
		{
			name:     "standard path",
			path:     filepath.Join("some", "root", "prompts", "manifest", "project.yaml"),
			wantTail: filepath.Join("some", "root"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveProjectRoot(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveProjectRoot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				// The result should end with the expected tail
				absPath, _ := filepath.Abs(tt.wantTail)
				if got != absPath {
					t.Errorf("ResolveProjectRoot() = %q, want suffix %q", got, absPath)
				}
			}
		})
	}
}
