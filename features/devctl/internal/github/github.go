package github

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/axsh/tokotachi/features/devctl/internal/cmdexec"
)

// DownloadedFile represents a file downloaded from a GitHub repository.
type DownloadedFile struct {
	RelativePath string
	Content      []byte
}

// Client centralizes GitHub operations.
// It provides HTTP API-based repository content access and gh CLI-based PR operations.
// Internal implementation details (HTTP vs gh CLI) are hidden from callers.
type Client struct {
	Owner     string
	Repo      string
	Branch    string
	Client    *http.Client
	BaseURL   string // Override for testing (default: https://api.github.com)
	token     string
	cmdRunner *cmdexec.Runner // For PR operations (optional)
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

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithCmdRunner sets the cmdexec.Runner for gh CLI operations (e.g. CreatePR).
func WithCmdRunner(r *cmdexec.Runner) ClientOption {
	return func(c *Client) {
		c.cmdRunner = r
	}
}

// WithBaseURL overrides the GitHub API base URL (for testing).
func WithBaseURL(u string) ClientOption {
	return func(c *Client) {
		c.BaseURL = u
	}
}

// WithHTTPClient overrides the HTTP client (for testing).
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.Client = hc
	}
}

// NewClient creates a Client from a repository URL.
// If repoURL is empty, a PR-only client is returned (owner/repo are left empty).
// Token is automatically resolved via GITHUB_TOKEN env var or gh auth token command.
func NewClient(repoURL string, opts ...ClientOption) (*Client, error) {
	c := &Client{
		Branch:  "main",
		Client:  &http.Client{Timeout: 30 * time.Second},
		BaseURL: defaultGitHubAPI,
		token:   resolveToken(),
	}

	if repoURL != "" {
		parsed, err := url.Parse(strings.TrimRight(repoURL, "/"))
		if err != nil {
			return nil, fmt.Errorf("invalid repository URL: %w", err)
		}

		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid repository URL: expected github.com/{owner}/{repo}, got %s", repoURL)
		}

		c.Owner = parts[0]
		c.Repo = parts[1]
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// resolveToken resolves a GitHub API token using the following priority:
// 1. GITHUB_TOKEN environment variable
// 2. gh auth token command output
// 3. empty string (unauthenticated access)
func resolveToken() string {
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}

	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output))
	}

	return ""
}

// newRequest creates an HTTP GET request with optional authentication.
func (c *Client) newRequest(reqURL string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return req, nil
}

// FetchFile retrieves a single file from the repository.
func (c *Client) FetchFile(filePath string) ([]byte, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s",
		c.BaseURL, c.Owner, c.Repo, filePath, c.Branch)

	req, err := c.newRequest(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", filePath, err)
	}

	resp, err := c.Client.Do(req)
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
func (c *Client) FetchDirectory(dirPath string) ([]DownloadedFile, error) {
	return c.fetchDirectoryRecursive(dirPath, "")
}

// fetchDirectoryRecursive is the internal recursive implementation.
func (c *Client) fetchDirectoryRecursive(dirPath, prefix string) ([]DownloadedFile, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s",
		c.BaseURL, c.Owner, c.Repo, dirPath, c.Branch)

	req, err := c.newRequest(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", dirPath, err)
	}

	resp, err := c.Client.Do(req)
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
			content, err := c.FetchFile(entry.Path)
			if err != nil {
				return nil, err
			}
			files = append(files, DownloadedFile{
				RelativePath: relativePath,
				Content:      content,
			})

		case "dir":
			subFiles, err := c.fetchDirectoryRecursive(entry.Path, relativePath)
			if err != nil {
				return nil, err
			}
			files = append(files, subFiles...)
		}
	}

	return files, nil
}

// CreatePR creates a Pull Request interactively using gh CLI.
// Requires WithCmdRunner to be set on the client.
func (c *Client) CreatePR(workDir string) error {
	if c.cmdRunner == nil {
		return fmt.Errorf("CmdRunner is required for PR operations")
	}

	ghCmd := cmdexec.ResolveCommand("DEVCTL_CMD_GH", "gh")
	opts := cmdexec.RunOption{Dir: workDir}
	return c.cmdRunner.RunInteractiveWithOpts(opts, ghCmd, "pr", "create")
}

// PRInfo holds information about a pull request.
type PRInfo struct {
	Number    int       `json:"number"`
	CreatedAt time.Time `json:"createdAt"`
}

// ListPRs returns open PRs matching the given head branch.
// Uses `gh pr list --head <branch> --json number,createdAt --limit 1`.
// If cmdRunner is set, uses it; otherwise falls back to exec.Command directly.
func (c *Client) ListPRs(workDir, branch string) ([]PRInfo, error) {
	ghCmd := cmdexec.ResolveCommand("DEVCTL_CMD_GH", "gh")
	args := []string{"pr", "list", "--head", branch, "--json", "number,createdAt", "--limit", "1"}

	var output string
	var err error

	if c.cmdRunner != nil {
		opts := cmdexec.RunOption{Dir: workDir, QuietCmd: true}
		output, err = c.cmdRunner.RunWithOpts(opts, ghCmd, args...)
	} else {
		cmd := exec.Command(ghCmd, args...)
		if workDir != "" {
			cmd.Dir = workDir
		}
		out, execErr := cmd.Output()
		output = strings.TrimSpace(string(out))
		err = execErr
	}

	if err != nil {
		return nil, fmt.Errorf("gh pr list failed: %w", err)
	}

	if output == "" || output == "[]" {
		return nil, nil
	}

	var prs []PRInfo
	if err := json.Unmarshal([]byte(output), &prs); err != nil {
		return nil, fmt.Errorf("failed to parse PR list: %w", err)
	}
	return prs, nil
}
