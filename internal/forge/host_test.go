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
