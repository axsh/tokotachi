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

	// Verify "000-foundation/branches/" subdirectory exists
	branchesDir := filepath.Join(phasesDir, "000-foundation", "branches")
	if _, err := os.Stat(branchesDir); os.IsNotExist(err) {
		t.Fatalf("000-foundation/branches/ directory not found at %s", branchesDir)
	}
}

func TestScanner_FindBranchFolder(t *testing.T) {
	branchesDir := filepath.Join(
		projectRoot(), "prompts", "phases",
		"000-foundation", "branches",
	)

	// Find the first branch directory under branches/
	entries, err := os.ReadDir(branchesDir)
	if err != nil {
		t.Fatalf("failed to read branches directory: %v", err)
	}

	var branchName string
	for _, entry := range entries {
		if entry.IsDir() {
			branchName = entry.Name()
			break
		}
	}
	if branchName == "" {
		t.Fatal("no branch directory found under branches/")
	}

	branchDir := filepath.Join(branchesDir, branchName, "ideas")

	// Verify branch ideas folder exists
	if _, err := os.Stat(branchDir); os.IsNotExist(err) {
		t.Fatalf("branch ideas folder not found at %s", branchDir)
	}

	// Verify at least one .md file exists
	ideaEntries, err := os.ReadDir(branchDir)
	if err != nil {
		t.Fatalf("failed to read branch ideas directory: %v", err)
	}

	mdCount := 0
	for _, entry := range ideaEntries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			mdCount++
		}
	}

	if mdCount == 0 {
		t.Fatal("no .md files found in branch ideas folder")
	}

	t.Logf("Found %d .md file(s) in %s branch ideas folder", mdCount, branchName)
}
