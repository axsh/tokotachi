package scaffold

import "fmt"

// EntryFetcher retrieves a ScaffoldEntry by category and name.
// In production, this uses the GitHub API via shard path computation.
// In tests, a mock implementation can be injected.
type EntryFetcher interface {
	FetchEntry(category, name string) (*ScaffoldEntry, error)
}

// ResolveDependencies recursively resolves the dependency chain of entry,
// returning entries in topological order (dependencies first, entry last).
// Returns an error if a circular dependency is detected.
func ResolveDependencies(fetcher EntryFetcher, entry *ScaffoldEntry) ([]ScaffoldEntry, error) {
	visited := make(map[string]bool) // tracks nodes on current recursion stack
	seen := make(map[string]bool)    // tracks nodes already added to result
	var result []ScaffoldEntry

	var resolve func(e *ScaffoldEntry) error
	resolve = func(e *ScaffoldEntry) error {
		key := e.Category + "/" + e.Name

		if visited[key] {
			return fmt.Errorf("circular dependency detected: %s", key)
		}
		if seen[key] {
			return nil // already resolved (dedup)
		}

		visited[key] = true

		// Resolve dependencies first (depth-first)
		for _, dep := range e.DependsOn {
			depEntry, err := fetcher.FetchEntry(dep.Category, dep.Name)
			if err != nil {
				return fmt.Errorf("failed to fetch dependency %s/%s: %w",
					dep.Category, dep.Name, err)
			}
			if err := resolve(depEntry); err != nil {
				return err
			}
		}

		// Backtrack: remove from recursion stack
		visited[key] = false
		// Add self to result
		seen[key] = true
		result = append(result, *e)
		return nil
	}

	if err := resolve(entry); err != nil {
		return nil, err
	}

	return result, nil
}
