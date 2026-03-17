package scanner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/axsh/tokotachi/features/release-note/internal/scanner"
)

func TestParsePhases(t *testing.T) {
	dir := t.TempDir()

	// Create phase directories
	os.MkdirAll(filepath.Join(dir, "000-foundation", "ideas"), 0o755)
	os.MkdirAll(filepath.Join(dir, "001-webservices", "ideas"), 0o755)

	s := scanner.NewScanner(dir)
	phases, err := s.ListPhases()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(phases) != 2 {
		t.Fatalf("expected 2 phases, got %d", len(phases))
	}

	// Should be sorted descending by number
	if phases[0].Number != 1 {
		t.Errorf("expected first phase number 1, got %d", phases[0].Number)
	}
	if phases[0].Name != "webservices" {
		t.Errorf("expected first phase name 'webservices', got '%s'", phases[0].Name)
	}
	if phases[1].Number != 0 {
		t.Errorf("expected second phase number 0, got %d", phases[1].Number)
	}
}

func TestFindSpecFolders_DirectMatch(t *testing.T) {
	dir := t.TempDir()

	// Create phase with a branch folder containing a spec file
	branchDir := filepath.Join(dir, "000-foundation", "ideas", "feat-xxx")
	os.MkdirAll(branchDir, 0o755)
	os.WriteFile(filepath.Join(branchDir, "000-Feature.md"), []byte("# Feature"), 0o644)

	s := scanner.NewScanner(dir)
	specs, err := s.FindSpecFolders([]string{"feat-xxx"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(specs))
	}
	if specs[0].BranchName != "feat-xxx" {
		t.Errorf("expected branch 'feat-xxx', got '%s'", specs[0].BranchName)
	}
	if len(specs[0].Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(specs[0].Files))
	}
}

func TestFindSpecFolders_FallbackPhase(t *testing.T) {
	dir := t.TempDir()

	// Create two phases, branch only exists in the lower phase
	os.MkdirAll(filepath.Join(dir, "001-webservices", "ideas"), 0o755)
	branchDir := filepath.Join(dir, "000-foundation", "ideas", "feat-xxx")
	os.MkdirAll(branchDir, 0o755)
	os.WriteFile(filepath.Join(branchDir, "000-Feature.md"), []byte("# Feature"), 0o644)

	s := scanner.NewScanner(dir)
	specs, err := s.FindSpecFolders([]string{"feat-xxx"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(specs) != 1 {
		t.Fatalf("expected 1 spec via fallback, got %d", len(specs))
	}
	if specs[0].PhaseName != "000-foundation" {
		t.Errorf("expected phase '000-foundation', got '%s'", specs[0].PhaseName)
	}
}

func TestFindSpecFolders_NotFound(t *testing.T) {
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "000-foundation", "ideas"), 0o755)

	s := scanner.NewScanner(dir)
	specs, err := s.FindSpecFolders([]string{"nonexistent-branch"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(specs) != 0 {
		t.Errorf("expected 0 specs for nonexistent branch, got %d", len(specs))
	}
}

func TestFindSpecFolders_MultipleBranches(t *testing.T) {
	dir := t.TempDir()

	// Create phase with multiple branches
	for _, branch := range []string{"feat-a", "feat-b"} {
		branchDir := filepath.Join(dir, "000-foundation", "ideas", branch)
		os.MkdirAll(branchDir, 0o755)
		os.WriteFile(filepath.Join(branchDir, "000-Spec.md"), []byte("# Spec"), 0o644)
	}

	s := scanner.NewScanner(dir)
	specs, err := s.FindSpecFolders([]string{"feat-a", "feat-b", "feat-c"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(specs) != 2 {
		t.Errorf("expected 2 specs, got %d", len(specs))
	}
}

func TestFindSpecFolders_IgnoresNonMdFiles(t *testing.T) {
	dir := t.TempDir()

	branchDir := filepath.Join(dir, "000-foundation", "ideas", "feat-xxx")
	os.MkdirAll(branchDir, 0o755)
	os.WriteFile(filepath.Join(branchDir, "000-Spec.md"), []byte("# Spec"), 0o644)
	os.WriteFile(filepath.Join(branchDir, ".gitkeep"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(branchDir, "notes.txt"), []byte("notes"), 0o644)

	s := scanner.NewScanner(dir)
	specs, err := s.FindSpecFolders([]string{"feat-xxx"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(specs))
	}
	if len(specs[0].Files) != 1 {
		t.Errorf("expected 1 .md file, got %d files", len(specs[0].Files))
	}
}
