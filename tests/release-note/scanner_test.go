package release_note_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

var phasePattern = regexp.MustCompile(`^\d{3}-.+$`)

func TestScanner_RealPhaseStructure(t *testing.T) {
	phasesDir := filepath.Join(projectRoot(), "prompts", "phases")

	// Verify phases directory exists
	if _, err := os.Stat(phasesDir); os.IsNotExist(err) {
		t.Fatalf("prompts/phases/ directory not found at %s", phasesDir)
	}

	// List phase directories
	entries, err := os.ReadDir(phasesDir)
	if err != nil {
		t.Fatalf("failed to read phases directory: %v", err)
	}

	// Find phase directories matching NNN-name pattern
	var phases []string
	for _, entry := range entries {
		if entry.IsDir() && phasePattern.MatchString(entry.Name()) {
			phases = append(phases, entry.Name())
		}
	}

	if len(phases) == 0 {
		t.Fatal("no phase directories found matching NNN-name pattern")
	}

	// Verify "000-foundation" exists
	found := false
	for _, p := range phases {
		if p == "000-foundation" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected '000-foundation' phase directory not found")
	}

	// Verify "000-foundation/ideas/" subdirectory exists
	ideasDir := filepath.Join(phasesDir, "000-foundation", "ideas")
	if _, err := os.Stat(ideasDir); os.IsNotExist(err) {
		t.Fatalf("000-foundation/ideas/ directory not found at %s", ideasDir)
	}
}

func TestScanner_FindBranchFolder(t *testing.T) {
	branchDir := filepath.Join(
		projectRoot(), "prompts", "phases",
		"000-foundation", "ideas", "fix-module-versioning",
	)

	// Verify branch folder exists
	if _, err := os.Stat(branchDir); os.IsNotExist(err) {
		t.Fatalf("branch folder not found at %s", branchDir)
	}

	// Verify at least one .md file exists
	entries, err := os.ReadDir(branchDir)
	if err != nil {
		t.Fatalf("failed to read branch directory: %v", err)
	}

	mdCount := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			mdCount++
		}
	}

	if mdCount == 0 {
		t.Fatal("no .md files found in branch folder")
	}

	t.Logf("Found %d .md file(s) in fix-module-versioning branch folder", mdCount)
}
