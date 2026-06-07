package catalog

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// DiscoveryResult represents a discovered originals directory and its scaffolds.
type DiscoveryResult struct {
	OriginalsDir string               // absolute path to the originals/ directory
	BaseDir      string               // parent of originals/ (output target)
	Definitions  []ScaffoldDefinition // scaffolds found in this originals/
}

// DiscoverOriginals walks searchRoot recursively to find directories named "originals".
// Returns an error if zero or more than one originals directories are found.
// When exactly one originals directory is found, it scans for scaffold.yaml files
// and returns the result.
func DiscoverOriginals(searchRoot string) (*DiscoveryResult, error) {
	absRoot, err := filepath.Abs(searchRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve search root: %w", err)
	}

	var found []string
	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == "originals" {
			found = append(found, path)
			// Skip descending into the originals directory to avoid
			// counting nested originals (e.g. originals/sub/originals/).
			return fs.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory tree: %w", err)
	}

	switch len(found) {
	case 0:
		return nil, fmt.Errorf("no originals directory found under %s", absRoot)
	case 1:
		// OK, continue below.
	default:
		var paths []string
		for _, p := range found {
			paths = append(paths, "  - "+p)
		}
		return nil, fmt.Errorf(
			"multiple originals directories found under %s:\n%s",
			absRoot, strings.Join(paths, "\n"),
		)
	}

	originalsDir := found[0]
	baseDir := filepath.Dir(originalsDir)

	defs, err := ScanScaffoldDefinitions(originalsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to scan scaffold definitions in %s: %w", originalsDir, err)
	}

	return &DiscoveryResult{
		OriginalsDir: originalsDir,
		BaseDir:      baseDir,
		Definitions:  defs,
	}, nil
}
