package scaffold

import "github.com/axsh/tokotachi/internal/github"

// DownloadedFile is a type alias for github.DownloadedFile.
// This maintains backward compatibility with existing scaffold code.
type DownloadedFile = github.DownloadedFile

// Downloader is the interface for fetching files from a template repository.
// github.Client implements this interface implicitly.
type Downloader interface {
	FetchFile(path string) ([]byte, error)
	FetchDirectory(path string) ([]DownloadedFile, error)
}
