package codestatus_test

import (
	"os"
	"path/filepath"
)

// writeTestFile creates a file at the given path with the given content.
// Creates parent directories as needed.
func writeTestFile(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
