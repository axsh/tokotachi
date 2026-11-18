package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/clihelp"
)

func execTT(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs(args)
	err = rootCmd.Execute()
	// Reset args for other tests
	rootCmd.SetArgs(nil)
	return outBuf.String(), errBuf.String(), err
}

func TestAgentHelp_RootListsCommands(t *testing.T) {
	stdout, _, err := execTT(t, "help")
	if err != nil {
		t.Fatalf("help: %v", err)
	}
	if !clihelp.ContainsRequiredHeadings(stdout) {
		t.Fatalf("root help missing headings:\n%s", stdout)
	}
	for _, want := range []string{"up", "prompt", "agent", "tt help"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("root help missing %q", want)
		}
	}
}

func TestAgentHelp_UpHasSevenHeadings(t *testing.T) {
	stdout, _, err := execTT(t, "help", "up")
	if err != nil {
		t.Fatalf("help up: %v", err)
	}
	if !clihelp.ContainsRequiredHeadings(stdout) {
		t.Fatalf("missing headings:\n%s", stdout)
	}
	if !strings.Contains(stdout, "# tt up") {
		t.Fatalf("title missing:\n%s", stdout)
	}
}

func TestAgentHelp_PromptUpdateHasSevenHeadings(t *testing.T) {
	stdout, _, err := execTT(t, "help", "prompt", "update")
	if err != nil {
		t.Fatalf("help prompt update: %v", err)
	}
	if !clihelp.ContainsRequiredHeadings(stdout) {
		t.Fatalf("missing headings:\n%s", stdout)
	}
}

func TestAgentHelp_FlagHelpSameAsTopic(t *testing.T) {
	viaHelp, _, err := execTT(t, "help", "up")
	if err != nil {
		t.Fatal(err)
	}
	viaFlag, _, err := execTT(t, "up", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !clihelp.ContainsRequiredHeadings(viaFlag) {
		t.Fatalf("--help missing headings:\n%s", viaFlag)
	}
	if !strings.Contains(viaHelp, "## Purpose") || !strings.Contains(viaFlag, "## Purpose") {
		t.Fatal("both forms should include Purpose")
	}
}

func TestAgentHelp_RequiredPathsRegistered(t *testing.T) {
	required := []string{
		"up", "down", "create", "delete", "open", "close", "editor", "status", "list", "shell", "exec",
		"prompt", "prompt compile", "prompt deploy", "prompt update", "prompt instruction",
		"agent", "agent record", "agent status", "agent intake", "agent knowledge",
		"merge", "pr", "doctor", "scaffold", "_update-code-status",
	}
	for _, p := range required {
		if _, ok := clihelp.Lookup(p); !ok {
			t.Errorf("missing clihelp registration for %q", p)
		}
	}
}

func TestAgentHelp_NoSideEffects(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	before, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := execTT(t, "help", "agent", "record"); err != nil {
		t.Fatalf("help: %v", err)
	}
	after, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("help created files: before=%d after=%d", len(before), len(after))
	}
	_ = filepath.Join // keep import used if empty
}
