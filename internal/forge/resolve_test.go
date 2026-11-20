package forge

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientForRepoOpts_CLIOverrideBeatsTTForgeAndURL(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "remote", "add", "origin", "https://github.com/a/b.git")
	t.Setenv(EnvForge, "github")

	runner, _ := newTestRunner(t)
	c, host, err := NewClientForRepoOpts(ClientForRepoOpts{
		RepoRoot:    repo,
		Runner:      runner,
		CLIOverride: "gitlab",
	})
	require.NoError(t, err)
	assert.Equal(t, HostGitLab, host)
	assert.Equal(t, HostGitLab, c.Host())
}

func TestNewClientForRepoOpts_EmptyCLI_UsesExistingBehavior(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "remote", "add", "origin", "https://git.example.com/g/p.git")
	t.Setenv(EnvForge, "gitlab")

	runner, _ := newTestRunner(t)
	c, host, err := NewClientForRepoOpts(ClientForRepoOpts{
		RepoRoot:    repo,
		Runner:      runner,
		CLIOverride: "",
	})
	require.NoError(t, err)
	assert.Equal(t, HostGitLab, host)
	assert.Equal(t, HostGitLab, c.Host())
}

func TestNewClientForRepoOpts_InvalidCLI_ReturnsError(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "remote", "add", "origin", "https://github.com/a/b.git")

	runner, _ := newTestRunner(t)
	_, _, err := NewClientForRepoOpts(ClientForRepoOpts{
		RepoRoot:    repo,
		Runner:      runner,
		CLIOverride: "bogus",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--forge")
}

func TestNewClientForRepo_TTForgeGitlab_OverridesNonGitlabURL(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "remote", "add", "origin", "https://git.example.com/g/p.git")

	t.Setenv(EnvForge, "gitlab")
	runner, _ := newTestRunner(t)
	c, host, err := NewClientForRepo(repo, runner)
	require.NoError(t, err)
	assert.Equal(t, HostGitLab, host)
	assert.Equal(t, HostGitLab, c.Host())
}

func TestNewClientForRepo_InvalidTTForge_ReturnsError(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "remote", "add", "origin", "https://github.com/a/b.git")

	t.Setenv(EnvForge, "bitbucket")
	runner, _ := newTestRunner(t)
	_, _, err := NewClientForRepo(repo, runner)
	require.Error(t, err)
}

func TestNewClientForRepo_GitHubURL(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "remote", "add", "origin", "https://github.com/axsh/tokotachi.git")
	_ = os.Unsetenv(EnvForge)

	runner, _ := newTestRunner(t)
	c, host, err := NewClientForRepo(repo, runner)
	require.NoError(t, err)
	assert.Equal(t, HostGitHub, host)
	assert.Equal(t, HostGitHub, c.Host())
}
