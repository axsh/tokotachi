package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func setEmbeddedVersionScript(t *testing.T) string {
	t.Helper()
	path := filepath.Join(projectRoot(), "scripts", "dist", "tool", "internal", "set_embedded_version.sh")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("set_embedded_version.sh not found: %v", err)
	}
	return path
}

func runBash(t *testing.T, args ...string) (string, error) {
	t.Helper()
	bash := "bash"
	if runtime.GOOS == "windows" {
		if p, err := exec.LookPath("bash"); err == nil {
			bash = p
		}
	}
	cmd := exec.Command(bash, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestSetEmbeddedVersion_RewritesOnlyConst(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "version.go")
	original := `package version

// version: 2 must not change
const Version = "v0.9.1"
`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runBash(t, setEmbeddedVersionScript(t), path, "v0.9.2")
	if err != nil {
		t.Fatalf("set_embedded_version: %v\n%s", err, out)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, `const Version = "v0.9.2"`) {
		t.Fatalf("Version not updated:\n%s", got)
	}
	if !strings.Contains(got, "// version: 2 must not change") {
		t.Fatalf("unrelated version text was changed:\n%s", got)
	}
	if strings.Contains(got, `const Version = "v0.9.1"`) {
		t.Fatalf("old Version still present:\n%s", got)
	}
}

func TestSetEmbeddedVersion_RejectsBadFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "version.go")
	if err := os.WriteFile(path, []byte("package version\n\nconst Version = \"v0.9.1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runBash(t, setEmbeddedVersionScript(t), path, "not-a-version")
	if err == nil {
		t.Fatalf("expected failure for bad format, got success:\n%s", out)
	}
}

func TestSetEmbeddedVersion_RejectsMissingConst(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "version.go")
	if err := os.WriteFile(path, []byte("package version\n\n// no const\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runBash(t, setEmbeddedVersionScript(t), path, "v0.9.2")
	if err == nil {
		t.Fatalf("expected failure for missing const, got success:\n%s", out)
	}
}
