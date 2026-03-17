package release_note_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// projectRoot returns the absolute path to the project root.
// Derived from this file's location: tests/release-note/ -> 2 levels up.
func projectRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get caller information")
	}
	// helpers_test.go is in tests/release-note/
	dir := filepath.Dir(filename)
	root, err := filepath.Abs(filepath.Join(dir, "..", ".."))
	if err != nil {
		panic(fmt.Sprintf("failed to resolve project root: %v", err))
	}
	return root
}

// TestMain runs all tests.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
