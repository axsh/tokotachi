package forge

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectHost(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want Host
	}{
		{name: "https github", url: "https://github.com/axsh/tokotachi.git", want: HostGitHub},
		{name: "ssh github", url: "git@github.com:axsh/tokotachi.git", want: HostGitHub},
		{name: "https gitlab.com", url: "https://gitlab.com/group/proj.git", want: HostGitLab},
		{name: "ssh gitlab.com", url: "git@gitlab.com:group/proj.git", want: HostGitLab},
		{name: "self-hosted gitlab hostname", url: "https://gitlab.example.com/g/p.git", want: HostGitLab},
		{name: "non-gitlab custom host", url: "https://git.example.com/g/p.git", want: HostUnknown},
		{name: "empty", url: "", want: HostUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, DetectHost(tt.url))
		})
	}
}

func TestParseForgeOverride(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    Host
		wantErr bool
	}{
		{name: "gitlab lower", value: "gitlab", want: HostGitLab},
		{name: "GitLab mixed", value: "GitLab", want: HostGitLab},
		{name: "GITHUB upper", value: "GITHUB", want: HostGitHub},
		{name: "empty means no override", value: "", want: HostUnknown, wantErr: false},
		{name: "whitespace only", value: "  ", want: HostUnknown, wantErr: false},
		{name: "invalid bitbucket", value: "bitbucket", wantErr: true},
		{name: "invalid foo", value: "foo", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseForgeOverride(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveHost_OverrideBeatsURL(t *testing.T) {
	tests := []struct {
		name     string
		override string
		url      string
		want     Host
		wantErr  bool
	}{
		{
			name:     "TT_FORGE=gitlab overrides non-gitlab URL",
			override: "gitlab",
			url:      "https://git.example.com/g/p.git",
			want:     HostGitLab,
		},
		{
			name:     "TT_FORGE=github overrides gitlab URL",
			override: "github",
			url:      "https://gitlab.com/g/p.git",
			want:     HostGitHub,
		},
		{
			name:     "unset with non-gitlab URL",
			override: "",
			url:      "https://git.example.com/g/p.git",
			want:     HostUnknown,
		},
		{
			name:     "unset with gitlab URL",
			override: "",
			url:      "https://gitlab.com/g/p.git",
			want:     HostGitLab,
		},
		{
			name:     "invalid override",
			override: "bitbucket",
			url:      "https://gitlab.com/g/p.git",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveHost(tt.override, tt.url)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveHostPriority(t *testing.T) {
	tests := []struct {
		name        string
		cli         string
		env         string
		url         string
		want        Host
		wantErr     bool
		errContains string
	}{
		{
			name: "cli gitlab beats env github and github URL",
			cli:  "gitlab",
			env:  "github",
			url:  "https://github.com/a/b.git",
			want: HostGitLab,
		},
		{
			name: "cli github beats env gitlab and gitlab URL",
			cli:  "github",
			env:  "gitlab",
			url:  "https://gitlab.com/g/p.git",
			want: HostGitHub,
		},
		{
			name: "empty cli uses env over URL",
			cli:  "",
			env:  "gitlab",
			url:  "https://git.example.com/g/p.git",
			want: HostGitLab,
		},
		{
			name: "empty cli and env uses gitlab URL",
			cli:  "",
			env:  "",
			url:  "https://gitlab.com/g/p.git",
			want: HostGitLab,
		},
		{
			name: "empty cli and env with non-gitlab URL",
			cli:  "",
			env:  "",
			url:  "https://git.example.com/g/p.git",
			want: HostUnknown,
		},
		{
			name:        "invalid cli fails fast even when env valid",
			cli:         "bogus",
			env:         "gitlab",
			url:         "https://gitlab.com/g/p.git",
			wantErr:     true,
			errContains: "--forge",
		},
		{
			name:        "invalid env errors when cli empty",
			cli:         "",
			env:         "bogus",
			url:         "https://gitlab.com/g/p.git",
			wantErr:     true,
			errContains: "TT_FORGE",
		},
		{
			name: "cli trim and case insensitive",
			cli:  "  GitLab  ",
			env:  "",
			url:  "https://github.com/a/b.git",
			want: HostGitLab,
		},
		{
			name: "empty everything",
			cli:  "",
			env:  "",
			url:  "",
			want: HostUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveHostPriority(tt.cli, tt.env, tt.url)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDetectHostFromRepoOpts_CLIOverrideBeatsEnvAndURL(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "remote", "add", "origin", "https://github.com/a/b.git")
	t.Setenv(EnvForge, "github")

	host, _, err := DetectHostFromRepoOpts(DetectOpts{
		RepoRoot:    repo,
		GitCmd:      "git",
		CLIOverride: "gitlab",
	})
	require.NoError(t, err)
	assert.Equal(t, HostGitLab, host)
}

func TestDetectHostFromRepoOpts_EmptyCLI_UsesEnv(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "remote", "add", "origin", "https://git.example.com/g/p.git")
	t.Setenv(EnvForge, "gitlab")

	host, _, err := DetectHostFromRepoOpts(DetectOpts{
		RepoRoot:    repo,
		GitCmd:      "git",
		CLIOverride: "",
	})
	require.NoError(t, err)
	assert.Equal(t, HostGitLab, host)
}

func TestDetectHostFromRepoOpts_InvalidCLI_ReturnsError(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "remote", "add", "origin", "https://github.com/a/b.git")

	_, _, err := DetectHostFromRepoOpts(DetectOpts{
		RepoRoot:    repo,
		GitCmd:      "git",
		CLIOverride: "bogus",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--forge")
}

func TestDetectHostFromRepo_UsesOriginAndOverride(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "remote", "add", "origin", "https://git.example.com/g/p.git")

	t.Setenv(EnvForge, "gitlab")
	host, remoteURL, err := DetectHostFromRepo(repo, "git")
	require.NoError(t, err)
	assert.Equal(t, HostGitLab, host)
	assert.Contains(t, remoteURL, "git.example.com")
}

func TestDetectHostFromRepo_OverrideWithoutRemote(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init")
	// no remote

	t.Setenv(EnvForge, "gitlab")
	host, remoteURL, err := DetectHostFromRepo(repo, "git")
	require.NoError(t, err)
	assert.Equal(t, HostGitLab, host)
	assert.Equal(t, "", remoteURL)
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, out)
	_ = filepath.Base(dir)
}
