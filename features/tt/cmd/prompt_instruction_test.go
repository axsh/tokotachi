package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestPromptCmd_HelpMentionsInstruction(t *testing.T) {
	if !strings.Contains(promptCmd.Short, "instruction") {
		t.Errorf("promptCmd.Short missing instruction guide: %q", promptCmd.Short)
	}
	if !strings.Contains(promptCmd.Short, "{{kind:id}}") {
		t.Errorf("promptCmd.Short missing {{kind:id}}: %q", promptCmd.Short)
	}
	if !strings.Contains(promptCmd.Long, "tt prompt instruction") {
		t.Errorf("promptCmd.Long missing tt prompt instruction: %q", promptCmd.Long)
	}
	found := false
	for _, c := range promptCmd.Commands() {
		if c.Name() == "instruction" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("promptCmd missing instruction subcommand")
	}
}

func TestPromptInstructionCmd_PrintsManual(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := promptInstructionCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	if err := runPromptInstruction(cmd, nil); err != nil {
		t.Fatalf("runPromptInstruction: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"{{kind:id}}", "policy", "procedure", "capability", ".mdc", "instructions.md"} {
		if !strings.Contains(out, want) {
			t.Errorf("manual missing %q; got:\n%s", want, out)
		}
	}
}
