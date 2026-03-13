package scaffold

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFetcher is a test double for EntryFetcher.
type mockFetcher struct {
	entries map[string]*ScaffoldEntry // key = "category/name"
}

func (m *mockFetcher) FetchEntry(category, name string) (*ScaffoldEntry, error) {
	key := category + "/" + name
	if e, ok := m.entries[key]; ok {
		return e, nil
	}
	return nil, fmt.Errorf("not found: %s", key)
}

func TestResolveDependencies_NoDeps(t *testing.T) {
	entry := &ScaffoldEntry{
		Name:     "default",
		Category: "root",
	}
	fetcher := &mockFetcher{entries: map[string]*ScaffoldEntry{}}

	result, err := ResolveDependencies(fetcher, entry)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "default", result[0].Name)
	assert.Equal(t, "root", result[0].Category)
}

func TestResolveDependencies_SingleChain(t *testing.T) {
	// A -> B -> C (no deps)
	entryC := &ScaffoldEntry{
		Name:     "default",
		Category: "root",
	}
	entryB := &ScaffoldEntry{
		Name:     "axsh-go-standard",
		Category: "project",
		DependsOn: []Dependency{
			{Category: "root", Name: "default"},
		},
	}
	entryA := &ScaffoldEntry{
		Name:     "axsh-go-standard",
		Category: "feature",
		DependsOn: []Dependency{
			{Category: "project", Name: "axsh-go-standard"},
		},
	}

	fetcher := &mockFetcher{entries: map[string]*ScaffoldEntry{
		"root/default":             entryC,
		"project/axsh-go-standard": entryB,
		"feature/axsh-go-standard": entryA,
	}}

	result, err := ResolveDependencies(fetcher, entryA)
	require.NoError(t, err)
	require.Len(t, result, 3)
	// Topological order: C first, B second, A last
	assert.Equal(t, "root", result[0].Category)
	assert.Equal(t, "default", result[0].Name)
	assert.Equal(t, "project", result[1].Category)
	assert.Equal(t, "axsh-go-standard", result[1].Name)
	assert.Equal(t, "feature", result[2].Category)
	assert.Equal(t, "axsh-go-standard", result[2].Name)
}

func TestResolveDependencies_CircularDependency(t *testing.T) {
	// A -> B -> A (circular)
	entryA := &ScaffoldEntry{
		Name:     "alpha",
		Category: "cat",
		DependsOn: []Dependency{
			{Category: "cat", Name: "beta"},
		},
	}
	entryB := &ScaffoldEntry{
		Name:     "beta",
		Category: "cat",
		DependsOn: []Dependency{
			{Category: "cat", Name: "alpha"},
		},
	}

	fetcher := &mockFetcher{entries: map[string]*ScaffoldEntry{
		"cat/alpha": entryA,
		"cat/beta":  entryB,
	}}

	_, err := ResolveDependencies(fetcher, entryA)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestResolveDependencies_DiamondDependency(t *testing.T) {
	// A -> {B, C}, B -> D, C -> D
	// D should appear only once in the result
	entryD := &ScaffoldEntry{
		Name:     "delta",
		Category: "cat",
	}
	entryB := &ScaffoldEntry{
		Name:     "beta",
		Category: "cat",
		DependsOn: []Dependency{
			{Category: "cat", Name: "delta"},
		},
	}
	entryC := &ScaffoldEntry{
		Name:     "gamma",
		Category: "cat",
		DependsOn: []Dependency{
			{Category: "cat", Name: "delta"},
		},
	}
	entryA := &ScaffoldEntry{
		Name:     "alpha",
		Category: "cat",
		DependsOn: []Dependency{
			{Category: "cat", Name: "beta"},
			{Category: "cat", Name: "gamma"},
		},
	}

	fetcher := &mockFetcher{entries: map[string]*ScaffoldEntry{
		"cat/alpha": entryA,
		"cat/beta":  entryB,
		"cat/gamma": entryC,
		"cat/delta": entryD,
	}}

	result, err := ResolveDependencies(fetcher, entryA)
	require.NoError(t, err)

	// D should appear exactly once
	deltaCount := 0
	for _, e := range result {
		if e.Category == "cat" && e.Name == "delta" {
			deltaCount++
		}
	}
	assert.Equal(t, 1, deltaCount, "delta should appear exactly once")

	// A should be last
	assert.Equal(t, "alpha", result[len(result)-1].Name)

	// D should be first (no deps)
	assert.Equal(t, "delta", result[0].Name)

	// Total should be 4: D, B, C, A
	assert.Len(t, result, 4)
}
