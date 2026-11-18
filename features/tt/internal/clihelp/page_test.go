package clihelp

import (
	"strings"
	"testing"
)

func samplePage() Page {
	return Page{
		Title:          "tt up",
		Purpose:        "Start a container.",
		WhenToUse:      "When the worktree exists.",
		WhenNotToUse:   "When you only need status.",
		Preconditions:  "Docker is available.",
		InputAndOutput: "`tt up <feature>`",
		ExitCodes:      "0 success\n1 failure",
		ReadNext:       "`tt help down`",
	}
}

func TestRender_SectionOrder(t *testing.T) {
	out := Render(samplePage())
	if !ContainsRequiredHeadings(out) {
		t.Fatalf("missing headings in order:\n%s", out)
	}
	for _, h := range RequiredHeadings {
		marker := "## " + h
		if !strings.Contains(out, marker) {
			t.Errorf("missing %q", marker)
		}
	}
}

func TestRender_TitleHeading(t *testing.T) {
	out := Render(samplePage())
	if !strings.HasPrefix(out, "# tt up\n") {
		t.Fatalf("title heading missing, got:\n%s", out)
	}
}

func TestValidate_RejectsEmpty(t *testing.T) {
	p := samplePage()
	p.Purpose = ""
	if err := Validate(p); err == nil {
		t.Fatal("expected error for empty Purpose")
	}
}

func TestHasRequiredHeadings_DetectsGap(t *testing.T) {
	if ContainsRequiredHeadings("# x\n\n## Purpose\n\nok\n") {
		t.Fatal("expected false when headings incomplete")
	}
}
