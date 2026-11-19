package backlog

import (
	"errors"
	"os"
)

// DistillReport is the per-source report of distill (bl_json cmd_distill).
type DistillReport struct {
	Source    string    `json:"source"`
	Items     int       `json:"items"`
	Collapsed int       `json:"collapsed"`
	Problems  []Problem `json:"problems"`
	Rewritten bool      `json:"rewritten,omitempty"`
}

// StatusReport is the data member of the status output.
type StatusReport struct {
	Dirty      bool     `json:"dirty"`
	ItemCount  int      `json:"item_count"`
	CorpusHash string   `json:"corpus_hash"`
	DBHash     string   `json:"db_hash"`
	DataHash   string   `json:"data_hash"`
	Reasons    []string `json:"reasons"`
}

func distillItems(items []Item, label string) (DistillReport, []Item, int) {
	collapsedItems, collapsed, conflicts := Collapse(items)
	probs := FindProblems(collapsedItems)
	for _, k := range conflicts {
		probs = append(probs, Problem{Key: k.String(), Problem: "duplicate logical key with different content"})
	}
	return DistillReport{Source: label, Items: len(collapsedItems), Collapsed: collapsed, Problems: probs},
		collapsedItems, collapsed
}

// DistillResult runs distill and returns the reports. problems is true when
// any problem was found (the CLI then fails with exit 1).
func (o Ops) DistillResult() (db DistillReport, data *DistillReport, problems bool, err error) {
	s, err := o.openStore()
	if err != nil {
		return db, nil, false, err
	}
	defer s.Close()
	items, err := s.ListAll()
	if err != nil {
		return db, nil, false, bizErr("database error: %v", err)
	}
	db, _, _ = distillItems(items, "database")
	problems = len(db.Problems) > 0

	if o.P.DataJSON != "" {
		if _, statErr := os.Stat(o.P.DataJSON); statErr == nil {
			dataItems, err := ReadItems(o.P.DataJSON)
			if err != nil {
				return db, nil, problems, err
			}
			rep, collapsedItems, collapsed := distillItems(dataItems, "data/items.json")
			if len(rep.Problems) == 0 && collapsed > 0 {
				if err := WriteItems(o.P.DataJSON, SortItems(collapsedItems)); err != nil {
					return db, nil, problems, err
				}
				rep.Rewritten = true
			}
			if len(rep.Problems) > 0 {
				problems = true
			}
			data = &rep
		}
	}
	return db, data, problems, nil
}

// Distill validates the database and data/items.json (distill.sh).
func (o Ops) Distill() (map[string]any, error) {
	db, data, problems, err := o.DistillResult()
	if err != nil {
		return nil, err
	}
	if problems {
		reports := []any{db}
		if data != nil {
			reports = append(reports, *data)
		}
		return nil, bizErrExtra(map[string]any{"reports": reports}, "distill found problems")
	}
	var dataField any
	if data != nil {
		dataField = *data
	}
	return map[string]any{"data": map[string]any{"db": db, "data_json": dataField}}, nil
}

// Dump exports the cache as one JSON array to the tmp directory (dump.sh).
func (o Ops) Dump() (map[string]any, error) {
	s, err := o.openStore()
	if err != nil {
		return nil, err
	}
	defer s.Close()
	n, err := s.DumpJSON(o.P.TmpJSON)
	if err != nil {
		var oe *OpError
		if errors.As(err, &oe) {
			return nil, err
		}
		return nil, bizErr("database error: %v", err)
	}
	return map[string]any{"data": map[string]any{"path": o.P.TmpJSON, "item_count": n}}, nil
}

// MergeOutcome is the result of Merge: the summary plus whether the cache was restored.
type MergeOutcome struct {
	Summary  MergeSummary
	Restored bool
}

// MergeResult merges tmp and data and, without conflicts, restores the cache.
// With conflicts it returns the outcome (Restored=false) and ErrMergeConflict.
func (o Ops) MergeResult(prefer string) (MergeOutcome, error) {
	var out MergeOutcome
	if prefer != "" && prefer != "tmp" && prefer != "data" {
		return out, usageErr("--prefer must be tmp or data")
	}
	if _, err := os.Stat(o.P.TmpJSON); err != nil {
		return out, bizErr("dump not found: %s (run dump first)", o.P.TmpJSON)
	}
	// The cache is opened first (like bl_ensure_db) so a broken path fails early.
	s, err := o.openStore()
	if err != nil {
		return out, err
	}
	defer s.Close()
	sum, err := MergeFiles(o.P.TmpJSON, o.P.DataJSON, o.P.DataJSON, prefer)
	out.Summary = sum
	if errors.Is(err, ErrMergeConflict) {
		return out, ErrMergeConflict
	}
	if err != nil {
		return out, err
	}
	if _, err := restoreInto(s, o.P.DataJSON); err != nil {
		return out, err
	}
	out.Restored = true
	return out, nil
}

// Merge is merge.sh: on conflicts it still succeeds with restored=false and
// conflict_count > 0 (the user contract of the former merge.sh).
func (o Ops) Merge(prefer string) (map[string]any, error) {
	out, err := o.MergeResult(prefer)
	if err != nil && !errors.Is(err, ErrMergeConflict) {
		return nil, err
	}
	return map[string]any{"data": map[string]any{"merge": out.Summary, "restored": out.Restored}}, nil
}

// Pack writes the corpus from data/items.json (pack.sh).
func (o Ops) Pack() (map[string]any, error) {
	if _, err := os.Stat(o.P.DataJSON); err != nil {
		return nil, bizErr("data/items.json not found: %s (run merge or unpack first)", o.P.DataJSON)
	}
	items, err := ReadItems(o.P.DataJSON)
	if err != nil {
		return nil, err
	}
	res, err := Pack(items, o.P.CorpusDir)
	if err != nil {
		return nil, err
	}
	return map[string]any{"data": res}, nil
}

// Unpack expands the corpus into data/items.json (unpack.sh).
func (o Ops) Unpack() (map[string]any, error) {
	if err := os.MkdirAll(o.P.DataDir, 0o755); err != nil {
		return nil, bizErr("create %s: %v", o.P.DataDir, err)
	}
	res, err := Unpack(o.P.CorpusDir, o.P.DataJSON)
	if err != nil {
		return nil, err
	}
	return map[string]any{"data": res}, nil
}

// restoreInto replaces every row of s with the items of dataJSON.
func restoreInto(s *Store, dataJSON string) (int, error) {
	items, err := ReadItems(dataJSON)
	if err != nil {
		return 0, err
	}
	_, collapsed, conflicts := Collapse(items)
	if collapsed > 0 || len(conflicts) > 0 {
		return 0, bizErrExtra(map[string]any{"conflicts": keyStrings(conflicts), "collapsed": collapsed},
			"duplicate logical keys in %s; run distill first", dataJSON)
	}
	if err := s.ReplaceAll(items); err != nil {
		return 0, bizErr("database error: %v", err)
	}
	return len(items), nil
}

// Restore rebuilds the cache from data/items.json (restore.sh).
func (o Ops) Restore() (map[string]any, error) {
	if _, err := os.Stat(o.P.DataJSON); err != nil {
		return nil, bizErr("data/items.json not found: %s (run unpack or merge first)", o.P.DataJSON)
	}
	s, err := o.openStore()
	if err != nil {
		return nil, err
	}
	defer s.Close()
	if _, err := restoreInto(s, o.P.DataJSON); err != nil {
		return nil, err
	}
	n, err := s.Count()
	if err != nil {
		return nil, bizErr("database error: %v", err)
	}
	return map[string]any{"data": map[string]any{"restored": n, "engine": "sqlite"}}, nil
}

// StatusReport computes the dirty state of the cache against the corpus and data/items.json.
func (o Ops) StatusReport() (StatusReport, error) {
	var rep StatusReport
	s, err := o.openStore()
	if err != nil {
		return rep, err
	}
	defer s.Close()
	items, err := s.ListAll()
	if err != nil {
		return rep, bizErr("database error: %v", err)
	}
	rep.ItemCount = len(items)
	rep.DBHash = ContentHashOf(items)
	rep.Reasons = []string{}
	manifest, err := LoadManifest(o.P.CorpusDir)
	if err != nil {
		return rep, err
	}
	if manifest == nil {
		rep.Reasons = append(rep.Reasons, "corpus manifest missing")
	} else {
		rep.CorpusHash = manifest.ContentHash
		if rep.CorpusHash != rep.DBHash {
			rep.Reasons = append(rep.Reasons, "db content differs from corpus")
		}
	}
	if o.P.DataJSON != "" {
		if _, statErr := os.Stat(o.P.DataJSON); statErr == nil {
			dataItems, err := ReadItems(o.P.DataJSON)
			if err != nil {
				return rep, err
			}
			rep.DataHash = ContentHashOf(dataItems)
			if rep.DataHash != rep.DBHash {
				rep.Reasons = append(rep.Reasons, "db content differs from data/items.json")
			}
		}
	}
	rep.Dirty = len(rep.Reasons) > 0
	return rep, nil
}

// Status is status.sh.
func (o Ops) Status() (map[string]any, error) {
	rep, err := o.StatusReport()
	if err != nil {
		return nil, err
	}
	return map[string]any{"data": rep}, nil
}
