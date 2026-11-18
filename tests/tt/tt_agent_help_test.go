package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentHelp_BinaryContracts(t *testing.T) {
	bin := ttBinary(t)
	cases := [][]string{
		{"help"},
		{"help", "up"},
		{"help", "prompt", "update"},
		{"help", "agent", "record"},
		{"up", "--help"},
		{"prompt", "update", "--help"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			cmd := exec.Command(bin, args...)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%v: %v\n%s", args, err, out)
			}
			s := string(out)
			for _, h := range []string{
				"## Purpose",
				"## When to use",
				"## When not to use",
				"## Preconditions",
				"## Input and output",
				"## Exit codes",
				"## Read next",
			} {
				if !strings.Contains(s, h) {
					t.Errorf("missing %s in output:\n%s", h, s)
				}
			}
		})
	}
}

func TestAgentHelp_DoesNotCreateSqlite(t *testing.T) {
	bin := ttBinary(t)
	dir := t.TempDir()
	cmd := exec.Command(bin, "help", "agent", "record")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("help: %v\n%s", err, out)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sqlite") || strings.Contains(e.Name(), "sqlite") {
			t.Fatalf("unexpected sqlite artifact %s", filepath.Join(dir, e.Name()))
		}
	}
	if len(entries) != 0 {
		t.Fatalf("help wrote files: %v", entries)
	}
}
