package integration_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const kdbEntryYAML = `  - id: %ID%
    module_path: features/%ID%
    corpus_path: features/%ID%/corpus
    status_cmd: %STATUS%
    distill_cmd: echo distill
    dump_cmd: echo dump
    merge_cmd: echo merge
    pack_cmd: echo pack
`

func kdbEntry(id, statusCmd string) string {
	s := strings.ReplaceAll(kdbEntryYAML, "%ID%", id)
	return strings.ReplaceAll(s, "%STATUS%", statusCmd)
}

// newKdbWorkspace creates a temp repo with a registry (backlog + designdb).
func newKdbWorkspace(t *testing.T, designdbStatus string) string {
	t.Helper()
	ws := t.TempDir()
	reg := "version: 1\nkdbs:\n" +
		kdbEntry("backlog", "echo backlog-unused") +
		kdbEntry("designdb", designdbStatus)
	mustWrite(t, filepath.Join(ws, "features", "kdb", "registry.yaml"), reg)
	if err := os.MkdirAll(filepath.Join(ws, "features", "backlog", "corpus"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(ws, "features", "designdb", "corpus"), 0o755); err != nil {
		t.Fatal(err)
	}
	return ws
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// runKdbTT runs tt with optional PATH override; returns stdout, stderr, exit code.
func runKdbTT(t *testing.T, env []string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(ttBinary(t), args...)
	cmd.Dir = t.TempDir()
	if env != nil {
		cmd.Env = env
	}
	var o, e bytes.Buffer
	cmd.Stdout, cmd.Stderr = &o, &e
	err := cmd.Run()
	code := 0
	if err != nil {
		if cmd.ProcessState == nil {
			t.Fatalf("run %v: %v", args, err)
		}
		code = cmd.ProcessState.ExitCode()
	}
	return o.String(), e.String(), code
}

func firstLineJSON(t *testing.T, out string) map[string]any {
	t.Helper()
	line := strings.SplitN(out, "\n", 2)[0]
	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("first line is not JSON: %q (%v)", line, err)
	}
	return m
}

func TestKdbTrack_List(t *testing.T) {
	ws := newKdbWorkspace(t, `echo '{"ok":true,"dirty":false}'`)
	out, stderr, code := runKdbTT(t, nil, "kdb-track", "list", "--workspace", ws)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, stderr)
	}
	m := firstLineJSON(t, out)
	if m["ok"] != true {
		t.Fatalf("ok != true: %v", m)
	}
	entries, _ := m["entries"].([]any)
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %v", m["entries"])
	}
	ids := []string{}
	for _, e := range entries {
		ids = append(ids, e.(map[string]any)["id"].(string))
	}
	if strings.Join(ids, ",") != "backlog,designdb" {
		t.Fatalf("ids = %v", ids)
	}
}

func TestKdbTrack_Unregistered(t *testing.T) {
	ws := newKdbWorkspace(t, "echo ok")
	if err := os.MkdirAll(filepath.Join(ws, "features", "orphan", "corpus"), 0o755); err != nil {
		t.Fatal(err)
	}
	out, stderr, code := runKdbTT(t, nil, "kdb-track", "unregistered", "--workspace", ws)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, stderr)
	}
	m := firstLineJSON(t, out)
	ids, _ := m["ids"].([]any)
	if len(ids) != 1 || ids[0] != "orphan" {
		t.Fatalf("ids = %v", m["ids"])
	}
}

func TestKdb_DelegatesToRegistryCommand(t *testing.T) {
	// A fixed marker proves the registry status_cmd (not a builtin) ran.
	ws := newKdbWorkspace(t, `echo '{"ok":true,"dirty":false,"marker":"designdb-stub"}'`)
	out, stderr, code := runKdbTT(t, nil, "kdb", "designdb", "status", "--workspace", ws)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, stderr)
	}
	m := firstLineJSON(t, out)
	if m["marker"] != "designdb-stub" {
		t.Fatalf("stub marker missing: %q", out)
	}
}

func TestKdb_UnknownIDAndOp(t *testing.T) {
	ws := newKdbWorkspace(t, "echo ok")
	out, _, code := runKdbTT(t, nil, "kdb", "nope", "status", "--workspace", ws)
	if code != 2 {
		t.Fatalf("unknown id: want exit 2, got %d\n%s", code, out)
	}
	if m := firstLineJSON(t, out); m["ok"] != false {
		t.Fatalf("want ok=false: %v", m)
	}
	out, _, code = runKdbTT(t, nil, "kdb", "designdb", "bogus", "--workspace", ws)
	if code != 2 {
		t.Fatalf("unknown op: want exit 2, got %d\n%s", code, out)
	}
}

func TestKdb_BacklogIsWiredBuiltin(t *testing.T) {
	ws := newKdbWorkspace(t, "echo ok")
	out, stderr, code := runKdbTT(t, nil, "kdb", "backlog", "list", "--workspace", ws)
	if code != 0 {
		t.Fatalf("backlog list exit %d\n%s\n%s", code, out, stderr)
	}
	m := firstLineJSON(t, out)
	if m["ok"] != true || m["total"] != float64(0) {
		t.Fatalf("unexpected list output: %v", m)
	}
}

func TestKdbTrack_StatusAllPartialFailureExits1(t *testing.T) {
	// designdb reports ok=false -> exit 1; the builtin backlog must still be reported.
	ws := newKdbWorkspace(t, `echo '{"ok":false,"error":"boom"}'`)
	out, _, code := runKdbTT(t, nil, "kdb-track", "status-all", "--workspace", ws)
	if code != 1 {
		t.Fatalf("want exit 1, got %d\n%s", code, out)
	}
	m := firstLineJSON(t, out)
	if m["ok"] != false {
		t.Fatalf("want ok=false: %v", m)
	}
	statuses, _ := m["statuses"].([]any)
	if len(statuses) != 2 {
		t.Fatalf("all entries must be processed: %v", m["statuses"])
	}
	if statuses[0].(map[string]any)["ok"] != true {
		t.Fatalf("backlog should succeed: %v", statuses[0])
	}
	if statuses[1].(map[string]any)["ok"] != false {
		t.Fatalf("designdb should fail: %v", statuses[1])
	}
}

func TestKdbTrack_SystematizeFirstLineJSONThenCandidates(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	ws := newKdbWorkspace(t, `echo '{"ok":true,"dirty":false}'`)
	// Only designdb is systematized; its corpus is git-dirty (untracked file).
	gitInit := exec.Command("git", "init", "-q")
	gitInit.Dir = ws
	if out, err := gitInit.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	mustWrite(t, filepath.Join(ws, "features", "designdb", "corpus", "a.md"), "x")

	out, stderr, code := runKdbTT(t, nil, "kdb-track", "systematize", "--id", "designdb", "--workspace", ws)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, stderr)
	}
	m := firstLineJSON(t, out)
	if m["ok"] != true {
		t.Fatalf("ok != true: %v", m)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 || lines[1] != "ADD_CANDIDATE: features/designdb/corpus" {
		t.Fatalf("unexpected output lines: %q", lines)
	}
}

func TestKdbTrack_HelpPages(t *testing.T) {
	for _, topic := range []string{
		"kdb-track", "kdb-track list", "kdb-track unregistered", "kdb-track status-cmds",
		"kdb-track status-all", "kdb-track systematize", "kdb",
	} {
		t.Run(topic, func(t *testing.T) {
			args := append([]string{"help"}, strings.Fields(topic)...)
			out, stderr, code := runKdbTT(t, nil, args...)
			if code != 0 {
				t.Fatalf("exit %d\n%s\n%s", code, out, stderr)
			}
			for _, h := range []string{
				"## Purpose", "## When to use", "## When not to use", "## Preconditions",
				"## Input and output", "## Exit codes", "## Read next",
			} {
				if !strings.Contains(out, h) {
					t.Errorf("missing %s in:\n%s", h, out)
				}
			}
			if !strings.Contains(out, "JSON") {
				t.Errorf("help must mention JSON output:\n%s", out)
			}
		})
	}
}

func TestKdbTrack_NoPythonOrSqliteOnPath(t *testing.T) {
	ws := newKdbWorkspace(t, "echo ok")
	// Minimal environment: empty PATH removes python/python3/sqlite3 entirely.
	env := []string{"PATH=", "SystemRoot=" + os.Getenv("SystemRoot"), "TEMP=" + os.TempDir(), "TMP=" + os.TempDir()}

	out, stderr, code := runKdbTT(t, env, "kdb-track", "list", "--workspace", ws)
	if code != 0 {
		t.Fatalf("list exit %d\n%s\n%s", code, out, stderr)
	}
	out, stderr, code = runKdbTT(t, env, "help", "kdb-track", "systematize")
	if code != 0 {
		t.Fatalf("help exit %d\n%s\n%s", code, out, stderr)
	}
}
