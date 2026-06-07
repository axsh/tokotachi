package converter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClean(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, dir string)
		wantGone []string
		wantKeep []string
	}{
		{
			name: "removes .git directory",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mustMkdir(t, filepath.Join(dir, ".git"))
				mustWriteFile(t, filepath.Join(dir, ".git", "config"), "gitconfig")
				mustWriteFile(t, filepath.Join(dir, "main.go"), "package main")
			},
			wantGone: []string{".git"},
			wantKeep: []string{"main.go"},
		},
		{
			name: "removes go.sum file",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mustWriteFile(t, filepath.Join(dir, "go.sum"), "checksum")
				mustWriteFile(t, filepath.Join(dir, "go.mod"), "module test")
			},
			wantGone: []string{"go.sum"},
			wantKeep: []string{"go.mod"},
		},
		{
			name: "removes vendor and bin directories",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mustMkdir(t, filepath.Join(dir, "vendor"))
				mustWriteFile(t, filepath.Join(dir, "vendor", "lib.go"), "package vendor")
				mustMkdir(t, filepath.Join(dir, "bin"))
				mustWriteFile(t, filepath.Join(dir, "bin", "app"), "binary")
				mustWriteFile(t, filepath.Join(dir, "main.go"), "package main")
			},
			wantGone: []string{"vendor", "bin"},
			wantKeep: []string{"main.go"},
		},
		{
			name: "removes .DS_Store file",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mustWriteFile(t, filepath.Join(dir, ".DS_Store"), "dsstore")
				mustWriteFile(t, filepath.Join(dir, "main.go"), "package main")
			},
			wantGone: []string{".DS_Store"},
			wantKeep: []string{"main.go"},
		},
		{
			name: "removes all excludes at once",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mustMkdir(t, filepath.Join(dir, ".git"))
				mustWriteFile(t, filepath.Join(dir, "go.sum"), "checksum")
				mustMkdir(t, filepath.Join(dir, "vendor"))
				mustMkdir(t, filepath.Join(dir, "bin"))
				mustWriteFile(t, filepath.Join(dir, ".DS_Store"), "dsstore")
				mustWriteFile(t, filepath.Join(dir, "go.mod"), "module test")
				mustMkdir(t, filepath.Join(dir, "internal"))
				mustWriteFile(t, filepath.Join(dir, "internal", "app.go"), "package internal")
			},
			wantGone: []string{".git", "go.sum", "vendor", "bin", ".DS_Store"},
			wantKeep: []string{"go.mod", "internal"},
		},
		{
			name: "no error when excludes do not exist",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mustWriteFile(t, filepath.Join(dir, "main.go"), "package main")
			},
			wantGone: []string{},
			wantKeep: []string{"main.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(t, tmpDir)

			err := Clean(tmpDir, DefaultExcludes)
			if err != nil {
				t.Fatalf("Clean() returned unexpected error: %v", err)
			}

			for _, gone := range tt.wantGone {
				p := filepath.Join(tmpDir, gone)
				if _, err := os.Stat(p); !os.IsNotExist(err) {
					t.Errorf("expected %q to be removed, but it still exists", gone)
				}
			}

			for _, keep := range tt.wantKeep {
				p := filepath.Join(tmpDir, keep)
				if _, err := os.Stat(p); os.IsNotExist(err) {
					t.Errorf("expected %q to be kept, but it was removed", keep)
				}
			}
		})
	}
}

// --- Test helpers ---

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create directory for file %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}
