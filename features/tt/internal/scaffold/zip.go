package scaffold

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// ExtractZip extracts files from a ZIP archive byte slice.
// Directory entries are skipped; only file entries are returned.
func ExtractZip(data []byte) ([]DownloadedFile, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip archive: %w", err)
	}

	var files []DownloadedFile
	for _, f := range reader.File {
		// Skip directory entries
		if strings.HasSuffix(f.Name, "/") {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s in zip: %w", f.Name, err)
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s from zip: %w", f.Name, err)
		}

		files = append(files, DownloadedFile{
			RelativePath: f.Name,
			Content:      content,
		})
	}

	return files, nil
}
