package forge

import (
	"github.com/axsh/tokotachi/internal/cmdexec"
)

// NewClientForRepo resolves the forge host for repoRoot and builds a Client.
func NewClientForRepo(repoRoot string, runner *cmdexec.Runner) (Client, Host, error) {
	gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")
	host, _, err := DetectHostFromRepo(repoRoot, gitCmd)
	if err != nil {
		return nil, HostUnknown, err
	}
	c, err := NewClient(Options{Host: host, CmdRunner: runner})
	return c, host, err
}
