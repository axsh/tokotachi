package emitter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileWithMode_Overwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")

	// Pre-create file with old content
	if err := os.WriteFile(path, []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Overwrite mode should replace content
	if err := writeFileWithMode(path, "new content", EmitModeOverwrite); err != nil {
		t.Fatalf("writeFileWithMode() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new content" {
		t.Errorf("expected 'new content', got %q", string(data))
	}
}

func TestWriteFileWithMode_Skip_Existing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")

	// Pre-create file with old content
	if err := os.WriteFile(path, []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Skip mode should NOT replace existing content
	if err := writeFileWithMode(path, "new content", EmitModeSkip); err != nil {
		t.Fatalf("writeFileWithMode() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old content" {
		t.Errorf("expected 'old content' (unchanged), got %q", string(data))
	}
}

func TestWriteFileWithMode_Skip_New(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")

	// Skip mode should create new file if it doesn't exist
	if err := writeFileWithMode(path, "new content", EmitModeSkip); err != nil {
		t.Fatalf("writeFileWithMode() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new content" {
		t.Errorf("expected 'new content', got %q", string(data))
	}
}

func TestWriteFileWithMode_Immune(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")

	// Pre-create file with old content
	if err := os.WriteFile(path, []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Immune mode writes same as overwrite (orphan cleanup is separate)
	if err := writeFileWithMode(path, "new content", EmitModeImmune); err != nil {
		t.Fatalf("writeFileWithMode() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new content" {
		t.Errorf("expected 'new content', got %q", string(data))
	}
}

func TestCleanOrphanFiles_RemovesOrphans(t *testing.T) {
	dir := t.TempDir()

	// Create emitted file
	emittedPath := filepath.Join(dir, "emitted.md")
	if err := os.WriteFile(emittedPath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create orphan file
	orphanPath := filepath.Join(dir, "orphan.md")
	if err := os.WriteFile(orphanPath, []byte("orphan"), 0644); err != nil {
		t.Fatal(err)
	}

	emittedFiles := map[string]bool{
		filepath.Clean(emittedPath): true,
	}

	orphans, err := CleanOrphanFiles([]string{dir}, emittedFiles, false)
	if err != nil {
		t.Fatalf("CleanOrphanFiles() error = %v", err)
	}

	// Verify orphan was found
	if len(orphans) != 1 {
		t.Fatalf("expected 1 orphan, got %d", len(orphans))
	}

	// Verify orphan was removed
	if _, err := os.Stat(orphanPath); !os.IsNotExist(err) {
		t.Error("orphan file should have been removed")
	}

	// Verify emitted file still exists
	if _, err := os.Stat(emittedPath); os.IsNotExist(err) {
		t.Error("emitted file should still exist")
	}
}

func TestCleanOrphanFiles_DryRun(t *testing.T) {
	dir := t.TempDir()

	// Create orphan file
	orphanPath := filepath.Join(dir, "orphan.md")
	if err := os.WriteFile(orphanPath, []byte("orphan"), 0644); err != nil {
		t.Fatal(err)
	}

	emittedFiles := map[string]bool{}

	orphans, err := CleanOrphanFiles([]string{dir}, emittedFiles, true)
	if err != nil {
		t.Fatalf("CleanOrphanFiles() error = %v", err)
	}

	// Verify orphan was found
	if len(orphans) != 1 {
		t.Fatalf("expected 1 orphan, got %d", len(orphans))
	}

	// Verify orphan was NOT removed (dry-run)
	if _, err := os.Stat(orphanPath); os.IsNotExist(err) {
		t.Error("orphan file should NOT be removed in dry-run mode")
	}
}

func TestCleanOrphanFiles_EmptyDir(t *testing.T) {
	// Non-existent directory should not cause an error
	orphans, err := CleanOrphanFiles([]string{"/nonexistent/path"}, map[string]bool{}, false)
	if err != nil {
		t.Fatalf("CleanOrphanFiles() error = %v", err)
	}
	if len(orphans) != 0 {
		t.Errorf("expected 0 orphans, got %d", len(orphans))
	}
}

func TestCleanOrphanFiles_SkipsGitkeep(t *testing.T) {
	dir := t.TempDir()

	// Create .gitkeep (should be ignored)
	gitkeepPath := filepath.Join(dir, ".gitkeep")
	if err := os.WriteFile(gitkeepPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	// Create README.md (should also be ignored)
	readmePath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readmePath, []byte("readme"), 0644); err != nil {
		t.Fatal(err)
	}

	orphans, err := CleanOrphanFiles([]string{dir}, map[string]bool{}, false)
	if err != nil {
		t.Fatalf("CleanOrphanFiles() error = %v", err)
	}
	if len(orphans) != 0 {
		t.Errorf("expected 0 orphans (gitkeep/README should be skipped), got %d", len(orphans))
	}

	// Verify files still exist
	if _, err := os.Stat(gitkeepPath); os.IsNotExist(err) {
		t.Error(".gitkeep should still exist")
	}
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Error("README.md should still exist")
	}
}

func TestValidEmitModes(t *testing.T) {
	tests := []struct {
		mode EmitMode
		want bool
	}{
		{EmitModeOverwrite, true},
		{EmitModeImmune, true},
		{EmitModeSkip, true},
		{EmitMode("unknown"), false},
		{EmitMode(""), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			got := ValidEmitModes(tt.mode)
			if got != tt.want {
				t.Errorf("ValidEmitModes(%q) = %v, want %v", tt.mode, got, tt.want)
			}
		})
	}
}
