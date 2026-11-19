package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const blPhase = "000-foundation"
const blBranch = "main"

// backlogRegistry returns a registry containing only the builtin backlog entry
// rooted at module (for example "features/backlog" or "scripts/kdb/backlog").
func backlogRegistry(module string) string {
	return "version: 1\nkdbs:\n" +
		"  - id: backlog\n" +
		"    module_path: " + module + "\n" +
		"    corpus_path: " + module + "/corpus\n" +
		"    status_cmd: echo unused\n" +
		"    distill_cmd: echo unused\n" +
		"    dump_cmd: echo unused\n" +
		"    merge_cmd: echo unused\n" +
		"    pack_cmd: echo unused\n"
}

func newBacklogWorkspace(t *testing.T, module string) string {
	t.Helper()
	ws := t.TempDir()
	mustWrite(t, filepath.Join(ws, "features", "kdb", "registry.yaml"), backlogRegistry(module))
	return ws
}

// blEnv is the process environment with BL_* overrides removed and BL_TODAY pinned.
func blEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "BL_") || strings.HasPrefix(kv, "KDB_REGISTRY=") || strings.HasPrefix(kv, "TT_WORKSPACE=") {
			continue
		}
		env = append(env, kv)
	}
	return append(env, "BL_TODAY=2026-11-18")
}

// blRun runs `tt kdb backlog <args...> --workspace ws` and returns the first JSON line.
func blRun(t *testing.T, ws string, args ...string) (map[string]any, string, int) {
	t.Helper()
	return blRunEnv(t, blEnv(), ws, args...)
}

func blRunEnv(t *testing.T, env []string, ws string, args ...string) (map[string]any, string, int) {
	t.Helper()
	full := append([]string{"kdb", "backlog"}, args...)
	full = append(full, "--workspace", ws)
	out, stderr, code := runKdbTT(t, env, full...)
	return firstLineJSON(t, out), out + stderr, code
}

func blMustOK(t *testing.T, ws string, args ...string) map[string]any {
	t.Helper()
	m, raw, code := blRun(t, ws, args...)
	if code != 0 || m["ok"] != true {
		t.Fatalf("backlog %v: exit %d\n%s", args, code, raw)
	}
	return m
}

func blAdd(t *testing.T, ws, title string, extra ...string) map[string]any {
	t.Helper()
	args := append([]string{"add", "--phase", blPhase, "--branch", blBranch, "--title", title, "--kind", "確認事項"}, extra...)
	return blMustOK(t, ws, args...)["item"].(map[string]any)
}

func blKeyArgs(it map[string]any) []string {
	return []string{"--phase", it["phase"].(string), "--branch", it["branch"].(string), "--id", it["local_id"].(string)}
}

func blData(m map[string]any) map[string]any { return m["data"].(map[string]any) }

func TestKdbBacklogHelpWithoutDB(t *testing.T) {
	ws := newBacklogWorkspace(t, "features/backlog")
	ops := []string{"add", "list", "get", "update", "resolve", "status", "distill", "dump", "merge",
		"pack", "unpack", "restore", "import-md", "migrate-uuids"}
	for _, op := range ops {
		for _, via := range [][]string{
			{"kdb", "backlog", op, "help", "--workspace", ws},
			{"kdb", "backlog", op, "--help", "--workspace", ws},
			{"help", "kdb", "backlog", op},
		} {
			out, stderr, code := runKdbTT(t, blEnv(), via...)
			if code != 0 {
				t.Fatalf("%v: exit %d\n%s\n%s", via, code, out, stderr)
			}
			for _, h := range []string{"## Purpose", "## When to use", "## When not to use", "## Preconditions",
				"## Input and output", "## Exit codes", "## Read next"} {
				if !strings.Contains(out, h) {
					t.Errorf("%v: missing %s", via, h)
				}
			}
			if !strings.Contains(out, "# tt kdb backlog "+op) {
				t.Errorf("%v: wrong page title:\n%s", via, out)
			}
			if strings.Contains(strings.ToLower(out), "python3") {
				t.Errorf("%v: help must not require Python:\n%s", via, out)
			}
		}
	}
	out, _, code := runKdbTT(t, blEnv(), "kdb", "backlog", "help", "--workspace", ws)
	if code != 0 || !strings.Contains(out, "# tt kdb backlog") {
		t.Fatalf("parent help: exit %d\n%s", code, out)
	}
	// Help must work even without a registry and must not create a database.
	empty := t.TempDir()
	if _, _, code := runKdbTT(t, blEnv(), "kdb", "backlog", "add", "help", "--workspace", empty); code != 0 {
		t.Fatalf("help without registry: exit %d", code)
	}
	for _, dir := range []string{ws, empty} {
		matches, _ := filepath.Glob(filepath.Join(dir, "features", "backlog", "data", "*"))
		if len(matches) != 0 {
			t.Fatalf("help created files: %v", matches)
		}
	}
}

func TestKdbBacklogFailureJSONHasOpHelpHint(t *testing.T) {
	ws := newBacklogWorkspace(t, "features/backlog")
	m, raw, code := blRun(t, ws, "add", "--phase", blPhase, "--bogus", "x")
	if code != 2 || m["ok"] != false {
		t.Fatalf("want exit 2 ok=false, got %d\n%s", code, raw)
	}
	if m["help"] != "tt kdb backlog add help" {
		t.Fatalf("help hint = %v", m["help"])
	}
	m, _, code = blRun(t, ws, "add", "--phase", blPhase)
	if code != 2 || !strings.Contains(m["error"].(string), "missing required flag --branch") {
		t.Fatalf("missing flag: %d %v", code, m)
	}
	m, _, code = blRun(t, ws, "nope")
	if code != 2 || m["help"] != "tt kdb backlog help" {
		t.Fatalf("unknown op: %d %v", code, m)
	}
}

func TestKdbBacklogCRUD(t *testing.T) {
	ws := newBacklogWorkspace(t, "features/backlog")

	a := blAdd(t, ws, "first question", "--problem", "RLS?", "--slug", "rls")
	if a["local_id"] != "000" || a["state"] != "未確認" || a["state_date"] != "2026-11-18" ||
		a["created_at"] != "2026-11-18" || a["updated_at"] != "2026-11-18" || a["problem"] != "RLS?" {
		t.Fatalf("add defaults wrong: %v", a)
	}
	uuid, _ := a["uuid"].(string)
	if len(uuid) != 36 {
		t.Fatalf("uuid = %q", uuid)
	}
	b := blAdd(t, ws, "second", "--kind", "先送り")
	if b["local_id"] != "001" {
		t.Fatalf("auto id: %v", b)
	}

	// Duplicate explicit id fails with exit 1.
	m, _, code := blRun(t, ws, "add", "--phase", blPhase, "--branch", blBranch, "--title", "dup", "--kind", "先送り", "--id", "000")
	if code != 1 || m["ok"] != false {
		t.Fatalf("duplicate id: %d %v", code, m)
	}

	// list
	l := blMustOK(t, ws, "list")
	if l["total"] != float64(2) || len(l["items"].([]any)) != 2 || l["limit"] != nil || l["offset"] != float64(0) {
		t.Fatalf("list: %v", l)
	}

	// get by key
	g := blMustOK(t, ws, append([]string{"get"}, blKeyArgs(a)...)...)["item"].(map[string]any)
	if g["uuid"] != uuid || g["title"] != "first question" {
		t.Fatalf("get: %v", g)
	}

	// update: replace + append-history
	u := blMustOK(t, ws, append(append([]string{"update"}, blKeyArgs(a)...),
		"--state", "確認中", "--history", "h1")...)["item"].(map[string]any)
	if u["state"] != "確認中" || u["history"] != "h1" || u["uuid"] != uuid {
		t.Fatalf("update: %v", u)
	}
	u = blMustOK(t, ws, append(append([]string{"update"}, blKeyArgs(a)...), "--append-history", "h2")...)["item"].(map[string]any)
	if u["history"] != "h1\nh2" {
		t.Fatalf("append-history: %v", u["history"])
	}
	// update must not close, and --history/--append-history are exclusive.
	for _, bad := range [][]string{{"--state", "決着"}, {"--history", "x", "--append-history", "y"}, {}} {
		_, _, code := blRun(t, ws, append(append([]string{"update"}, blKeyArgs(a)...), bad...)...)
		if code != 2 {
			t.Fatalf("update %v: want exit 2, got %d", bad, code)
		}
	}

	// resolve
	r := blMustOK(t, ws, append(append([]string{"resolve"}, blKeyArgs(a)...), "--state", "決着", "--decision", "no RLS")...)["item"].(map[string]any)
	if r["state"] != "決着" || r["decision"] != "no RLS" || r["state_date"] != "2026-11-18" {
		t.Fatalf("resolve: %v", r)
	}
	if _, _, code := blRun(t, ws, append(append([]string{"resolve"}, blKeyArgs(b)...), "--state", "確認中")...); code != 2 {
		t.Fatalf("resolve with open state must exit 2, got %d", code)
	}
	if _, _, code := blRun(t, ws, "get", "--phase", blPhase, "--branch", blBranch, "--id", "404"); code != 1 {
		t.Fatalf("missing item must exit 1, got %d", code)
	}
}

func TestKdbBacklogListFilters(t *testing.T) {
	ws := newBacklogWorkspace(t, "features/backlog")
	a := blAdd(t, ws, "Alpha rule", "--problem", "about RLS")
	blAdd(t, ws, "Beta", "--kind", "先送り")
	c := blAdd(t, ws, "Gamma")
	blMustOK(t, ws, append(append([]string{"resolve"}, blKeyArgs(c)...), "--state", "取り下げ")...)

	total := func(args ...string) float64 {
		return blMustOK(t, ws, append([]string{"list"}, args...)...)["total"].(float64)
	}
	if n := total(); n != 2 {
		t.Fatalf("closed must be excluded by default, total=%v", n)
	}
	if n := total("--include-closed", "1"); n != 3 {
		t.Fatalf("include-closed total=%v", n)
	}
	if n := total("--state", "取り下げ"); n != 1 {
		t.Fatalf("explicit --state total=%v", n)
	}
	if n := total("--match", "rls"); n != 1 {
		t.Fatalf("match total=%v", n)
	}
	if n := total("--kind", "先送り"); n != 1 {
		t.Fatalf("kind total=%v", n)
	}
	page := blMustOK(t, ws, "list", "--include-closed", "1", "--sort", "title", "--order", "desc", "--limit", "1", "--offset", "1")
	items := page["items"].([]any)
	if page["total"] != float64(3) || page["limit"] != float64(1) || page["offset"] != float64(1) || len(items) != 1 ||
		items[0].(map[string]any)["title"] != "Beta" {
		t.Fatalf("paging: %v", page)
	}
	m := blMustOK(t, ws, "list", "--match", "alpha")
	if m["items"].([]any)[0].(map[string]any)["uuid"] != a["uuid"] {
		t.Fatalf("match picked wrong item: %v", m)
	}
	for _, bad := range [][]string{{"--match", "("}, {"--limit", "0"}, {"--sort", "nope"}, {"--order", "up"}, {"--state", "bad"}} {
		if _, _, code := blRun(t, ws, append([]string{"list"}, bad...)...); code != 2 {
			t.Fatalf("list %v: want exit 2, got %d", bad, code)
		}
	}
}

func TestKdbBacklogGetByUUID(t *testing.T) {
	ws := newBacklogWorkspace(t, "features/backlog")
	a := blAdd(t, ws, "first")
	b := blAdd(t, ws, "second")

	g := blMustOK(t, ws, "get", "--uuid", a["uuid"].(string))["item"].(map[string]any)
	if g["local_id"] != a["local_id"] {
		t.Fatalf("get by uuid: %v", g)
	}
	g = blMustOK(t, ws, "get", "--uuid", strings.ToUpper(a["uuid"].(string)))["item"].(map[string]any)
	if g["uuid"] != a["uuid"] {
		t.Fatalf("upper-case uuid: %v", g)
	}
	args := append([]string{"get", "--uuid", a["uuid"].(string)}, blKeyArgs(b)...)
	m, _, code := blRun(t, ws, args...)
	if code != 1 || !strings.Contains(m["error"].(string), "does not match") {
		t.Fatalf("mismatch: %d %v", code, m)
	}
	if _, _, code := blRun(t, ws, "get", "--uuid", "not-a-uuid"); code != 2 {
		t.Fatalf("invalid uuid must exit 2, got %d", code)
	}
	if _, _, code := blRun(t, ws, "get", "--uuid", "550e8400-e29b-41d4-a716-446655440000"); code != 1 {
		t.Fatalf("unknown uuid must exit 1, got %d", code)
	}
}

func TestKdbBacklogRoundTrip(t *testing.T) {
	ws := newBacklogWorkspace(t, "features/backlog")
	a := blAdd(t, ws, "first", "--problem", "日本語 <tag> & \"quote\"\nline2")
	b := blAdd(t, ws, "second")

	blMustOK(t, ws, "distill")
	d := blData(blMustOK(t, ws, "dump"))
	if d["item_count"] != float64(2) {
		t.Fatalf("dump: %v", d)
	}
	mg := blData(blMustOK(t, ws, "merge"))
	if mg["restored"] != true || mg["merge"].(map[string]any)["conflict_count"] != float64(0) {
		t.Fatalf("merge: %v", mg)
	}
	p1 := blData(blMustOK(t, ws, "pack"))
	hash := p1["content_hash"].(string)
	if len(hash) != 64 || p1["item_count"] != float64(2) {
		t.Fatalf("pack: %v", p1)
	}
	manifestPath := filepath.Join(ws, "features", "backlog", "corpus", "manifest.json")
	manifest1, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}

	// Re-running the whole flow keeps the content hash and the manifest bytes.
	blMustOK(t, ws, "distill")
	blMustOK(t, ws, "dump")
	blMustOK(t, ws, "merge")
	p2 := blData(blMustOK(t, ws, "pack"))
	if p2["content_hash"] != hash {
		t.Fatalf("content_hash changed: %v -> %v", hash, p2["content_hash"])
	}
	manifest2, _ := os.ReadFile(manifestPath)
	if string(manifest1) != string(manifest2) {
		t.Fatalf("manifest changed although content did not:\n%s\n%s", manifest1, manifest2)
	}
	var man map[string]any
	if err := json.Unmarshal(manifest1, &man); err != nil {
		t.Fatal(err)
	}
	if man["kdb"] != "backlog" || man["content_hash"] != hash || man["chunk_size"] != float64(100) {
		t.Fatalf("manifest: %v", man)
	}

	st := blData(blMustOK(t, ws, "status"))
	if st["dirty"] != false || st["db_hash"] != hash || st["corpus_hash"] != hash {
		t.Fatalf("status after pack: %v", st)
	}

	// Drop the cache and the expanded data; rebuild from the corpus only.
	if err := os.RemoveAll(filepath.Join(ws, "features", "backlog", "data")); err != nil {
		t.Fatal(err)
	}
	st = blData(blMustOK(t, ws, "status"))
	if st["dirty"] != true || st["item_count"] != float64(0) {
		t.Fatalf("empty cache must be dirty: %v", st)
	}
	u := blData(blMustOK(t, ws, "unpack"))
	if u["content_hash"] != hash || u["item_count"] != float64(2) {
		t.Fatalf("unpack: %v", u)
	}
	r := blData(blMustOK(t, ws, "restore"))
	if r["restored"] != float64(2) {
		t.Fatalf("restore: %v", r)
	}
	for _, it := range []map[string]any{a, b} {
		g := blMustOK(t, ws, "get", "--uuid", it["uuid"].(string))["item"].(map[string]any)
		if g["title"] != it["title"] || g["problem"] != it["problem"] {
			t.Fatalf("item lost after restore: %v vs %v", g, it)
		}
	}
	st = blData(blMustOK(t, ws, "status"))
	if st["dirty"] != false || st["db_hash"] != hash {
		t.Fatalf("status after restore: %v", st)
	}

	// A local change makes it dirty and changes the hash.
	blMustOK(t, ws, append(append([]string{"update"}, blKeyArgs(a)...), "--title", "changed")...)
	st = blData(blMustOK(t, ws, "status"))
	if st["dirty"] != true || st["db_hash"] == hash {
		t.Fatalf("status after update: %v", st)
	}
}

func TestKdbBacklogMergeConflict(t *testing.T) {
	ws := newBacklogWorkspace(t, "features/backlog")
	a := blAdd(t, ws, "local title")
	blMustOK(t, ws, "dump")

	// The "corpus" side holds the same key with different content.
	other := map[string]any{}
	for k, v := range a {
		other[k] = v
	}
	other["title"] = "corpus title"
	raw, _ := json.Marshal([]any{other})
	dataJSON := filepath.Join(ws, "features", "backlog", "data", "items.json")
	mustWrite(t, dataJSON, string(raw))

	m, out, code := blRun(t, ws, "merge")
	if code != 0 || m["ok"] != true {
		t.Fatalf("merge conflict must exit 0 with ok=true: %d\n%s", code, out)
	}
	d := blData(m)
	merge := d["merge"].(map[string]any)
	if d["restored"] != false || merge["conflict_count"].(float64) < 1 {
		t.Fatalf("want restored=false and conflict_count>0: %v", d)
	}
	// The cache was not touched and pack refuses the conflicting array.
	g := blMustOK(t, ws, "get", "--uuid", a["uuid"].(string))["item"].(map[string]any)
	if g["title"] != "local title" {
		t.Fatalf("cache must be unchanged: %v", g)
	}
	if _, _, code := blRun(t, ws, "pack"); code != 1 {
		t.Fatalf("pack with conflicts must exit 1, got %d", code)
	}
	if _, _, code := blRun(t, ws, "distill"); code != 1 {
		t.Fatalf("distill with conflicts must exit 1, got %d", code)
	}

	// Recovery: reset data to the corpus version and prefer one side.
	mustWrite(t, dataJSON, string(raw))
	m, out, code = blRun(t, ws, "merge", "--prefer", "data")
	if code != 0 || blData(m)["restored"] != true {
		t.Fatalf("--prefer data: %d\n%s", code, out)
	}
	g = blMustOK(t, ws, "get", "--uuid", a["uuid"].(string))["item"].(map[string]any)
	if g["title"] != "corpus title" {
		t.Fatalf("prefer data must keep corpus: %v", g)
	}
	if _, _, code := blRun(t, ws, "merge", "--prefer", "bogus"); code != 2 {
		t.Fatalf("invalid --prefer must exit 2, got %d", code)
	}
}

func TestKdbBacklogMergeWithoutDumpFails(t *testing.T) {
	ws := newBacklogWorkspace(t, "features/backlog")
	m, _, code := blRun(t, ws, "merge")
	if code != 1 || !strings.Contains(m["error"].(string), "dump not found") {
		t.Fatalf("merge without dump: %d %v", code, m)
	}
}

func TestKdbBacklogNoPythonNoSqliteCLI(t *testing.T) {
	ws := newBacklogWorkspace(t, "features/backlog")
	// An empty PATH removes python, python3 and sqlite3 (and bash) entirely.
	env := []string{"PATH=", "BL_TODAY=2026-11-18", "SystemRoot=" + os.Getenv("SystemRoot"),
		"TEMP=" + os.TempDir(), "TMP=" + os.TempDir(), "HOME=" + os.TempDir(), "USERPROFILE=" + os.TempDir()}
	for _, name := range []string{"python", "python3", "sqlite3"} {
		if _, err := exec.LookPath(name); err == nil {
			// Present on the developer PATH is fine: the child never sees it.
			t.Logf("%s exists on the test runner PATH but is hidden from tt", name)
		}
	}
	run := func(args ...string) map[string]any {
		t.Helper()
		m, raw, code := blRunEnv(t, env, ws, args...)
		if code != 0 || m["ok"] != true {
			t.Fatalf("%v with empty PATH: exit %d\n%s", args, code, raw)
		}
		return m
	}
	a := run("add", "--phase", blPhase, "--branch", blBranch, "--title", "t", "--kind", "確認事項")["item"].(map[string]any)
	run("list", "--match", "t")
	run("get", "--uuid", a["uuid"].(string))
	run("distill")
	run("dump")
	run("merge")
	run("pack")
	run("unpack")
	run("restore")
	run("status")
}

func TestKdbTrackSystematizeBacklog(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	ws := newBacklogWorkspace(t, "features/backlog")
	if out, err := exec.Command("git", "-C", ws, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	a := blAdd(t, ws, "dirty item")

	// status-all sees the builtin backlog as dirty.
	out, stderr, code := runKdbTT(t, blEnv(), "kdb-track", "status-all", "--workspace", ws)
	if code != 0 {
		t.Fatalf("status-all: exit %d\n%s\n%s", code, out, stderr)
	}
	statuses := firstLineJSON(t, out)["statuses"].([]any)
	if len(statuses) != 1 || statuses[0].(map[string]any)["dirty"] != true {
		t.Fatalf("backlog must be dirty: %v", statuses)
	}

	out, stderr, code = runKdbTT(t, blEnv(), "kdb-track", "systematize", "--id", "backlog", "--workspace", ws)
	if code != 0 {
		t.Fatalf("systematize: exit %d\n%s\n%s", code, out, stderr)
	}
	m := firstLineJSON(t, out)
	if m["ok"] != true {
		t.Fatalf("systematize ok != true: %v", m)
	}
	cands, _ := m["add_candidates"].([]any)
	if len(cands) != 1 || cands[0] != "features/backlog/corpus" {
		t.Fatalf("add_candidates = %v", m["add_candidates"])
	}
	entries := m["entries"].([]any)
	if entries[0].(map[string]any)["action"] != "systematized" {
		t.Fatalf("entry: %v", entries[0])
	}
	if !strings.Contains(out, "ADD_CANDIDATE: features/backlog/corpus\n") {
		t.Fatalf("missing ADD_CANDIDATE line:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(ws, "features", "backlog", "corpus", "manifest.json")); err != nil {
		t.Fatalf("corpus not packed: %v", err)
	}

	// Status is clean afterwards and the item survived.
	st := blData(blMustOK(t, ws, "status"))
	if st["dirty"] != false || st["item_count"] != float64(1) {
		t.Fatalf("status after systematize: %v", st)
	}
	blMustOK(t, ws, "get", "--uuid", a["uuid"].(string))
}

func TestKdbTrackSystematizeBacklogFailsOnInvalidItem(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	ws := newBacklogWorkspace(t, "features/backlog")
	if out, err := exec.Command("git", "-C", ws, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	// An item with an empty title cannot be added; craft it through restore.
	bad := `[{"phase":"p","branch":"b","local_id":"001","title":"","state":"未確認","kind":"確認事項","uuid":"550e8400-e29b-41d4-a716-446655440000"}]`
	mustWrite(t, filepath.Join(ws, "features", "backlog", "data", "items.json"), bad)
	blMustOK(t, ws, "restore")

	out, _, code := runKdbTT(t, blEnv(), "kdb-track", "systematize", "--id", "backlog", "--workspace", ws)
	if code != 1 {
		t.Fatalf("want exit 1, got %d\n%s", code, out)
	}
	m := firstLineJSON(t, out)
	entry := m["entries"].([]any)[0].(map[string]any)
	if m["ok"] != false || entry["stage"] != "distill" || !strings.Contains(entry["error"].(string), "empty title") {
		t.Fatalf("distill failure must be reported: %v", m)
	}
}

func TestKdbPathsScriptsLayout(t *testing.T) {
	ws := newBacklogWorkspace(t, "scripts/kdb/backlog")
	blAdd(t, ws, "layout item")
	blMustOK(t, ws, "dump")
	blMustOK(t, ws, "merge")
	blMustOK(t, ws, "pack")

	for _, rel := range []string{
		"scripts/kdb/backlog/data/backlog.sqlite",
		"scripts/kdb/backlog/data/items.json",
		"scripts/kdb/backlog/corpus/manifest.json",
		"scripts/kdb/backlog/corpus/items/chunk-0000.json",
		"tmp/kdb/backlog/items.json",
	} {
		if _, err := os.Stat(filepath.Join(ws, filepath.FromSlash(rel))); err != nil {
			t.Errorf("expected %s: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(ws, "features", "backlog")); err == nil {
		t.Errorf("features/backlog must not be created for the scripts layout")
	}
}

func TestKdbBacklogPathEnvOverrides(t *testing.T) {
	ws := newBacklogWorkspace(t, "features/backlog")
	custom := filepath.Join(t.TempDir(), "custom")
	env := append(blEnv(),
		"BL_DB_PATH="+filepath.Join(custom, "db", "x.sqlite"),
		"BL_CORPUS_DIR="+filepath.Join(custom, "corpus"),
		"BL_DATA_DIR="+filepath.Join(custom, "data"),
		"BL_TMP_DIR="+filepath.Join(custom, "tmp"))
	run := func(args ...string) {
		t.Helper()
		m, raw, code := blRunEnv(t, env, ws, args...)
		if code != 0 || m["ok"] != true {
			t.Fatalf("%v: exit %d\n%s", args, code, raw)
		}
	}
	run("add", "--phase", blPhase, "--branch", blBranch, "--title", "t", "--kind", "確認事項")
	run("dump")
	run("merge")
	run("pack")
	for _, rel := range []string{"db/x.sqlite", "corpus/manifest.json", "data/items.json", "tmp/items.json"} {
		if _, err := os.Stat(filepath.Join(custom, filepath.FromSlash(rel))); err != nil {
			t.Errorf("expected %s: %v", rel, err)
		}
	}
}

func TestKdbBacklogImportMDAndMigrateUUIDs(t *testing.T) {
	ws := newBacklogWorkspace(t, "features/backlog")
	dir := filepath.Join(ws, "prompts", "phases", blPhase, "branches", blBranch, "backlogs")
	mustWrite(t, filepath.Join(dir, "007-Rls.md"),
		"# 007 RLS policy\n\n- 状態: 確認中（2026-05-02）\n- 種類: 確認事項\n\n## 何が問題か\n\nUndecided.\n\n## 経緯\n\n- 2026-05-01 created\n")
	m := blMustOK(t, ws, "import-md", dir)
	if m["imported"] != float64(1) || m["files"] != float64(1) {
		t.Fatalf("import-md: %v", m)
	}
	g := blMustOK(t, ws, "get", "--phase", blPhase, "--branch", blBranch, "--id", "007")["item"].(map[string]any)
	if g["title"] != "RLS policy" || g["state"] != "確認中" || g["state_date"] != "2026-05-02" || g["created_at"] != "2026-05-01" {
		t.Fatalf("imported item: %v", g)
	}
	if _, _, code := blRun(t, ws, "import-md", filepath.Join(ws, "missing")); code != 2 {
		t.Fatalf("missing path must exit 2, got %d", code)
	}

	legacy := filepath.Join(ws, "legacy.json")
	mustWrite(t, legacy, `[{"phase":"p","branch":"b","local_id":"001","title":"t","logical_uuid":"x"}]`)
	mu := blMustOK(t, ws, "migrate-uuids", "--in", legacy)
	if mu["assigned"] != float64(1) || mu["logical_uuid_removed"] != float64(1) || mu["count"] != float64(1) {
		t.Fatalf("migrate-uuids: %v", mu)
	}
	raw, _ := os.ReadFile(legacy)
	if strings.Contains(string(raw), "logical_uuid") || !strings.Contains(string(raw), `"uuid": "`) {
		t.Fatalf("legacy file not migrated:\n%s", raw)
	}
}
