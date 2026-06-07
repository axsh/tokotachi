package notify

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"golang.org/x/text/unicode/norm"
)

var whitespaceRe = regexp.MustCompile(`\s+`)

// NormalizePayload normalizes text fields in-place:
// 1. NFC normalization for task_summary and each raw_note
// 2. Trim leading/trailing whitespace
// 3. Compress consecutive whitespace to single space
// 4. Remove empty notes from raw_notes after normalization
// Returns an error if raw_notes becomes empty after normalization.
func NormalizePayload(p *agent.NotifyPayload) error {
	p.TaskSummary = normalizeText(p.TaskSummary)

	var filtered []string
	for _, note := range p.RawNotes {
		normalized := normalizeText(note)
		if normalized != "" {
			filtered = append(filtered, normalized)
		}
	}
	p.RawNotes = filtered

	if len(p.RawNotes) == 0 {
		return fmt.Errorf("raw_notes is empty after normalization: all notes were blank or whitespace-only")
	}

	return nil
}

// normalizeText applies NFC normalization, trims whitespace, and compresses
// consecutive whitespace into a single space.
func normalizeText(s string) string {
	s = norm.NFC.String(s)
	s = whitespaceRe.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	return s
}
