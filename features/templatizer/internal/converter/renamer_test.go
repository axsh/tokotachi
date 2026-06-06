package converter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenameDirectories(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, dir string)
		oldName string
		newName string
		wantDir string // expected directory to exist after rename
	}{
		{
			name: "renames cmd/old-app to cmd/new-app",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mustMkdir(t, filepath.Join(dir, "cmd", "old-app"))
				mustWriteFile(t, filepath.Join(dir, "cmd", "old-app", "main.go"), "package main")
			},
			oldName: "old-app",
			newName: "new-app",
			wantDir: "cmd/new-app",
		},
		{
			name: "preserves file content after rename",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mustMkdir(t, filepath.Join(dir, "cmd", "old-app"))
				mustWriteFile(t, filepath.Join(dir, "cmd", "old-app", "main.go"), "package main\nfunc main() {}")
			},
			oldName: "old-app",
			newName: "new-app",
			wantDir: "cmd/new-app",
		},
		{
			name: "no error when cmd directory does not exist",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mustWriteFile(t, filepath.Join(dir, "main.go"), "package main")
			},
			oldName: "old-app",
			newName: "new-app",
			wantDir: "",
		},
		{
			name: "no error when cmd/old-name does not exist",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				mustMkdir(t, filepath.Join(dir, "cmd", "other-app"))
			},
			oldName: "old-app",
			newName: "new-app",
			wantDir: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(t, tmpDir)

			err := RenameDirectories(tmpDir, tt.oldName, tt.newName)
			if err != nil {
				t.Fatalf("RenameDirectories() error: %v", err)
			}

			if tt.wantDir != "" {
				wantPath := filepath.Join(tmpDir, tt.wantDir)
				if _, err := os.Stat(wantPath); os.IsNotExist(err) {
					t.Errorf("expected directory %q to exist, but it does not", tt.wantDir)
				}

				// Verify old directory is gone.
				oldPath := filepath.Join(tmpDir, "cmd", tt.oldName)
				if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
					t.Errorf("expected old directory %q to be removed", "cmd/"+tt.oldName)
				}
			}
		})
	}
}
