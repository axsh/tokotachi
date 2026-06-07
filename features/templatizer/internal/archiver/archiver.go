package archiver

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ZipDirectory creates a ZIP archive of srcDir and writes it to destPath.
// The archive contains files with paths relative to srcDir.
// If destPath already exists, it will be overwritten.
// Returns error if srcDir does not exist or is not a directory.
func ZipDirectory(srcDir, destPath string) error {
	info, err := os.Stat(srcDir)
	if err != nil {
		return fmt.Errorf("source directory does not exist: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source path is not a directory: %s", srcDir)
	}

	// Ensure parent directory of destination exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create ZIP file: %w", err)
	}
	defer outFile.Close()

	w := zip.NewWriter(outFile)
	defer w.Close()

	err = filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip directories themselves; only add files
		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path for %s: %w", path, err)
		}
		// Use forward slashes for cross-platform ZIP compatibility
		relPath = filepath.ToSlash(relPath)

		fw, err := w.Create(relPath)
		if err != nil {
			return fmt.Errorf("failed to create ZIP entry %s: %w", relPath, err)
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", path, err)
		}
		defer f.Close()

		if _, err := io.Copy(fw, f); err != nil {
			return fmt.Errorf("failed to write file %s to ZIP: %w", relPath, err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory %s: %w", srcDir, err)
	}

	return nil
}
