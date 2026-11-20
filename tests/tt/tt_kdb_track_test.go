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
    module_path: shared/kdb/%ID%
    corpus_path: shared/kdb/%ID%/corpus
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

// newKdbWorkspace creates a temp repo with a registry (backlog + designdb) under shared/kdb.
func newKdbWorkspace(t *testing.T, designdbStatus string) string {
	t.Helper()
	ws := t.TempDir()
	reg := "version: 1\nkdbs:\n" +
		kdbEntry("backlog", "echo backlog-unused") +
		kdbEntry("designdb", designdbStatus)
	mustWrite(t, filepath.Join(ws, "shared", "kdb", "registry.yaml"), reg)
	if err := os.MkdirAll(filepath.Join(ws, "shared", "kdb", "backlog", "corpus"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(ws, "shared", "kdb", "designdb", "corpus"), 0o755); err != nil {
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
		em := e.(map[string]any)
		ids = append(ids, em["id"].(string))
		if em["source"] != "registry" {
			t.Fatalf("yaml entries should be source=registry: %v", em)
		}
	}
	if strings.Join(ids, ",") != "backlog,designdb" {
		t.Fatalf("ids = %v", ids)
	}
}

func TestKdbTrack_List_MissingRegistry(t *testing.T) {
	ws := t.TempDir()
	out, stderr, code := runKdbTT(t, nil, "kdb-track", "list", "--workspace", ws)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, stderr)
	}
	m := firstLineJSON(t, out)
	if m["ok"] != true || m["registry_missing"] != true {
		t.Fatalf("want ok+registry_missing: %v", m)
	}
	if !strings.Contains(fmtString(m["message"]), "not found") {
		t.Fatalf("message=%v", m["message"])
	}
	entries, _ := m["entries"].([]any)
	if len(entries) != 1 {
		t.Fatalf("want builtin backlog only: %v", m["entries"])
	}
	e := entries[0].(map[string]any)
	if e["id"] != "backlog" || e["source"] != "builtin" {
		t.Fatalf("entry=%v", e)
	}
	if e["module_path"] != "shared/kdb/backlog" {
		t.Fatalf("module_path=%v", e["module_path"])
	}
}

func fmtString(v any) string {
	s, _ := v.(string)
	return s
}

func TestKdbTrack_List_SourceFields(t *testing.T) {
	ws := t.TempDir()
	reg := "version: 1\nkdbs:\n" + kdbEntry("designdb", "echo ok")
	mustWrite(t, filepath.Join(ws, "shared", "kdb", "registry.yaml"), reg)
	out, stderr, code := runKdbTT(t, nil, "kdb-track", "list", "--workspace", ws)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, stderr)
	}
	m := firstLineJSON(t, out)
	entries, _ := m["entries"].([]any)
	if len(entries) != 2 {
		t.Fatalf("want 2: %v", entries)
	}
	byID := map[string]map[string]any{}
	for _, e := range entries {
		em := e.(map[string]any)
		byID[em["id"].(string)] = em
	}
	if byID["backlog"]["source"] != "builtin" {
		t.Fatalf("backlog source=%v", byID["backlog"]["source"])
	}
	if byID["designdb"]["source"] != "registry" {
		t.Fatalf("designdb source=%v", byID["designdb"]["source"])
	}
}

func TestKdbTrack_Unregistered(t *testing.T) {
	ws := newKdbWorkspace(t, "echo ok")
	if err := os.MkdirAll(filepath.Join(ws, "shared", "kdb", "orphan", "corpus"), 0o755); err != nil {
		t.Fatal(err)
	}
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
		t.Fatalf("ids = %v (features/orphan must be ignored)", m["ids"])
	}
}

func TestKdbTrack_Skeleton_StdoutAndWrite(t *testing.T) {
	ws := t.TempDir()
	out, stderr, code := runKdbTT(t, nil, "kdb-track", "skeleton", "--workspace", ws)
	if code != 0 {
		t.Fatalf("stdout skeleton exit %d\n%s\n%s", code, out, stderr)
	}
	if !strings.Contains(out, "version: 1") || !strings.Contains(out, "--verbose") {
		t.Fatalf("skeleton missing expected content:\n%s", out)
	}

	out, stderr, code = runKdbTT(t, nil, "kdb-track", "skeleton", "--write", "--workspace", ws)
	if code != 0 {
		t.Fatalf("write exit %d\n%s\n%s", code, out, stderr)
	}
	m := firstLineJSON(t, out)
	path, _ := m["path"].(string)
	if path == "" {
		t.Fatalf("missing path: %v", m)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("skeleton not written: %v", err)
	}
	out, stderr, code = runKdbTT(t, nil, "kdb-track", "skeleton", "--write", "--workspace", ws)
	if code == 0 {
		t.Fatalf("second --write should fail; got ok\n%s\n%s", out, stderr)
	}
}

func TestKdbTrack_KdbRootOverride(t *testing.T) {
	ws := t.TempDir()
	out, stderr, code := runKdbTT(t, nil, "kdb-track", "list", "--workspace", ws, "--kdb-root", "other/kdb")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, stderr)
	}
	m := firstLineJSON(t, out)
	entries, _ := m["entries"].([]any)
	e := entries[0].(map[string]any)
	if e["module_path"] != "other/kdb/backlog" {
		t.Fatalf("module_path=%v", e["module_path"])
	}
	reg, _ := m["registry"].(string)
	if !strings.Contains(filepath.ToSlash(reg), "other/kdb/registry.yaml") {
		t.Fatalf("registry=%v", reg)
	}
}

func TestKdbTrack_StatusCmds_KeepsArgs(t *testing.T) {
	ws := t.TempDir()
	reg := "version: 1\nkdbs:\n" +
		kdbEntry("designdb", "echo hello --flag")
	mustWrite(t, filepath.Join(ws, "shared", "kdb", "registry.yaml"), reg)
	out, stderr, code := runKdbTT(t, nil, "kdb-track", "status-cmds", "--workspace", ws)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, stderr)
	}
	m := firstLineJSON(t, out)
	cmds, _ := m["commands"].([]any)
	found := false
	for _, c := range cmds {
		cm := c.(map[string]any)
		if cm["id"] == "designdb" {
			found = true
			if cm["status_cmd"] != "echo hello --flag" {
				t.Fatalf("status_cmd=%v", cm["status_cmd"])
			}
		}
	}
	if !found {
		t.Fatalf("designdb missing: %v", cmds)
	}
}

func TestKdb_DelegatesToRegistryCommand(t *testing.T) {
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
	out, stderr, code := runKdbTT(t, nil, "kdb", "nope", "status", "--workspace", ws)
	if code != 2 {
		t.Fatalf("unknown id: want exit 2, got %d\n%s\n%s", code, out, stderr)
	}
	if m := firstLineJSON(t, stderr); m["ok"] != false {
		t.Fatalf("want ok=false on stderr: %v", m)
	}
	out, stderr, code = runKdbTT(t, nil, "kdb", "designdb", "bogus", "--workspace", ws)
	if code != 2 {
		t.Fatalf("unknown op: want exit 2, got %d\n%s\n%s", code, out, stderr)
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

func TestKdbBacklog_WorksWithoutRegistry(t *testing.T) {
	ws := t.TempDir()
	out, stderr, code := runKdbTT(t, nil, "kdb", "backlog", "list", "--workspace", ws)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, stderr)
	}
	m := firstLineJSON(t, out)
	if m["ok"] != true {
		t.Fatalf("want ok: %v", m)
	}
}

func TestKdbTrack_StatusAllPartialFailureExits1(t *testing.T) {
	ws := newKdbWorkspace(t, `echo '{"ok":false,"error":"boom"}'`)
	out, stderr, code := runKdbTT(t, nil, "kdb-track", "status-all", "--workspace", ws)
	if code != 1 {
		t.Fatalf("want exit 1, got %d\n%s\n%s", code, out, stderr)
	}
	m := firstLineJSON(t, stderr)
	if m["ok"] != false {
		t.Fatalf("want ok=false on stderr: %v", m)
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
		t.Fatalf("git not available: %v", err)
	}
	ws := newKdbWorkspace(t, `echo '{"ok":true,"dirty":false}'`)
	gitInit := exec.Command("git", "init", "-q")
	gitInit.Dir = ws
	if out, err := gitInit.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	mustWrite(t, filepath.Join(ws, "shared", "kdb", "designdb", "corpus", "a.md"), "x")

	out, stderr, code := runKdbTT(t, nil, "kdb-track", "systematize", "--id", "designdb", "--workspace", ws)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, stderr)
	}
	m := firstLineJSON(t, out)
	if m["ok"] != true {
		t.Fatalf("ok != true: %v", m)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 || lines[1] != "ADD_CANDIDATE: shared/kdb/designdb/corpus" {
		t.Fatalf("unexpected output lines: %q", lines)
	}
}

func TestKdbTrack_HelpPages(t *testing.T) {
	for _, topic := range []string{
		"kdb-track", "kdb-track list", "kdb-track unregistered", "kdb-track status-cmds",
		"kdb-track status-all", "kdb-track systematize", "kdb-track skeleton", "kdb",
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
			if topic != "kdb-track skeleton" && !strings.Contains(out, "JSON") {
				t.Errorf("help must mention JSON output:\n%s", out)
			}
		})
	}
}

func TestKdbTrack_NoPythonOrSqliteOnPath(t *testing.T) {
	ws := newKdbWorkspace(t, "echo ok")
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
