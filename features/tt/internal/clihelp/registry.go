package clihelp

import (
	"fmt"
	"sort"
	"sync"
)

var (
	mu    sync.RWMutex
	pages = map[string]Page{}
)

// Register stores a help page under a cobra path without the root name.
// Example keys: "up", "prompt update", "agent record".
func Register(path string, p Page) {
	if err := Validate(p); err != nil {
		panic(err)
	}
	mu.Lock()
	defer mu.Unlock()
	pages[path] = p
}

// Lookup returns a registered page.
func Lookup(path string) (Page, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := pages[path]
	return p, ok
}

// MustLookup panics when path is missing (for init-time wiring only).
func MustLookup(path string) Page {
	p, ok := Lookup(path)
	if !ok {
		panic(fmt.Sprintf("clihelp: missing page %q", path))
	}
	return p
}

// AllPaths returns registered paths sorted.
func AllPaths() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(pages))
	for k := range pages {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
