package listing

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/state"
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
	Branch       string            `json:"branch"`
	Path         string            `json:"path"`
	Features     []FeatureInfo     `json:"features"`
	MainWorktree bool              `json:"main_worktree,omitempty"`
	CodeStatus   *state.CodeStatus `json:"code_status,omitempty"`
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
			bi.CodeStatus = sf.CodeStatus
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

// containerColumn builds a display string for the CONTAINER column.
// Returns the status of features, "(no state)", or "(main worktree)".
func containerColumn(bi BranchInfo) string {
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

// FormatCodeColumn builds a display string for the CODE column.
// The now parameter is injected for testability.
func FormatCodeColumn(bi BranchInfo, now time.Time) string {
	if bi.MainWorktree {
		return "-"
	}
	if bi.CodeStatus == nil {
		return "(unknown)"
	}
	switch bi.CodeStatus.Status {
	case state.CodeStatusLocal:
		return "(local)"
	case state.CodeStatusHosted:
		return "hosted"
	case state.CodeStatusDeleted:
		return "deleted"
	case state.CodeStatusPR:
		if bi.CodeStatus.PRCreatedAt == nil {
			return "PR"
		}
		return formatPRElapsed(now, *bi.CodeStatus.PRCreatedAt)
	default:
		return string(bi.CodeStatus.Status)
	}
}

// formatPRElapsed formats PR elapsed time.
func formatPRElapsed(now, createdAt time.Time) string {
	d := now.Sub(createdAt)
	switch {
	case d < 60*time.Minute:
		return fmt.Sprintf("PR(%dm ago)", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("PR(%dh ago)", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("PR(%dd ago)", int(d.Hours()/24))
	default:
		return fmt.Sprintf("PR(%02d/%02d)", createdAt.Month(), createdAt.Day())
	}
}

// TableOptions controls the table output format.
type TableOptions struct {
	ShowPath bool // --path: show PATH column
	Full     bool // --full: disable truncation
	MaxWidth int  // TT_LIST_WIDTH (default: 40)
	Padding  int  // TT_LIST_PADDING (default: 2)
}

// TruncateCell truncates s if len(s) > maxWidth.
// Returns s[:maxWidth-4] + " ..." when truncated.
func TruncateCell(s string, maxWidth int) string {
	if maxWidth <= 0 || len(s) <= maxWidth {
		return s
	}
	cutAt := maxWidth - 4
	if cutAt < 0 {
		cutAt = 0
	}
	return s[:cutAt] + " ..."
}

// FormatTable writes branch info as a human-readable table with dynamic column widths.
// Columns: BRANCH, FEATURE, CONTAINER, CODE, (PATH if opts.ShowPath is true).
func FormatTable(w io.Writer, branches []BranchInfo, opts TableOptions) {
	// Apply defaults for zero values
	if opts.MaxWidth <= 0 {
		opts.MaxWidth = 40
	}
	if opts.Padding <= 0 {
		opts.Padding = 2
	}

	now := time.Now()

	// Build header
	headers := []string{"BRANCH", "FEATURE", "CONTAINER", "CODE"}
	if opts.ShowPath {
		headers = append(headers, "PATH")
	}
	numCols := len(headers)

	// Build cell data for each row
	rows := make([][]string, len(branches))
	for i, bi := range branches {
		feat := featureColumn(bi)
		ct := containerColumn(bi)
		code := FormatCodeColumn(bi, now)

		row := []string{bi.Branch, feat, ct, code}
		if opts.ShowPath {
			row = append(row, bi.Path)
		}

		// Apply truncation (unless --full)
		if !opts.Full {
			for j := range row {
				row[j] = TruncateCell(row[j], opts.MaxWidth)
			}
		}

		rows[i] = row
	}

	// Calculate column widths from headers and cell data
	widths := make([]int, numCols)
	for j, h := range headers {
		widths[j] = len(h)
	}
	for _, row := range rows {
		for j, cell := range row {
			if len(cell) > widths[j] {
				widths[j] = len(cell)
			}
		}
	}

	// Print header
	for j, h := range headers {
		if j < numCols-1 {
			fmt.Fprintf(w, "%-*s", widths[j]+opts.Padding, h)
		} else {
			fmt.Fprint(w, h)
		}
	}
	fmt.Fprint(w, "\n")

	// Print rows
	for _, row := range rows {
		for j, cell := range row {
			if j < numCols-1 {
				fmt.Fprintf(w, "%-*s", widths[j]+opts.Padding, cell)
			} else {
				fmt.Fprint(w, cell)
			}
		}
		fmt.Fprint(w, "\n")
	}
}

// FormatJSON writes branch info as indented JSON.
func FormatJSON(w io.Writer, branches []BranchInfo) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(branches)
}
