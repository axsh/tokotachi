package clihelp

import (
	"fmt"
	"strings"
)

// Page is one agent-oriented help topic (English Markdown).
type Page struct {
	Title          string // e.g. "tt up", "tt prompt update"
	Purpose        string
	WhenToUse      string
	WhenNotToUse   string
	Preconditions  string
	InputAndOutput string
	ExitCodes      string
	ReadNext       string
}

// RequiredHeadings lists the fixed section titles in order.
var RequiredHeadings = []string{
	"Purpose",
	"When to use",
	"When not to use",
	"Preconditions",
	"Input and output",
	"Exit codes",
	"Read next",
}

// Validate reports an error when Title or any section body is empty.
func Validate(p Page) error {
	if strings.TrimSpace(p.Title) == "" {
		return fmt.Errorf("clihelp: Title is required")
	}
	fields := []struct {
		name string
		val  string
	}{
		{"Purpose", p.Purpose},
		{"When to use", p.WhenToUse},
		{"When not to use", p.WhenNotToUse},
		{"Preconditions", p.Preconditions},
		{"Input and output", p.InputAndOutput},
		{"Exit codes", p.ExitCodes},
		{"Read next", p.ReadNext},
	}
	for _, f := range fields {
		if strings.TrimSpace(f.val) == "" {
			return fmt.Errorf("clihelp: %s is required for %q", f.name, p.Title)
		}
	}
	return nil
}

// Render returns Markdown with the fixed seven headings in order.
func Render(p Page) string {
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(strings.TrimSpace(p.Title))
	b.WriteString("\n\n")
	sections := []struct {
		heading string
		body    string
	}{
		{"Purpose", p.Purpose},
		{"When to use", p.WhenToUse},
		{"When not to use", p.WhenNotToUse},
		{"Preconditions", p.Preconditions},
		{"Input and output", p.InputAndOutput},
		{"Exit codes", p.ExitCodes},
		{"Read next", p.ReadNext},
	}
	for _, s := range sections {
		b.WriteString("## ")
		b.WriteString(s.heading)
		b.WriteString("\n\n")
		b.WriteString(strings.TrimSpace(s.body))
		b.WriteString("\n\n")
	}
	return b.String()
}

// ContainsRequiredHeadings reports whether s includes all ## headings in order.
func ContainsRequiredHeadings(s string) bool {
	pos := 0
	for _, h := range RequiredHeadings {
		marker := "## " + h
		i := strings.Index(s[pos:], marker)
		if i < 0 {
			return false
		}
		pos += i + len(marker)
	}
	return true
}
