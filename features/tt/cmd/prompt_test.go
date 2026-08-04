package cmd

import (
	"strings"
	"testing"
)

func TestPromptPathHelpConstants_ContainPathExpressions(t *testing.T) {
	for _, s := range PromptPathHelpConstants() {
		if !strings.Contains(s, "Default:") {
			t.Errorf("help constant missing Default: prefix: %q", s)
		}
	}
	if !PromptPathHelpContainsPathExpr(helpProject) {
		t.Errorf("helpProject should contain path expression, got: %q", helpProject)
	}
	if !PromptPathHelpContainsPathExpr(helpPromptsDir) {
		t.Errorf("helpPromptsDir should contain path expression, got: %q", helpPromptsDir)
	}
}
