package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func readEmbeddedVersion(t *testing.T) string {
	t.Helper()
	path := filepath.Join(projectRoot(), "features", "tt", "internal", "version", "version.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read version.go: %v", err)
	}
	re := regexp.MustCompile(`(?m)^const Version = "([^"]+)"$`)
	m := re.FindSubmatch(data)
	if m == nil {
		t.Fatalf("const Version not found in %s", path)
	}
	return string(m[1])
}

func TestVersion_Subcommand(t *testing.T) {
	bin := ttBinary(t)
	want := readEmbeddedVersion(t)
	cmd := exec.Command(bin, "version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("tt version: %v\n%s", err, out)
	}
	got := strings.TrimSpace(string(out))
	expect := "tt " + want
	if got != expect {
		t.Fatalf("stdout %q, want %q", got, expect)
	}
}

func TestVersion_FlagLong(t *testing.T) {
	bin := ttBinary(t)
	want := readEmbeddedVersion(t)
	cmd := exec.Command(bin, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("tt --version: %v\n%s", err, out)
	}
	got := strings.TrimSpace(string(out))
	expect := "tt " + want
	if got != expect {
		t.Fatalf("stdout %q, want %q", got, expect)
	}
}

func TestVersion_FlagShortV(t *testing.T) {
	bin := ttBinary(t)
	want := readEmbeddedVersion(t)
	cmd := exec.Command(bin, "-V")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("tt -V: %v\n%s", err, out)
	}
	got := strings.TrimSpace(string(out))
	expect := "tt " + want
	if got != expect {
		t.Fatalf("stdout %q, want %q", got, expect)
	}
}

func TestVersion_Help(t *testing.T) {
	bin := ttBinary(t)
	cmd := exec.Command(bin, "help", "version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("tt help version: %v\n%s", err, out)
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
	if !strings.Contains(s, "# tt version") {
		t.Errorf("missing title # tt version:\n%s", s)
	}
}
