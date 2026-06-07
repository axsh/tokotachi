package converter

import (
	"io/fs"
	"os"
	"path/filepath"
)

// DefaultExcludes is the list of file/directory names to remove during cleanup.
var DefaultExcludes = []string{
	".git",
	"go.sum",
	"vendor",
	"bin",
	".DS_Store",
}

// Clean removes files and directories in tempDir whose base name matches
// any entry in the excludes list. It walks the directory tree and removes
// matches using os.RemoveAll, skipping into removed directories.
func Clean(tempDir string, excludes []string) error {
	excludeSet := make(map[string]bool, len(excludes))
	for _, e := range excludes {
		excludeSet[e] = true
	}

	return filepath.WalkDir(tempDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Do not remove the root directory itself.
		if path == tempDir {
			return nil
		}

		name := d.Name()
		if excludeSet[name] {
			if removeErr := os.RemoveAll(path); removeErr != nil {
				return removeErr
			}
			if d.IsDir() {
				return fs.SkipDir
			}
		}

		return nil
	})
}
