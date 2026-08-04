package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
)

func newTagTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	addPromptTagFlags(cmd)
	return cmd
}

func TestResolveTagsFlag_DefaultsAndEnv(t *testing.T) {
	t.Setenv(manifest.EnvKeyTags, "")
	promptTags = ""
	cmd := newTagTestCmd()
	tags, warnings, err := resolveTagsFlag(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if len(tags) != 1 || tags[0] != manifest.BaselineTag {
		t.Fatalf("got %v, want [baseline]", tags)
	}

	t.Setenv(manifest.EnvKeyTags, "test")
	promptTags = ""
	cmd = newTagTestCmd()
	tags, _, err = resolveTagsFlag(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 1 || tags[0] != "test" {
		t.Fatalf("got %v, want [test] from env", tags)
	}
}

func TestResolveTagsFlag_CLIOverridesEnv(t *testing.T) {
	t.Setenv(manifest.EnvKeyTags, "from-env")
	cmd := newTagTestCmd()
	if err := cmd.Flags().Set("tags", "baseline, test"); err != nil {
		t.Fatal(err)
	}
	promptTags = "baseline, test"
	tags, warnings, err := resolveTagsFlag(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 || tags[0] != "baseline" || tags[1] != "test" {
		t.Fatalf("got %v", tags)
	}
	_ = warnings
}

func TestResolveTagsFlag_DedupWarning(t *testing.T) {
	cmd := newTagTestCmd()
	if err := cmd.Flags().Set("tags", "baseline,baseline"); err != nil {
		t.Fatal(err)
	}
	promptTags = "baseline,baseline"
	tags, warnings, err := resolveTagsFlag(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 1 || tags[0] != "baseline" {
		t.Fatalf("got %v", tags)
	}
	if len(warnings) == 0 {
		t.Fatal("expected duplicate warning")
	}
}

func TestResolveTagRefsFlag_DefaultAndEnv(t *testing.T) {
	t.Setenv(manifest.EnvKeyTagRefs, "")
	promptTagRefs = ""
	cmd := newTagTestCmd()
	mode, err := resolveTagRefsFlag(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != manifest.TagRefsInclude {
		t.Fatalf("got %q", mode)
	}

	t.Setenv(manifest.EnvKeyTagRefs, "strict")
	promptTagRefs = ""
	cmd = newTagTestCmd()
	mode, err = resolveTagRefsFlag(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != manifest.TagRefsStrict {
		t.Fatalf("got %q", mode)
	}
}

func TestPrintTagWarnings(t *testing.T) {
	cmd := &cobra.Command{Use: "x"}
	var buf bytes.Buffer
	cmd.SetErr(&buf)
	printTagWarnings(cmd, []string{"duplicate tag \"baseline\" removed"})
	if !strings.Contains(buf.String(), "WARNING:") {
		t.Fatalf("expected WARNING in %q", buf.String())
	}
}
