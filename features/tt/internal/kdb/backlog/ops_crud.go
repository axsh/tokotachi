package backlog

import (
	"strings"

	"github.com/axsh/tokotachi/features/tt/internal/kdb/paths"
)

// Ops runs backlog operations against one resolved set of paths.
// Each op returns the JSON fields to be wrapped as {"ok":true,...}, or an
// error (an *OpError carries the exit code and optional extra JSON members).
type Ops struct {
	P paths.Paths
}

// Flag names accepted by each CRUD op (identical to the former *.sh scripts).
var (
	AddFlags = []string{"phase", "branch", "id", "slug", "title", "kind", "ask-to", "due", "origin",
		"problem", "current-handling", "options", "questions", "after-decision", "history", "state",
		"state-date", "decision"}
	GetFlags    = []string{"uuid", "phase", "branch", "id"}
	UpdateFlags = []string{"phase", "branch", "id", "slug", "title", "state", "state-date", "kind", "ask-to",
		"due", "origin", "decision", "problem", "current-handling", "options", "questions",
		"after-decision", "history", "append-history"}
	ResolveFlags = []string{"phase", "branch", "id", "state", "decision"}
	ListFlags    = []string{"phase", "branch", "state", "kind", "match", "include-closed", "limit", "offset", "sort", "order"}
)

func (o Ops) openStore() (*Store, error) {
	if o.P.DBPath == "" {
		return nil, usageErr("cannot resolve the backlog database path (set BL_DB_PATH or fix the registry entry)")
	}
	return Open(o.P.DBPath)
}

// Add inserts a new item (add.sh).
func (o Ops) Add(f Flags) (map[string]any, error) {
	if err := f.Need("phase", "branch", "title", "kind"); err != nil {
		return nil, err
	}
	if err := CheckName("phase", f.Get("phase")); err != nil {
		return nil, err
	}
	if err := CheckName("branch", f.Get("branch")); err != nil {
		return nil, err
	}
	if err := CheckKind(f.Get("kind")); err != nil {
		return nil, err
	}
	state := StateUnconfirmed
	if f.Get("state") != "" {
		state = f.Get("state")
	}
	if err := CheckState(state); err != nil {
		return nil, err
	}
	today := o.P.Today
	stateDate := today
	if f.Get("state-date") != "" {
		stateDate = f.Get("state-date")
	}
	if err := CheckDate(stateDate); err != nil {
		return nil, err
	}
	idGiven := f.Has("id")
	if idGiven {
		if err := CheckLocalID(f.Get("id")); err != nil {
			return nil, err
		}
	}
	s, err := o.openStore()
	if err != nil {
		return nil, err
	}
	defer s.Close()

	maxAttempts := 2
	if idGiven {
		maxAttempts = 1
	}
	for attempt := 1; ; attempt++ {
		localID := f.Get("id")
		if !idGiven {
			if localID, err = s.NextLocalID(f.Get("phase"), f.Get("branch")); err != nil {
				return nil, bizErr("database error: %v", err)
			}
		}
		_, exists, err := s.GetByKey(f.Get("phase"), f.Get("branch"), localID)
		if err != nil {
			return nil, bizErr("database error: %v", err)
		}
		if exists {
			return nil, bizErr("item already exists: %s/%s/%s", f.Get("phase"), f.Get("branch"), localID)
		}
		id := NewUUID()
		if err := CheckUUID(id); err != nil {
			return nil, err
		}
		it := Item{
			Phase: f.Get("phase"), Branch: f.Get("branch"), LocalID: localID,
			Slug: f.Get("slug"), Title: f.Get("title"), State: state, StateDate: StrPtr(stateDate),
			Kind: f.Get("kind"), AskTo: f.Get("ask-to"), Due: f.Get("due"), Origin: f.Get("origin"),
			Decision: f.Get("decision"), Problem: f.Get("problem"), CurrentHandling: f.Get("current-handling"),
			Options: f.Get("options"), Questions: f.Get("questions"), AfterDecision: f.Get("after-decision"),
			History: f.Get("history"), CreatedAt: today, UpdatedAt: today, UUID: id,
		}
		if err := s.Insert(it); err != nil {
			if strings.Contains(err.Error(), "UNIQUE") && attempt < maxAttempts {
				continue
			}
			return nil, bizErr("%v", err)
		}
		got, ok, err := s.GetByUUID(id)
		if err != nil || !ok {
			return nil, bizErr("inserted item could not be read back: %v", err)
		}
		return map[string]any{"item": got}, nil
	}
}

// Get returns one item by uuid and/or logical key (get.sh).
func (o Ops) Get(f Flags) (map[string]any, error) {
	haveUUID := f.Get("uuid") != ""
	haveKey := f.Get("phase") != "" || f.Get("branch") != "" || f.Get("id") != ""
	if !haveUUID && !haveKey {
		return nil, usageErr("give --uuid, or --phase --branch --id")
	}
	if haveKey {
		if err := f.Need("phase", "branch", "id"); err != nil {
			return nil, err
		}
		if err := CheckName("phase", f.Get("phase")); err != nil {
			return nil, err
		}
		if err := CheckName("branch", f.Get("branch")); err != nil {
			return nil, err
		}
		if err := CheckLocalID(f.Get("id")); err != nil {
			return nil, err
		}
	}
	uuidVal := ""
	if haveUUID {
		if err := CheckUUID(f.Get("uuid")); err != nil {
			return nil, err
		}
		uuidVal = strings.ToLower(f.Get("uuid"))
	}
	s, err := o.openStore()
	if err != nil {
		return nil, err
	}
	defer s.Close()

	if haveUUID {
		item, ok, err := s.GetByUUID(uuidVal)
		if err != nil {
			return nil, bizErr("database error: %v", err)
		}
		if !ok {
			return nil, bizErr("item not found: uuid %s", uuidVal)
		}
		if haveKey {
			keyed, kok, err := s.GetByKey(f.Get("phase"), f.Get("branch"), f.Get("id"))
			if err != nil {
				return nil, bizErr("database error: %v", err)
			}
			if !kok || !keyed.Equal(item) {
				return nil, bizErr("uuid %s does not match key %s/%s/%s", uuidVal, f.Get("phase"), f.Get("branch"), f.Get("id"))
			}
		}
		return map[string]any{"item": item}, nil
	}
	item, ok, err := s.GetByKey(f.Get("phase"), f.Get("branch"), f.Get("id"))
	if err != nil {
		return nil, bizErr("database error: %v", err)
	}
	if !ok {
		return nil, bizErr("item not found: %s/%s/%s", f.Get("phase"), f.Get("branch"), f.Get("id"))
	}
	return map[string]any{"item": item}, nil
}

// Update changes selected fields of one item (update.sh).
func (o Ops) Update(f Flags) (map[string]any, error) {
	if err := f.Need("phase", "branch", "id"); err != nil {
		return nil, err
	}
	if err := CheckName("phase", f.Get("phase")); err != nil {
		return nil, err
	}
	if err := CheckName("branch", f.Get("branch")); err != nil {
		return nil, err
	}
	if err := CheckLocalID(f.Get("id")); err != nil {
		return nil, err
	}
	if f.Has("title") && f.Get("title") == "" {
		return nil, usageErr("--title must not be empty")
	}
	if f.Has("state") {
		if err := CheckState(f.Get("state")); err != nil {
			return nil, err
		}
		if st := f.Get("state"); st == StateSettled || st == StateWithdrawn {
			return nil, usageErr("state '%s' cannot be set with update; use resolve to close an item", st)
		}
	}
	if f.Has("kind") {
		if err := CheckKind(f.Get("kind")); err != nil {
			return nil, err
		}
	}
	if f.Has("state-date") && f.Get("state-date") != "" {
		if err := CheckDate(f.Get("state-date")); err != nil {
			return nil, err
		}
	}
	fieldFlags := []string{"slug", "title", "state", "kind", "ask-to", "due", "origin", "decision",
		"problem", "current-handling", "options", "questions", "after-decision", "history", "state-date"}
	changed := 0
	for _, n := range fieldFlags {
		if f.Has(n) {
			changed++
		}
	}
	if f.Has("append-history") {
		if f.Has("history") {
			return nil, usageErr("--history and --append-history cannot be combined")
		}
		changed++
	}
	if changed == 0 {
		return nil, usageErr("no field flag given; nothing to update")
	}

	s, err := o.openStore()
	if err != nil {
		return nil, err
	}
	defer s.Close()
	it, ok, err := s.GetByKey(f.Get("phase"), f.Get("branch"), f.Get("id"))
	if err != nil {
		return nil, bizErr("database error: %v", err)
	}
	if !ok {
		return nil, bizErr("item not found: %s/%s/%s", f.Get("phase"), f.Get("branch"), f.Get("id"))
	}

	setStr := func(flag string, dst *string) {
		if f.Has(flag) {
			*dst = f.Get(flag)
		}
	}
	setStr("slug", &it.Slug)
	setStr("title", &it.Title)
	setStr("state", &it.State)
	setStr("kind", &it.Kind)
	setStr("ask-to", &it.AskTo)
	setStr("due", &it.Due)
	setStr("origin", &it.Origin)
	setStr("decision", &it.Decision)
	setStr("problem", &it.Problem)
	setStr("current-handling", &it.CurrentHandling)
	setStr("options", &it.Options)
	setStr("questions", &it.Questions)
	setStr("after-decision", &it.AfterDecision)
	setStr("history", &it.History)
	if f.Has("state-date") {
		if f.Get("state-date") == "" {
			it.StateDate = nil
		} else {
			it.StateDate = StrPtr(f.Get("state-date"))
		}
	}
	if f.Has("append-history") {
		if it.History == "" {
			it.History = f.Get("append-history")
		} else {
			it.History = it.History + "\n" + f.Get("append-history")
		}
	}
	it.UpdatedAt = o.P.Today
	if err := s.Update(it); err != nil {
		return nil, bizErr("database error: %v", err)
	}
	got, _, err := s.GetByKey(it.Phase, it.Branch, it.LocalID)
	if err != nil {
		return nil, bizErr("database error: %v", err)
	}
	return map[string]any{"item": got}, nil
}

// Resolve closes an item as settled or withdrawn (resolve.sh).
func (o Ops) Resolve(f Flags) (map[string]any, error) {
	if err := f.Need("phase", "branch", "id", "state"); err != nil {
		return nil, err
	}
	if err := CheckName("phase", f.Get("phase")); err != nil {
		return nil, err
	}
	if err := CheckName("branch", f.Get("branch")); err != nil {
		return nil, err
	}
	if err := CheckLocalID(f.Get("id")); err != nil {
		return nil, err
	}
	if st := f.Get("state"); st != StateSettled && st != StateWithdrawn {
		return nil, usageErr("invalid state '%s' for resolve (allowed: 決着, 取り下げ)", st)
	}
	s, err := o.openStore()
	if err != nil {
		return nil, err
	}
	defer s.Close()
	it, ok, err := s.GetByKey(f.Get("phase"), f.Get("branch"), f.Get("id"))
	if err != nil {
		return nil, bizErr("database error: %v", err)
	}
	if !ok {
		return nil, bizErr("item not found: %s/%s/%s", f.Get("phase"), f.Get("branch"), f.Get("id"))
	}
	it.State = f.Get("state")
	it.StateDate = StrPtr(o.P.Today)
	it.UpdatedAt = o.P.Today
	if f.Has("decision") {
		it.Decision = f.Get("decision")
	}
	if err := s.Update(it); err != nil {
		return nil, bizErr("database error: %v", err)
	}
	got, _, err := s.GetByKey(it.Phase, it.Branch, it.LocalID)
	if err != nil {
		return nil, bizErr("database error: %v", err)
	}
	return map[string]any{"item": got}, nil
}

// List returns filtered, sorted and paged items (list.sh + list-filter).
func (o Ops) List(f Flags) (map[string]any, error) {
	if f.Has("state") {
		if err := CheckState(f.Get("state")); err != nil {
			return nil, err
		}
	}
	if f.Has("kind") {
		if err := CheckKind(f.Get("kind")); err != nil {
			return nil, err
		}
	}
	s, err := o.openStore()
	if err != nil {
		return nil, err
	}
	defer s.Close()
	all, err := s.ListAll()
	if err != nil {
		return nil, bizErr("database error: %v", err)
	}
	includeClosed := f.Get("include-closed") == "1"
	kept := make([]Item, 0, len(all))
	for _, it := range all {
		if f.Has("phase") && it.Phase != f.Get("phase") {
			continue
		}
		if f.Has("branch") && it.Branch != f.Get("branch") {
			continue
		}
		if f.Has("state") && it.State != f.Get("state") {
			continue
		}
		if f.Has("kind") && it.Kind != f.Get("kind") {
			continue
		}
		if !f.Has("state") && !includeClosed && (it.State == StateSettled || it.State == StateWithdrawn) {
			continue
		}
		kept = append(kept, it)
	}
	page, err := ListFilter(kept, ListOptions{
		Match: f.Get("match"), Sort: f.Get("sort"), Order: f.Get("order"),
		Limit: f.Get("limit"), Offset: f.Get("offset"),
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"total": page.Total, "limit": page.Limit, "offset": page.Offset, "items": page.Items}, nil
}

// ImportMD imports legacy markdown backlog files into the database (import-md.sh).
func (o Ops) ImportMD(paths []string) (map[string]any, error) {
	if len(paths) == 0 {
		return nil, usageErr("missing PATH argument")
	}
	items, res, err := ParseMarkdownItems(paths, o.P.Today)
	if err != nil {
		return nil, err
	}
	s, err := o.openStore()
	if err != nil {
		return nil, err
	}
	defer s.Close()
	if err := s.UpsertAll(items); err != nil {
		return nil, bizErr("database error: %v", err)
	}
	return map[string]any{"imported": res.Imported, "files": res.Files, "warnings": res.Warnings, "keys": res.Keys}, nil
}

// MigrateUUIDs assigns a uuid to items without a valid one in a JSON array
// file and drops the legacy logical_uuid member (bl_json migrate-uuids).
func (o Ops) MigrateUUIDs(f Flags) (map[string]any, error) {
	if err := f.Need("in"); err != nil {
		return nil, err
	}
	in := f.Get("in")
	out := f.Get("out")
	if out == "" {
		out = in
	}
	if out == "-" {
		return nil, usageErr("--out must be a file path")
	}
	raw, err := readRawInput(in)
	if err != nil {
		return nil, err
	}
	maps, err := parseRawObjects(raw, in)
	if err != nil {
		return nil, err
	}
	assigned, removed := 0, 0
	items := make([]Item, 0, len(maps))
	for _, m := range maps {
		if cur, ok := m["uuid"].(string); !ok || !ValidUUID(cur) {
			m["uuid"] = NewUUID()
			assigned++
		}
		if _, ok := m["logical_uuid"]; ok {
			delete(m, "logical_uuid")
			removed++
		}
		it, err := NormItem(m)
		if err != nil {
			return nil, bizErr("%s: %v", in, err)
		}
		items = append(items, it)
	}
	if err := WriteItems(out, items); err != nil {
		return nil, err
	}
	return map[string]any{"count": len(items), "assigned": assigned, "logical_uuid_removed": removed, "out": out}, nil
}