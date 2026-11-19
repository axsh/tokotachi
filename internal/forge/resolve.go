package forge

import (
	"github.com/axsh/tokotachi/internal/cmdexec"
)

// ClientForRepoOpts configures NewClientForRepoOpts.
type ClientForRepoOpts struct {
	RepoRoot    string
	Runner      *cmdexec.Runner
	CLIOverride string // --forge; empty = unset
}

// NewClientForRepoOpts resolves the forge host for the repo (CLI > TT_FORGE > origin URL)
// and builds a Client.
func NewClientForRepoOpts(opts ClientForRepoOpts) (Client, Host, error) {
	gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")
	host, _, err := DetectHostFromRepoOpts(DetectOpts{
		RepoRoot:    opts.RepoRoot,
		GitCmd:      gitCmd,
		CLIOverride: opts.CLIOverride,
	})
	if err != nil {
		return nil, HostUnknown, err
	}
	c, err := NewClient(Options{Host: host, CmdRunner: opts.Runner})
	return c, host, err
}

// NewClientForRepo resolves the forge host for repoRoot and builds a Client.
// Equivalent to NewClientForRepoOpts with an empty CLIOverride.
func NewClientForRepo(repoRoot string, runner *cmdexec.Runner) (Client, Host, error) {
	return NewClientForRepoOpts(ClientForRepoOpts{
		RepoRoot:    repoRoot,
		Runner:      runner,
		CLIOverride: "",
	})
}
