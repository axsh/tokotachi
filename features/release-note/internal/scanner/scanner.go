package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var phaseRe = regexp.MustCompile(`^(\d{3})-(.+)$`)

// PhaseInfo represents a phase directory.
type PhaseInfo struct {
	Number  int    // e.g. 0, 1, 2
	Name    string // e.g. "foundation"
	DirName string // e.g. "000-foundation"
}

// BranchSpec represents a found specification folder for a branch.
type BranchSpec struct {
	BranchName string   // e.g. "feat-xxx"
	PhaseName  string   // e.g. "000-foundation"
	FolderPath string   // absolute path to the ideas/{branch} folder
	Files      []string // list of .md file paths in the folder
}

// Scanner searches for specification folders.
type Scanner struct {
	phasesRoot string
}

// NewScanner creates a new Scanner for the given phases root directory.
func NewScanner(phasesRoot string) *Scanner {
	return &Scanner{phasesRoot: phasesRoot}
}

// ListPhases reads prompts/phases/ and returns phases sorted descending
// by phase number.
func (s *Scanner) ListPhases() ([]PhaseInfo, error) {
	entries, err := os.ReadDir(s.phasesRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to read phases directory: %w", err)
	}

	var phases []PhaseInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		matches := phaseRe.FindStringSubmatch(entry.Name())
		if len(matches) < 3 {
			continue
		}
		num, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}
		phases = append(phases, PhaseInfo{
			Number:  num,
			Name:    matches[2],
			DirName: entry.Name(),
		})
	}

	// Sort descending by number
	sort.Slice(phases, func(i, j int) bool {
		return phases[i].Number > phases[j].Number
	})

	return phases, nil
}

// FindSpecFolders finds spec folders for the given branch names.
// Searches from the highest phase number downward. Stops searching
// for a branch when the phase number reaches 000 or (maxPhase - 5).
func (s *Scanner) FindSpecFolders(branches []string) ([]BranchSpec, error) {
	phases, err := s.ListPhases()
	if err != nil {
		return nil, err
	}

	if len(phases) == 0 {
		return nil, nil
	}

	maxPhase := phases[0].Number
	lowerBound := maxPhase - 5
	if lowerBound < 0 {
		lowerBound = 0
	}

	var results []BranchSpec

	for _, branch := range branches {
		spec := s.findBranchInPhases(branch, phases, lowerBound)
		if spec != nil {
			results = append(results, *spec)
		}
	}

	return results, nil
}

// findBranchInPhases searches for a branch folder across all phases
// from highest to lowest, stopping at lowerBound.
func (s *Scanner) findBranchInPhases(branch string, phases []PhaseInfo, lowerBound int) *BranchSpec {
	for _, phase := range phases {
		if phase.Number < lowerBound {
			break
		}

		ideasDir := filepath.Join(s.phasesRoot, phase.DirName, "ideas", branch)
		info, err := os.Stat(ideasDir)
		if err != nil || !info.IsDir() {
			continue
		}

		// Found the branch folder, collect .md files
		files, err := collectMdFiles(ideasDir)
		if err != nil {
			continue
		}

		if len(files) == 0 {
			continue
		}

		return &BranchSpec{
			BranchName: branch,
			PhaseName:  phase.DirName,
			FolderPath: ideasDir,
			Files:      files,
		}
	}

	return nil
}

// collectMdFiles returns all .md file paths in the given directory.
func collectMdFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	sort.Strings(files)
	return files, nil
}
