package scaffold

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

// Downloader is the interface for fetching files from a template repository.
type Downloader interface {
	// FetchFile retrieves the content of a single file at the given path.
	FetchFile(path string) ([]byte, error)
	// FetchDirectory recursively retrieves all files under the given directory path.
	FetchDirectory(path string) ([]DownloadedFile, error)
}

// DownloadedFile represents a file downloaded from the template repository.
type DownloadedFile struct {
	RelativePath string
	Content      []byte
}

// GitHubDownloader fetches files using the GitHub Contents API.
type GitHubDownloader struct {
	Owner   string
	Repo    string
	Branch  string
	Client  *http.Client
	BaseURL string // Override for testing (default: https://api.github.com)
	Token   string // GitHub API token (from GITHUB_TOKEN env var)
}

// githubContentResponse represents the GitHub API response for a single file.
type githubContentResponse struct {
	Type     string `json:"type"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
	Name     string `json:"name"`
	Path     string `json:"path"`
}

const defaultGitHubAPI = "https://api.github.com"

// NewGitHubDownloader creates a GitHubDownloader from a repository URL.
// The URL should be in the format: https://github.com/{owner}/{repo}
func NewGitHubDownloader(repoURL string) (*GitHubDownloader, error) {
	parsed, err := url.Parse(strings.TrimRight(repoURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL: %w", err)
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid repository URL: expected github.com/{owner}/{repo}, got %s", repoURL)
	}

	return &GitHubDownloader{
		Owner:   parts[0],
		Repo:    parts[1],
		Branch:  "main",
		Client:  &http.Client{Timeout: 30 * time.Second},
		BaseURL: defaultGitHubAPI,
		Token:   os.Getenv("GITHUB_TOKEN"),
	}, nil
}

// newRequest creates an HTTP GET request with optional authentication.
func (d *GitHubDownloader) newRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if d.Token != "" {
		req.Header.Set("Authorization", "Bearer "+d.Token)
	}
	return req, nil
}

// FetchFile retrieves a single file from the repository.
func (d *GitHubDownloader) FetchFile(filePath string) ([]byte, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s",
		d.BaseURL, d.Owner, d.Repo, filePath, d.Branch)

	req, err := d.newRequest(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", filePath, err)
	}

	resp, err := d.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", filePath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: HTTP %d", filePath, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response for %s: %w", filePath, err)
	}

	var content githubContentResponse
	if err := json.Unmarshal(body, &content); err != nil {
		return nil, fmt.Errorf("failed to parse response for %s: %w", filePath, err)
	}

	decoded, err := base64.StdEncoding.DecodeString(
		strings.ReplaceAll(content.Content, "\n", ""))
	if err != nil {
		return nil, fmt.Errorf("failed to decode content for %s: %w", filePath, err)
	}

	return decoded, nil
}

// FetchDirectory recursively retrieves all files under a directory.
func (d *GitHubDownloader) FetchDirectory(dirPath string) ([]DownloadedFile, error) {
	return d.fetchDirectoryRecursive(dirPath, "")
}

// fetchDirectoryRecursive is the internal recursive implementation.
func (d *GitHubDownloader) fetchDirectoryRecursive(dirPath, prefix string) ([]DownloadedFile, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s",
		d.BaseURL, d.Owner, d.Repo, dirPath, d.Branch)

	req, err := d.newRequest(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", dirPath, err)
	}

	resp, err := d.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory %s: %w", dirPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list directory %s: HTTP %d", dirPath, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory listing for %s: %w", dirPath, err)
	}

	var entries []githubContentResponse
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse directory listing for %s: %w", dirPath, err)
	}

	var files []DownloadedFile
	for _, entry := range entries {
		relativePath := path.Join(prefix, entry.Name)

		switch entry.Type {
		case "file":
			content, err := d.FetchFile(entry.Path)
			if err != nil {
				return nil, err
			}
			files = append(files, DownloadedFile{
				RelativePath: relativePath,
				Content:      content,
			})

		case "dir":
			subFiles, err := d.fetchDirectoryRecursive(entry.Path, relativePath)
			if err != nil {
				return nil, err
			}
			files = append(files, subFiles...)
		}
	}

	return files, nil
}
