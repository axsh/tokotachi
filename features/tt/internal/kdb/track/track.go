// Package track implements cross-KDB operations driven by the registry:
// list, unregistered corpus detection, status collection, and systematize.
package track

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/axsh/tokotachi/features/tt/internal/kdb/builtin"
	"github.com/axsh/tokotachi/features/tt/internal/kdb/delegate"
	"github.com/axsh/tokotachi/features/tt/internal/kdb/paths"
	"github.com/axsh/tokotachi/features/tt/internal/kdb/registry"
)

// Source labels for list responses.
const (
	SourceBuiltin  = "builtin"
	SourceRegistry = "registry"
)

// ListedEntry is a registry entry plus where it came from.
type ListedEntry struct {
	registry.Entry
	Source string `json:"source"`
}

// ListResult is the output of List.
type ListResult struct {
	OK              bool          `json:"ok"`
	Registry        string        `json:"registry,omitempty"`
	RegistryMissing bool          `json:"registry_missing,omitempty"`
	Message         string        `json:"message,omitempty"`
	Entries         []ListedEntry `json:"entries"`
}

// UnregisteredResult is the output of Unregistered.
type UnregisteredResult struct {
	OK  bool     `json:"ok"`
	IDs []string `json:"ids"`
}

// EntryStatus is the status of one registry entry.
type EntryStatus struct {
	ID     string          `json:"id"`
	OK     bool            `json:"ok"`
	Dirty  bool            `json:"dirty,omitempty"`
	Output json.RawMessage `json:"output,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// EntrySystematize is the systematize outcome for one entry.
type EntrySystematize struct {
	ID         string `json:"id"`
	OK         bool   `json:"ok"`
	Action     string `json:"action"` // "skip" or "systematized"
	Dirty      bool   `json:"dirty,omitempty"`
	CorpusPath string `json:"corpus_path,omitempty"`
	Stage      string `json:"stage,omitempty"` // failing stage
	Error      string `json:"error,omitempty"`
}

// SystematizeResult is the output of Systematize.
type SystematizeResult struct {
	OK            bool               `json:"ok"`
	AddCandidates []string           `json:"add_candidates"`
	Entries       []EntrySystematize `json:"entries"`
	Warnings      []string           `json:"warnings,omitempty"`
}

// EffectiveEntries merges yaml entries with the built-in backlog default.
// When yaml already has id=backlog, the yaml entry wins (source=registry).
func EffectiveEntries(f *registry.File, kdbRoot string) []ListedEntry {
	var listed []ListedEntry
	hasBacklog := false
	if f != nil {
		for _, e := range f.KDBs {
			src := SourceRegistry
			listed = append(listed, ListedEntry{Entry: e, Source: src})
			if e.ID == builtin.BacklogID {
				hasBacklog = true
			}
		}
	}
	if !hasBacklog {
		listed = append([]ListedEntry{{
			Entry:  paths.BuiltinBacklogEntry(kdbRoot),
			Source: SourceBuiltin,
		}}, listed...)
	}
	return listed
}

// ApplyEffective rewrites f.KDBs from EffectiveEntries and returns the listed view.
func ApplyEffective(f *registry.File, kdbRoot string) []ListedEntry {
	listed := EffectiveEntries(f, kdbRoot)
	entries := make([]registry.Entry, len(listed))
	for i, le := range listed {
		entries[i] = le.Entry
	}
	if f == nil {
		return listed
	}
	f.KDBs = entries
	return listed
}

// List returns effective registry entries (builtin backlog always included).
func List(f *registry.File, registryPath string, missing bool, kdbRoot string) ListResult {
	listed := ApplyEffective(f, kdbRoot)
	r := ListResult{
		OK:       true,
		Registry: registryPath,
		Entries:  listed,
	}
	if missing {
		r.RegistryMissing = true
		r.Message = "registry file not found: " + registryPath
	}
	return r
}

// Unregistered scans {kdbRoot}/*/corpus and returns ids whose corpus_path
// is not registered. It does not scan features/*/corpus.
func Unregistered(workspace, kdbRoot string, f *registry.File) (UnregisteredResult, error) {
	registered := map[string]bool{}
	if f != nil {
		for _, e := range f.KDBs {
			registered[normRel(e.CorpusPath)] = true
		}
	}
	res := UnregisteredResult{OK: true, IDs: []string{}}

	root := paths.ResolveKdbRoot(kdbRoot)
	base := filepath.Join(workspace, filepath.FromSlash(root))
	dirs, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return res, nil
		}
		return UnregisteredResult{}, fmt.Errorf("scan %s: %w", base, err)
	}
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		corpus := filepath.Join(base, d.Name(), "corpus")
		st, err := os.Stat(corpus)
		if err != nil || !st.IsDir() {
			continue
		}
		rel := root + "/" + d.Name() + "/corpus"
		if !registered[normRel(rel)] {
			res.IDs = append(res.IDs, d.Name())
		}
	}
	sort.Strings(res.IDs)
	return res, nil
}

// StatusCmds returns {"ok":true,"commands":[{id,status_cmd},...]}.
func StatusCmds(f *registry.File) map[string]any {
	cmds := make([]map[string]string, 0, len(f.KDBs))
	for _, e := range f.KDBs {
		cmds = append(cmds, map[string]string{"id": e.ID, "status_cmd": e.StatusCmd})
	}
	return map[string]any{"ok": true, "commands": cmds}
}

// EntryStatusOf computes the status of a single entry (builtin or delegated).
func EntryStatusOf(ctx context.Context, workspace, kdbRoot string, e registry.Entry) EntryStatus {
	st := EntryStatus{ID: e.ID}
	var raw []byte
	if builtin.IsBuiltin(e.ID) {
		p, err := paths.ForEntry(paths.InputFromEnv(workspace, "", kdbRoot), e)
		if err != nil {
			st.Error = err.Error()
			return st
		}
		out, err := builtin.Status(ctx, p)
		if err != nil {
			st.Error = err.Error()
			return st
		}
		raw = out
	} else {
		out, err := delegate.Run(ctx, workspace, e.StatusCmd)
		if err != nil {
			st.Error = err.Error()
			return st
		}
		raw = []byte(out)
	}
	st.OK = true
	st.Output, st.Dirty, st.OK, st.Error = interpretOutput(raw)
	return st
}

// interpretOutput normalises status stdout into valid JSON and extracts
// the dirty flag. Non-JSON output is wrapped as a JSON string. A JSON object
// with "ok": false is reported as a failure.
func interpretOutput(raw []byte) (out json.RawMessage, dirty bool, ok bool, errMsg string) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, false, true, ""
	}
	var obj map[string]any
	if err := json.Unmarshal(trimmed, &obj); err != nil {
		wrapped, _ := json.Marshal(string(trimmed))
		return wrapped, false, true, ""
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, trimmed); err != nil {
		compact.Reset()
		compact.Write(trimmed)
	}
	if d, isBool := obj["dirty"].(bool); isBool {
		dirty = d
	} else if data, isObj := obj["data"].(map[string]any); isObj {
		// status.sh / tt kdb backlog status nest the report under "data".
		if d, isBool := data["dirty"].(bool); isBool {
			dirty = d
		}
	}
	if okv, isBool := obj["ok"].(bool); isBool && !okv {
		msg, _ := obj["error"].(string)
		if msg == "" {
			msg = "status reported ok=false"
		}
		return compact.Bytes(), dirty, false, msg
	}
	return compact.Bytes(), dirty, true, ""
}

// StatusAll runs status for every entry, never stopping early.
// anyFail is true when at least one entry failed.
func StatusAll(ctx context.Context, workspace, kdbRoot string, f *registry.File) (statuses []EntryStatus, anyFail bool) {
	statuses = make([]EntryStatus, 0, len(f.KDBs))
	for _, e := range f.KDBs {
		st := EntryStatusOf(ctx, workspace, kdbRoot, e)
		if !st.OK {
			anyFail = true
		}
		statuses = append(statuses, st)
	}
	return statuses, anyFail
}

// Systematize runs distill->dump->merge->pack for every dirty entry
// (or only onlyID when non-empty). All targeted entries are processed;
// the exit code is 1 when any entry failed, 2 when onlyID is unknown, else 0.
func Systematize(ctx context.Context, workspace, kdbRoot string, f *registry.File, onlyID string) (SystematizeResult, int) {
	res := SystematizeResult{OK: true, AddCandidates: []string{}, Entries: []EntrySystematize{}}

	targets := f.KDBs
	if onlyID != "" {
		e, ok := f.ByID(onlyID)
		if !ok {
			res.OK = false
			res.Warnings = append(res.Warnings, fmt.Sprintf("unknown id %q", onlyID))
			return res, 2
		}
		targets = []registry.Entry{e}
	}

	root := paths.ResolveKdbRoot(kdbRoot)
	if un, err := Unregistered(workspace, kdbRoot, f); err == nil {
		for _, id := range un.IDs {
			res.Warnings = append(res.Warnings, fmt.Sprintf("unregistered corpus: %s/%s/corpus", root, id))
		}
	}

	anyFail := false
	for _, e := range targets {
		r := systematizeEntry(ctx, workspace, kdbRoot, e)
		if !r.OK {
			anyFail = true
		} else if r.Action == "systematized" {
			res.AddCandidates = append(res.AddCandidates, e.CorpusPath)
		}
		res.Entries = append(res.Entries, r)
	}
	res.OK = !anyFail
	if anyFail {
		return res, 1
	}
	return res, 0
}

func systematizeEntry(ctx context.Context, workspace, kdbRoot string, e registry.Entry) EntrySystematize {
	r := EntrySystematize{ID: e.ID, CorpusPath: e.CorpusPath}

	st := EntryStatusOf(ctx, workspace, kdbRoot, e)
	if !st.OK {
		r.Stage, r.Error = "status", st.Error
		return r
	}
	gitDirty, err := CorpusGitDirty(workspace, e.CorpusPath)
	if err != nil {
		r.Stage, r.Error = "git", err.Error()
		return r
	}
	r.Dirty = st.Dirty || gitDirty
	if !r.Dirty {
		r.OK, r.Action = true, "skip"
		return r
	}

	stages := []struct {
		name string
		cmd  string
	}{
		{"distill", e.DistillCmd},
		{"dump", e.DumpCmd},
		{"merge", e.MergeCmd},
		{"pack", e.PackCmd},
	}
	var p paths.Paths
	if builtin.IsBuiltin(e.ID) {
		p, err = paths.ForEntry(paths.InputFromEnv(workspace, "", kdbRoot), e)
		if err != nil {
			r.Stage, r.Error = "paths", err.Error()
			return r
		}
	}
	for _, s := range stages {
		var runErr error
		if builtin.IsBuiltin(e.ID) {
			runErr = builtin.Op(ctx, s.name, p, "")
		} else {
			_, runErr = delegate.Run(ctx, workspace, s.cmd)
		}
		if runErr != nil {
			r.Stage, r.Error = s.name, strings.TrimSpace(runErr.Error())
			return r
		}
	}
	r.OK, r.Action = true, "systematized"
	return r
}

func normRel(p string) string {
	return path.Clean(strings.ReplaceAll(p, "\\", "/"))
}
