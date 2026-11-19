// Package backlog implements the builtin backlog knowledge DB: an embedded
// SQLite cache, a JSON array pipeline (distill/dump/merge/pack/unpack/restore),
// and the CRUD operations behind `tt kdb backlog <op>`.
//
// The algorithms mirror the former scripts/kdb-backlog/bl_json.py so that
// corpora produced by either implementation have identical content hashes.
package backlog

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Columns is the column order of the items table (and of every JSON item).
var Columns = []string{
	"phase", "branch", "local_id", "slug", "title", "state", "state_date", "kind",
	"ask_to", "due", "origin", "decision", "problem", "current_handling", "options",
	"questions", "after_decision", "history", "created_at", "updated_at", "uuid",
}

// States are the allowed item states (data contract string literals).
var States = []string{"未確認", "確認中", "決着", "取り下げ"}

// Kinds are the allowed item kinds (data contract string literals).
var Kinds = []string{"仮の決定", "確認事項", "先送り"}

// SortKeys are the keys accepted by list --sort.
var SortKeys = []string{"local_id", "updated_at", "created_at", "state", "title"}

const (
	// ChunkSize is the number of items per corpus chunk file.
	ChunkSize = 100
	// FormatVersion is the corpus manifest format version.
	FormatVersion = 1

	// StateUnconfirmed is the default state of a new item.
	StateUnconfirmed = "未確認"
	// StateSettled marks a settled item.
	StateSettled = "決着"
	// StateWithdrawn marks a withdrawn item.
	StateWithdrawn = "取り下げ"
)

var (
	uuidRE    = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	nameRE    = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	localIDRE = regexp.MustCompile(`^[0-9]{3,}$`)
	dateRE    = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
)

// Item is one backlog record. Field order equals Columns.
type Item struct {
	Phase           string  `json:"phase"`
	Branch          string  `json:"branch"`
	LocalID         string  `json:"local_id"`
	Slug            string  `json:"slug"`
	Title           string  `json:"title"`
	State           string  `json:"state"`
	StateDate       *string `json:"state_date"` // nil (JSON null) differs from ""
	Kind            string  `json:"kind"`
	AskTo           string  `json:"ask_to"`
	Due             string  `json:"due"`
	Origin          string  `json:"origin"`
	Decision        string  `json:"decision"`
	Problem         string  `json:"problem"`
	CurrentHandling string  `json:"current_handling"`
	Options         string  `json:"options"`
	Questions       string  `json:"questions"`
	AfterDecision   string  `json:"after_decision"`
	History         string  `json:"history"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
	UUID            string  `json:"uuid"`
}

// Key is the logical key of an item.
type Key struct{ Phase, Branch, LocalID string }

// String renders the key as phase/branch/local_id.
func (k Key) String() string { return k.Phase + "/" + k.Branch + "/" + k.LocalID }

// KeyOf returns the logical key of an item.
func KeyOf(it Item) Key { return Key{it.Phase, it.Branch, it.LocalID} }

// StrPtr returns a pointer to s.
func StrPtr(s string) *string { return &s }

// Equal reports whether two items hold identical column values.
func (it Item) Equal(o Item) bool {
	if (it.StateDate == nil) != (o.StateDate == nil) {
		return false
	}
	if it.StateDate != nil && *it.StateDate != *o.StateDate {
		return false
	}
	a, b := it, o
	a.StateDate, b.StateDate = nil, nil
	return a == b
}

// NormItem builds an Item from a decoded JSON object (bl_json.norm_item).
// Missing or null columns become "" (state defaults to 未確認, state_date to
// nil); scalar values are stringified; nested arrays/objects are rejected.
func NormItem(m map[string]any) (Item, error) {
	vals := make(map[string]string, len(Columns))
	var stateDate *string
	for _, col := range Columns {
		raw, present := m[col]
		if raw != nil {
			if _, bad := raw.([]any); bad {
				return Item{}, fmt.Errorf("column %q must be a scalar value", col)
			}
			if _, bad := raw.(map[string]any); bad {
				return Item{}, fmt.Errorf("column %q must be a scalar value", col)
			}
		}
		s := scalarString(raw)
		switch col {
		case "state_date":
			if raw != nil && s != "" {
				v := s
				stateDate = &v
			}
		case "state":
			if !present || raw == nil {
				s = StateUnconfirmed
			}
			vals[col] = s
		default:
			vals[col] = s
		}
	}
	return Item{
		Phase: vals["phase"], Branch: vals["branch"], LocalID: vals["local_id"],
		Slug: vals["slug"], Title: vals["title"], State: vals["state"], StateDate: stateDate,
		Kind: vals["kind"], AskTo: vals["ask_to"], Due: vals["due"], Origin: vals["origin"],
		Decision: vals["decision"], Problem: vals["problem"], CurrentHandling: vals["current_handling"],
		Options: vals["options"], Questions: vals["questions"], AfterDecision: vals["after_decision"],
		History: vals["history"], CreatedAt: vals["created_at"], UpdatedAt: vals["updated_at"],
		UUID: vals["uuid"],
	}, nil
}

// scalarString mimics Python str() for decoded JSON scalars.
func scalarString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case json.Number:
		return x.String()
	case bool:
		if x {
			return "True"
		}
		return "False"
	default:
		return fmt.Sprint(x)
	}
}

// SortItems returns items sorted by (phase, branch, local_id); stable.
func SortItems(items []Item) []Item {
	out := make([]Item, len(items))
	copy(out, items)
	sort.SliceStable(out, func(i, j int) bool { return keyLess(KeyOf(out[i]), KeyOf(out[j])) })
	return out
}

func keyLess(a, b Key) bool {
	if a.Phase != b.Phase {
		return a.Phase < b.Phase
	}
	if a.Branch != b.Branch {
		return a.Branch < b.Branch
	}
	return a.LocalID < b.LocalID
}

// ValidState reports whether s is one of States.
func ValidState(s string) bool { return contains(States, s) }

// ValidKind reports whether s is one of Kinds.
func ValidKind(s string) bool { return contains(Kinds, s) }

// ValidUUID reports whether s is a canonical RFC 4122 uuid string.
func ValidUUID(s string) bool { return uuidRE.MatchString(s) }

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// CheckState returns a usage error unless s is an allowed state.
func CheckState(s string) error {
	if !ValidState(s) {
		return usageErr("invalid state '%s' (allowed: %s)", s, strings.Join(States, " "))
	}
	return nil
}

// CheckKind returns a usage error unless s is an allowed kind.
func CheckKind(s string) error {
	if !ValidKind(s) {
		return usageErr("invalid kind '%s' (allowed: %s)", s, strings.Join(Kinds, " "))
	}
	return nil
}

// CheckUUID returns a usage error unless s is a canonical RFC 4122 string.
func CheckUUID(s string) error {
	if !ValidUUID(s) {
		return usageErr("invalid uuid '%s' (expected RFC 4122 form, e.g. 550e8400-e29b-41d4-a716-446655440000)", s)
	}
	return nil
}

// CheckLocalID returns a usage error unless s is zero-padded digits (3+).
func CheckLocalID(s string) error {
	if !localIDRE.MatchString(s) {
		return usageErr("invalid id '%s' (expected zero-padded digits, at least 3)", s)
	}
	return nil
}

// CheckName validates a phase or branch directory name.
func CheckName(label, v string) error {
	if !nameRE.MatchString(v) {
		return usageErr("invalid %s '%s' (allowed: letters, digits, dot, underscore, hyphen)", label, v)
	}
	return nil
}

// CheckDate returns a usage error unless s is YYYY-MM-DD shaped.
func CheckDate(s string) error {
	if !dateRE.MatchString(s) {
		return usageErr("invalid date '%s' (expected YYYY-MM-DD)", s)
	}
	return nil
}
