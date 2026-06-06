package emitter

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeFileWithMode writes content to path according to the given mode.
//
// overwrite: always write (current behavior)
// skip:      write only if file does not exist
// immune:    always write (orphan cleanup is handled separately after emit)
func writeFileWithMode(path, content string, mode EmitMode) error {
	if mode == EmitModeSkip {
		if _, err := os.Stat(path); err == nil {
			fmt.Fprintf(os.Stderr, "SKIP: file already exists: %s\n", path)
			return nil
		}
	}
	return writeFile(path, content)
}

// CleanOrphanFiles scans targetDirs for files not in emittedFiles
// and deletes them (or warns only if dryRun).
// Returns the list of orphan file paths found.
func CleanOrphanFiles(targetDirs []string, emittedFiles map[string]bool, dryRun bool) ([]string, error) {
	var orphans []string
	for _, dir := range targetDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			// Skip non-content files
			if info.Name() == "README.md" || info.Name() == ".gitkeep" {
				return nil
			}
			cleanPath := filepath.Clean(path)
			if !emittedFiles[cleanPath] {
				orphans = append(orphans, cleanPath)
				if dryRun {
					fmt.Fprintf(os.Stderr, "ORPHAN (dry-run): %s\n", cleanPath)
				} else {
					fmt.Fprintf(os.Stderr, "ORPHAN: removing %s\n", cleanPath)
					if rmErr := os.Remove(cleanPath); rmErr != nil {
						return fmt.Errorf("failed to remove orphan %s: %w", cleanPath, rmErr)
					}
				}
			}
			return nil
		})
		if err != nil {
			return orphans, err
		}
	}
	return orphans, nil
}
