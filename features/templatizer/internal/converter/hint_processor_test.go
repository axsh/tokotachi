package converter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProcessHints(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, dir string)
		params    map[string]string
		wantFiles map[string]string // expected file name -> expected content substring
		wantGone  []string          // files that should not exist after processing
		wantErr   bool
	}{
		{
			name: "applies replacement and renames to tmpl",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mustWriteFile(t, filepath.Join(dir, "Makefile"), "APP_NAME=old-app\nMODULE=github.com/old-org/old-app\n")
				mustWriteFile(t, filepath.Join(dir, "Makefile.hints"), `replacements:
  - match: "old-app"
    replace_with: "{{feature_name}}"
  - match: "github.com/old-org/old-app"
    replace_with: "{{module_path}}"
`)
			},
			params: map[string]string{
				"feature_name": "new-app",
				"module_path":  "github.com/new-org/new-app",
			},
			wantFiles: map[string]string{
				"Makefile.tmpl": "APP_NAME=new-app",
			},
			wantGone: []string{"Makefile.hints", "Makefile"},
		},
		{
			name: "processes multiple hint files",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mustWriteFile(t, filepath.Join(dir, "Makefile"), "old-app")
				mustWriteFile(t, filepath.Join(dir, "Makefile.hints"), `replacements:
  - match: "old-app"
    replace_with: "{{feature_name}}"
`)
				mustWriteFile(t, filepath.Join(dir, "Dockerfile"), "FROM old-app")
				mustWriteFile(t, filepath.Join(dir, "Dockerfile.hints"), `replacements:
  - match: "old-app"
    replace_with: "{{feature_name}}"
`)
			},
			params: map[string]string{
				"feature_name": "new-app",
			},
			wantFiles: map[string]string{
				"Makefile.tmpl":   "new-app",
				"Dockerfile.tmpl": "FROM new-app",
			},
			wantGone: []string{"Makefile.hints", "Dockerfile.hints", "Makefile", "Dockerfile"},
		},
		{
			name: "no error when no hint files exist",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mustWriteFile(t, filepath.Join(dir, "main.go"), "package main")
			},
			params:    map[string]string{},
			wantFiles: map[string]string{},
			wantGone:  []string{},
		},
		{
			name: "error on invalid yaml hints",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mustWriteFile(t, filepath.Join(dir, "config.yaml"), "key: value")
				mustWriteFile(t, filepath.Join(dir, "config.yaml.hints"), "{{{invalid yaml")
			},
			params:  map[string]string{},
			wantErr: true,
		},
		{
			name: "hints in subdirectory",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mustMkdir(t, filepath.Join(dir, "scripts"))
				mustWriteFile(t, filepath.Join(dir, "scripts", "build.sh"), "old-app build")
				mustWriteFile(t, filepath.Join(dir, "scripts", "build.sh.hints"), `replacements:
  - match: "old-app"
    replace_with: "{{feature_name}}"
`)
			},
			params: map[string]string{
				"feature_name": "new-app",
			},
			wantFiles: map[string]string{
				"scripts/build.sh.tmpl": "new-app build",
			},
			wantGone: []string{"scripts/build.sh.hints", "scripts/build.sh"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(t, tmpDir)

			err := ProcessHints(tmpDir, tt.params)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ProcessHints() expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ProcessHints() unexpected error: %v", err)
			}

			for wantFile, wantContains := range tt.wantFiles {
				p := filepath.Join(tmpDir, wantFile)
				content, readErr := os.ReadFile(p)
				if readErr != nil {
					t.Errorf("expected file %q to exist: %v", wantFile, readErr)
					continue
				}
				if wantContains != "" {
					if got := string(content); !containsStr(got, wantContains) {
						t.Errorf("file %q: expected to contain %q, got %q", wantFile, wantContains, got)
					}
				}
			}

			for _, gone := range tt.wantGone {
				p := filepath.Join(tmpDir, gone)
				if _, err := os.Stat(p); !os.IsNotExist(err) {
					t.Errorf("expected %q to be removed, but it still exists", gone)
				}
			}
		})
	}
}

func containsStr(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := range len(s) - len(substr) + 1 {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
