package converter

import (
	"os"
	"path/filepath"
)

// RenameDirectories renames program-name-dependent directories in tempDir.
// Currently handles the pattern: cmd/<oldName> -> cmd/<newName>.
// If the directory does not exist, it is silently skipped.
func RenameDirectories(tempDir, oldName, newName string) error {
	if oldName == newName {
		return nil
	}

	oldDir := filepath.Join(tempDir, "cmd", oldName)
	if _, err := os.Stat(oldDir); os.IsNotExist(err) {
		return nil
	}

	newDir := filepath.Join(tempDir, "cmd", newName)
	return os.Rename(oldDir, newDir)
}
