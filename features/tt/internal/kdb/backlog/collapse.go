package backlog

import (
	"fmt"
	"sort"
	"strings"
)

// Problem is one validation finding reported by distill.
type Problem struct {
	Key     string `json:"key"`
	Problem string `json:"problem"`
}

// Collapse folds identical duplicates of the same logical key into one item.
// Items sharing a key but differing in content are all kept and the key is
// listed in conflicts (first-seen order). Output keeps first-seen key order.
func Collapse(items []Item) (out []Item, collapsed int, conflicts []Key) {
	seen := map[Key][]Item{}
	var order []Key
	for _, it := range items {
		k := KeyOf(it)
		group, ok := seen[k]
		if !ok {
			seen[k] = []Item{it}
			order = append(order, k)
			continue
		}
		dup := false
		for _, g := range group {
			if g.Equal(it) {
				dup = true
				break
			}
		}
		if dup {
			collapsed++
			continue
		}
		seen[k] = append(group, it)
		if !hasKey(conflicts, k) {
			conflicts = append(conflicts, k)
		}
	}
	for _, k := range order {
		out = append(out, seen[k]...)
	}
	return out, collapsed, conflicts
}

func hasKey(keys []Key, k Key) bool {
	for _, x := range keys {
		if x == k {
			return true
		}
	}
	return false
}

// FindProblems validates items: empty key components, invalid state/kind,
// empty title, empty/invalid/duplicate uuid.
func FindProblems(items []Item) []Problem {
	probs := []Problem{}
	for _, it := range items {
		k := KeyOf(it).String()
		if it.Phase == "" || it.Branch == "" || it.LocalID == "" {
			probs = append(probs, Problem{k, "empty logical key component"})
		}
		if !ValidState(it.State) {
			probs = append(probs, Problem{k, "invalid state: " + it.State})
		}
		if it.Kind != "" && !ValidKind(it.Kind) {
			probs = append(probs, Problem{k, "invalid kind: " + it.Kind})
		}
		if strings.TrimSpace(it.Title) == "" {
			probs = append(probs, Problem{k, "empty title"})
		}
		if it.UUID == "" {
			probs = append(probs, Problem{k, "empty uuid"})
		} else if !ValidUUID(it.UUID) {
			probs = append(probs, Problem{k, "invalid uuid format: " + it.UUID})
		}
	}
	byUUID := map[string][]string{}
	for _, it := range items {
		if it.UUID != "" {
			u := strings.ToLower(it.UUID)
			byUUID[u] = append(byUUID[u], KeyOf(it).String())
		}
	}
	uuids := make([]string, 0, len(byUUID))
	for u := range byUUID {
		uuids = append(uuids, u)
	}
	sort.Strings(uuids)
	for _, u := range uuids {
		if keys := byUUID[u]; len(keys) > 1 {
			probs = append(probs, Problem{strings.Join(keys, ", "), fmt.Sprintf("duplicate uuid: %s", u)})
		}
	}
	return probs
}

func keyStrings(keys []Key) []string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k.String())
	}
	return out
}
