package copier

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyDir recursively copies the contents of srcDir into destDir.
// destDir must already exist.
// Returns error if srcDir does not exist or is not a directory.
func CopyDir(srcDir, destDir string) error {
	info, err := os.Stat(srcDir)
	if err != nil {
		return fmt.Errorf("source directory does not exist: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source path is not a directory: %s", srcDir)
	}

	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path for %s: %w", path, err)
		}

		destPath := filepath.Join(destDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}

		return copyFile(path, destPath)
	})
}

// copyFile copies a single file from src to dest.
func copyFile(src, dest string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", dest, err)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dest, err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file %s to %s: %w", src, dest, err)
	}

	return nil
}
