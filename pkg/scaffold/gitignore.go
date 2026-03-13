package scaffold

import (
	"os"
	"strings"
)

// Gitignore manages .gitignore file entries with syntax-aware operations.
type Gitignore struct {
	lines []string
}

// LoadGitignore reads a .gitignore file. Returns an empty Gitignore if the file does not exist.
func LoadGitignore(path string) (*Gitignore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Gitignore{}, nil
		}
		return nil, err
	}

	content := string(data)
	// Remove trailing newline to avoid empty last element
	content = strings.TrimRight(content, "\n")
	var lines []string
	if content != "" {
		lines = strings.Split(content, "\n")
	}
	return &Gitignore{lines: lines}, nil
}

// Save writes the gitignore content to a file.
// Ensures a trailing newline if there are entries.
func (g *Gitignore) Save(path string) error {
	if len(g.lines) == 0 {
		return os.WriteFile(path, nil, 0o644)
	}
	content := strings.Join(g.lines, "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0o644)
}

// AddEntries adds entries to the gitignore, skipping duplicates.
// Returns the number of entries actually added.
// Comparison is done after trimming trailing whitespace on existing lines.
func (g *Gitignore) AddEntries(entries []string) int {
	existing := g.effectiveEntrySet()
	added := 0
	for _, entry := range entries {
		trimmed := strings.TrimRight(entry, " \t")
		if trimmed == "" {
			continue
		}
		if _, ok := existing[trimmed]; !ok {
			g.lines = append(g.lines, entry)
			existing[trimmed] = true
			added++
		}
	}
	return added
}

// RemoveEntries removes entries matching the given patterns (exact match after trim).
// Returns the number of entries removed.
func (g *Gitignore) RemoveEntries(entries []string) int {
	toRemove := make(map[string]bool, len(entries))
	for _, e := range entries {
		toRemove[strings.TrimRight(e, " \t")] = true
	}

	removed := 0
	filtered := make([]string, 0, len(g.lines))
	for _, line := range g.lines {
		trimmed := strings.TrimRight(line, " \t")
		if isEffectiveLine(trimmed) && toRemove[trimmed] {
			removed++
			continue
		}
		filtered = append(filtered, line)
	}
	g.lines = filtered
	return removed
}

// HasEntry checks if the given entry exists among effective (non-comment, non-empty) lines.
func (g *Gitignore) HasEntry(entry string) bool {
	trimmed := strings.TrimSpace(entry)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return false
	}
	for _, line := range g.lines {
		lineTrimmed := strings.TrimSpace(line)
		if lineTrimmed == trimmed {
			return true
		}
	}
	return false
}

// effectiveEntrySet returns a set of all effective (non-comment, non-empty) entries.
func (g *Gitignore) effectiveEntrySet() map[string]bool {
	set := make(map[string]bool, len(g.lines))
	for _, line := range g.lines {
		trimmed := strings.TrimRight(line, " \t")
		if isEffectiveLine(trimmed) {
			set[trimmed] = true
		}
	}
	return set
}

// isEffectiveLine returns true if the line is a real gitignore pattern
// (not a comment and not empty).
func isEffectiveLine(line string) bool {
	return line != "" && !strings.HasPrefix(line, "#")
}
