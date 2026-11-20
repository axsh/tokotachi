package codestatus

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/axsh/tokotachi/internal/forge"
	"github.com/axsh/tokotachi/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockForge struct {
	host forge.Host
	info *forge.ChangeRequestInfo
	err  error
}

func (m *mockForge) Host() forge.Host { return m.host }
func (m *mockForge) CreateChangeRequest(string) error { return nil }
func (m *mockForge) FindOpenChangeRequest(string, string) (*forge.ChangeRequestInfo, error) {
	return m.info, m.err
}

func TestResolve_GitLab_OpenMR(t *testing.T) {
	repo := initRepoWithRemoteBranch(t, "feat-x")
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	c := &Checker{
		GitCmd:   "git",
		RepoRoot: repo,
		Timeout:  10 * time.Second,
		Host:     forge.HostGitLab,
		Forge: &mockForge{
			host: forge.HostGitLab,
			info: &forge.ChangeRequestInfo{Number: 42, CreatedAt: created},
		},
	}

	bs, err := c.Resolve(context.Background(), "feat-x", state.CodeStatusHosted)
	require.NoError(t, err)
	assert.Equal(t, state.CodeStatusPR, bs.Status)
	require.NotNil(t, bs.PRCreatedAt)
	assert.True(t, bs.PRCreatedAt.UTC().Equal(created))
}

func TestResolve_GitHub_OpenPR(t *testing.T) {
	repo := initRepoWithRemoteBranch(t, "feat-y")
	created := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	c := &Checker{
		GitCmd:   "git",
		RepoRoot: repo,
		Timeout:  10 * time.Second,
		Host:     forge.HostGitHub,
		Forge: &mockForge{
			host: forge.HostGitHub,
			info: &forge.ChangeRequestInfo{Number: 7, CreatedAt: created},
		},
	}

	bs, err := c.Resolve(context.Background(), "feat-y", state.CodeStatusLocal)
	require.NoError(t, err)
	assert.Equal(t, state.CodeStatusPR, bs.Status)
}

func TestResolve_ForgeCLIFailure_FallsBackToHosted(t *testing.T) {
	repo := initRepoWithRemoteBranch(t, "feat-z")
	c := &Checker{
		GitCmd:   "git",
		RepoRoot: repo,
		Timeout:  10 * time.Second,
		Forge: &mockForge{
			host: forge.HostGitLab,
			err:  fmt.Errorf("glab failed"),
		},
	}

	bs, err := c.Resolve(context.Background(), "feat-z", state.CodeStatusLocal)
	require.NoError(t, err)
	assert.Equal(t, state.CodeStatusHosted, bs.Status)
}

func TestResolve_NoRemoteBranch_LocalOrDeleted(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init")
	// bare remote without the branch
	remote := t.TempDir()
	runGit(t, remote, "init", "--bare")
	runGit(t, repo, "remote", "add", "origin", remote)

	c := &Checker{
		GitCmd:   "git",
		RepoRoot: repo,
		Timeout:  10 * time.Second,
		Forge:    &mockForge{host: forge.HostGitHub},
	}

	bs, err := c.Resolve(context.Background(), "missing", state.CodeStatusLocal)
	require.NoError(t, err)
	assert.Equal(t, state.CodeStatusLocal, bs.Status)

	bs, err = c.Resolve(context.Background(), "missing", state.CodeStatusPR)
	require.NoError(t, err)
	assert.Equal(t, state.CodeStatusDeleted, bs.Status)
}

func initRepoWithRemoteBranch(t *testing.T, branch string) string {
	t.Helper()
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "test")
	require.NoError(t, os.WriteFile(repo+"/README", []byte("x"), 0o644))
	runGit(t, repo, "add", "README")
	runGit(t, repo, "commit", "-m", "init")

	remote := t.TempDir()
	runGit(t, remote, "init", "--bare")
	runGit(t, repo, "remote", "add", "origin", remote)
	runGit(t, repo, "branch", "-M", branch)
	runGit(t, repo, "push", "-u", "origin", branch)
	return repo
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
}
