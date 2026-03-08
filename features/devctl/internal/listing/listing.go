package listing

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/axsh/tokotachi/features/devctl/internal/state"
)

// WorktreeEntry represents a parsed git worktree entry.
type WorktreeEntry struct {
	Path   string // absolute path
	Branch string // branch name (from "branch refs/heads/<name>")
	Bare   bool   // true if main worktree (bare)
}

// FeatureInfo holds feature name and status for display.
type FeatureInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// BranchInfo holds the merged branch overview.
type BranchInfo struct {
	Branch       string        `json:"branch"`
	Path         string        `json:"path"`
	Features     []FeatureInfo `json:"features"`
	MainWorktree bool          `json:"main_worktree,omitempty"`
}

// ParseWorktreeOutput parses `git worktree list --porcelain` output.
// Each worktree block is separated by an empty line.
func ParseWorktreeOutput(output string) []WorktreeEntry {
	if strings.TrimSpace(output) == "" {
		return nil
	}

	blocks := splitBlocks(output)
	entries := make([]WorktreeEntry, 0, len(blocks))

	for _, block := range blocks {
		entry := parseBlock(block)
		if entry.Path != "" {
			entries = append(entries, entry)
		}
	}

	if len(entries) == 0 {
		return nil
	}
	return entries
}

// splitBlocks splits porcelain output into individual worktree blocks.
func splitBlocks(output string) []string {
	// Normalize line endings and split by double newline
	output = strings.ReplaceAll(output, "\r\n", "\n")
	raw := strings.Split(output, "\n\n")
	var blocks []string
	for _, b := range raw {
		trimmed := strings.TrimSpace(b)
		if trimmed != "" {
			blocks = append(blocks, trimmed)
		}
	}
	return blocks
}

// parseBlock parses a single porcelain block into a WorktreeEntry.
func parseBlock(block string) WorktreeEntry {
	var entry WorktreeEntry
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "worktree "):
			entry.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "branch refs/heads/"):
			entry.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		case line == "bare":
			entry.Bare = true
		}
	}
	return entry
}

// CollectBranches merges worktree entries with state files.
// The result is sorted alphabetically by branch name.
func CollectBranches(entries []WorktreeEntry, states map[string]state.StateFile) []BranchInfo {
	branches := make([]BranchInfo, 0, len(entries))

	for _, e := range entries {
		bi := BranchInfo{
			Branch:       e.Branch,
			Path:         e.Path,
			MainWorktree: e.Bare,
			Features:     []FeatureInfo{}, // never nil for JSON
		}

		if sf, ok := states[e.Branch]; ok {
			features := make([]FeatureInfo, 0, len(sf.Features))
			for name, fs := range sf.Features {
				features = append(features, FeatureInfo{
					Name:   name,
					Status: string(fs.Status),
				})
			}
			// Sort features by name for deterministic output
			sort.Slice(features, func(i, j int) bool {
				return features[i].Name < features[j].Name
			})
			bi.Features = features
		}

		branches = append(branches, bi)
	}

	// Sort by branch name
	sort.Slice(branches, func(i, j int) bool {
		return branches[i].Branch < branches[j].Branch
	})

	return branches
}

// featureColumn builds a display string for the FEATURE column.
// Returns feature names (comma-separated if multiple), or "-" if none.
func featureColumn(bi BranchInfo) string {
	if bi.MainWorktree || len(bi.Features) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(bi.Features))
	for _, f := range bi.Features {
		parts = append(parts, f.Name)
	}
	return strings.Join(parts, ", ")
}

// stateColumn builds a display string for the STATE column.
// Returns the status of features, "(no state)", or "(main worktree)".
func stateColumn(bi BranchInfo) string {
	if bi.MainWorktree {
		return "(main worktree)"
	}
	if len(bi.Features) == 0 {
		return "(no state)"
	}
	parts := make([]string, 0, len(bi.Features))
	for _, f := range bi.Features {
		parts = append(parts, f.Status)
	}
	return strings.Join(parts, ", ")
}

// FormatTable writes branch info as a human-readable table.
// Columns: BRANCH, FEATURE, STATE, (PATH if showPath is true).
func FormatTable(w io.Writer, branches []BranchInfo, showPath bool) {
	if showPath {
		fmt.Fprintf(w, "%-24s %-20s %-20s %s\n", "BRANCH", "FEATURE", "STATE", "PATH")
	} else {
		fmt.Fprintf(w, "%-24s %-20s %s\n", "BRANCH", "FEATURE", "STATE")
	}

	for _, bi := range branches {
		feat := featureColumn(bi)
		st := stateColumn(bi)
		if showPath {
			fmt.Fprintf(w, "%-24s %-20s %-20s %s\n", bi.Branch, feat, st, bi.Path)
		} else {
			fmt.Fprintf(w, "%-24s %-20s %s\n", bi.Branch, feat, st)
		}
	}
}

// FormatJSON writes branch info as indented JSON.
func FormatJSON(w io.Writer, branches []BranchInfo) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(branches)
}
