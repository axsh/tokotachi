package forge

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/axsh/tokotachi/internal/cmdexec"
)

// ChangeRequestInfo is host-agnostic PR/MR metadata mapped to CodeStatusPR.
type ChangeRequestInfo struct {
	Number    int
	CreatedAt time.Time
}

// Client creates and queries change requests (GitHub PR or GitLab MR).
type Client interface {
	Host() Host
	CreateChangeRequest(workDir string) error
	FindOpenChangeRequest(workDir, branch string) (*ChangeRequestInfo, error)
}

// Options configures a forge Client.
type Options struct {
	Host      Host
	CmdRunner *cmdexec.Runner
	GhCmd     string // default via ResolveCommand("TT_CMD_GH", "gh")
	GlabCmd   string // default via ResolveCommand("TT_CMD_GLAB", "glab")
}

// NewClient returns a GitHub or GitLab client based on opts.Host.
// HostUnknown selects the GitHub path for backward compatibility.
func NewClient(opts Options) (Client, error) {
	if opts.CmdRunner == nil {
		return nil, fmt.Errorf("CmdRunner is required")
	}
	ghCmd := opts.GhCmd
	if ghCmd == "" {
		ghCmd = cmdexec.ResolveCommand("TT_CMD_GH", "gh")
	}
	glabCmd := opts.GlabCmd
	if glabCmd == "" {
		glabCmd = cmdexec.ResolveCommand("TT_CMD_GLAB", "glab")
	}

	switch opts.Host {
	case HostGitLab:
		return &gitlabClient{runner: opts.CmdRunner, glabCmd: glabCmd, host: HostGitLab}, nil
	default:
		return &githubClient{runner: opts.CmdRunner, ghCmd: ghCmd, host: HostGitHub}, nil
	}
}

type githubClient struct {
	runner *cmdexec.Runner
	ghCmd  string
	host   Host
}

func (c *githubClient) Host() Host { return c.host }

func (c *githubClient) CreateChangeRequest(workDir string) error {
	opts := cmdexec.RunOption{Dir: workDir}
	return c.runner.RunInteractiveWithOpts(opts, c.ghCmd, "pr", "create")
}

func (c *githubClient) FindOpenChangeRequest(workDir, branch string) (*ChangeRequestInfo, error) {
	opts := cmdexec.RunOption{Dir: workDir, QuietCmd: true}
	out, err := c.runner.RunWithOpts(opts, c.ghCmd, "pr", "list",
		"--head", branch,
		"--json", "number,createdAt",
		"--limit", "1",
	)
	if err != nil {
		return nil, fmt.Errorf("gh pr list failed: %w", err)
	}
	return parseGitHubPRListJSON(out)
}

type gitlabClient struct {
	runner  *cmdexec.Runner
	glabCmd string
	host    Host
}

func (c *gitlabClient) Host() Host { return c.host }

func (c *gitlabClient) CreateChangeRequest(workDir string) error {
	opts := cmdexec.RunOption{Dir: workDir}
	return c.runner.RunInteractiveWithOpts(opts, c.glabCmd, "mr", "create")
}

func (c *gitlabClient) FindOpenChangeRequest(workDir, branch string) (*ChangeRequestInfo, error) {
	opts := cmdexec.RunOption{Dir: workDir, QuietCmd: true}
	out, err := c.runner.RunWithOpts(opts, c.glabCmd, "mr", "list",
		"--source-branch", branch,
		"--output", "json",
	)
	if err != nil {
		return nil, fmt.Errorf("glab mr list failed: %w", err)
	}
	return parseGitLabMRListJSON(out)
}

type ghPRJSON struct {
	Number    int       `json:"number"`
	CreatedAt time.Time `json:"createdAt"`
}

type glabMRJSON struct {
	IID       int       `json:"iid"`
	CreatedAt time.Time `json:"created_at"`
}

func parseGitHubPRListJSON(output string) (*ChangeRequestInfo, error) {
	output = strings.TrimSpace(output)
	if output == "" || output == "[]" {
		return nil, nil
	}
	var prs []ghPRJSON
	if err := json.Unmarshal([]byte(output), &prs); err != nil {
		return nil, fmt.Errorf("failed to parse PR list: %w", err)
	}
	if len(prs) == 0 {
		return nil, nil
	}
	return &ChangeRequestInfo{Number: prs[0].Number, CreatedAt: prs[0].CreatedAt}, nil
}

func parseGitLabMRListJSON(output string) (*ChangeRequestInfo, error) {
	output = strings.TrimSpace(output)
	if output == "" || output == "[]" {
		return nil, nil
	}
	var mrs []glabMRJSON
	if err := json.Unmarshal([]byte(output), &mrs); err != nil {
		return nil, fmt.Errorf("failed to parse MR list: %w", err)
	}
	if len(mrs) == 0 {
		return nil, nil
	}
	return &ChangeRequestInfo{Number: mrs[0].IID, CreatedAt: mrs[0].CreatedAt}, nil
}
