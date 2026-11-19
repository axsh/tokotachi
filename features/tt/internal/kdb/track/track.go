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

// ListResult is the output of List.
type ListResult struct {
	OK      bool             `json:"ok"`
	Entries []registry.Entry `json:"entries"`
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

// List returns all registry entries.
func List(f *registry.File) ListResult {
	entries := f.KDBs
	if entries == nil {
		entries = []registry.Entry{}
	}
	return ListResult{OK: true, Entries: entries}
}

// Unregistered scans workspace/features/*/corpus and returns ids whose
// features/{id}/corpus path is not a registered corpus_path.
func Unregistered(workspace string, f *registry.File) (UnregisteredResult, error) {
	registered := map[string]bool{}
	for _, e := range f.KDBs {
		registered[normRel(e.CorpusPath)] = true
	}
	res := UnregisteredResult{OK: true, IDs: []string{}}

	featuresDir := filepath.Join(workspace, "features")
	dirs, err := os.ReadDir(featuresDir)
	if err != nil {
		if os.IsNotExist(err) {
			return res, nil
		}
		return UnregisteredResult{}, fmt.Errorf("scan %s: %w", featuresDir, err)
	}
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		corpus := filepath.Join(featuresDir, d.Name(), "corpus")
		st, err := os.Stat(corpus)
		if err != nil || !st.IsDir() {
			continue
		}
		rel := "features/" + d.Name() + "/corpus"
		if !registered[rel] {
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
func EntryStatusOf(ctx context.Context, workspace string, e registry.Entry) EntryStatus {
	st := EntryStatus{ID: e.ID}
	var raw []byte
	if builtin.IsBuiltin(e.ID) {
		p, err := paths.ForEntry(paths.InputFromEnv(workspace, ""), e)
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
func StatusAll(ctx context.Context, workspace string, f *registry.File) (statuses []EntryStatus, anyFail bool) {
	statuses = make([]EntryStatus, 0, len(f.KDBs))
	for _, e := range f.KDBs {
		st := EntryStatusOf(ctx, workspace, e)
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
func Systematize(ctx context.Context, workspace string, f *registry.File, onlyID string) (SystematizeResult, int) {
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

	if un, err := Unregistered(workspace, f); err == nil {
		for _, id := range un.IDs {
			res.Warnings = append(res.Warnings, fmt.Sprintf("unregistered corpus: features/%s/corpus", id))
		}
	}

	anyFail := false
	for _, e := range targets {
		r := systematizeEntry(ctx, workspace, e)
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

func systematizeEntry(ctx context.Context, workspace string, e registry.Entry) EntrySystematize {
	r := EntrySystematize{ID: e.ID, CorpusPath: e.CorpusPath}

	st := EntryStatusOf(ctx, workspace, e)
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
		p, err = paths.ForEntry(paths.InputFromEnv(workspace, ""), e)
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
