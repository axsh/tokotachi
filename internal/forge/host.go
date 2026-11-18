package forge

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// Host identifies the remote forge kind.
type Host string

const (
	HostGitHub  Host = "github"
	HostGitLab  Host = "gitlab"
	HostUnknown Host = "unknown"
)

// EnvForge is the environment variable for explicit forge selection.
// Valid values: "github" | "gitlab" (case-insensitive).
const EnvForge = "TT_FORGE"

// DetectHost classifies a git remote URL into HostGitHub, HostGitLab, or HostUnknown.
func DetectHost(remoteURL string) Host {
	host := extractHostname(remoteURL)
	if host == "" {
		return HostUnknown
	}

	h := strings.ToLower(host)
	if h == "github.com" || strings.HasSuffix(h, ".github.com") {
		return HostGitHub
	}
	if h == "gitlab.com" || strings.Contains(h, "gitlab") {
		return HostGitLab
	}
	return HostUnknown
}

// ParseForgeOverride parses TT_FORGE-style values.
// Empty string => HostUnknown, nil error (means "no override").
// Invalid value => HostUnknown, non-nil error.
func ParseForgeOverride(value string) (Host, error) {
	v := strings.ToLower(strings.TrimSpace(value))
	switch v {
	case "":
		return HostUnknown, nil
	case "github":
		return HostGitHub, nil
	case "gitlab":
		return HostGitLab, nil
	default:
		return HostUnknown, fmt.Errorf("invalid TT_FORGE %q: want github or gitlab", value)
	}
}

// ResolveHost applies priority: TT_FORGE override > DetectHost(remoteURL).
func ResolveHost(override, remoteURL string) (Host, error) {
	h, err := ParseForgeOverride(override)
	if err != nil {
		return HostUnknown, err
	}
	if h == HostGitHub || h == HostGitLab {
		return h, nil
	}
	return DetectHost(remoteURL), nil
}

// DetectHostFromRepo runs `git remote get-url origin` then ResolveHost(os.Getenv(EnvForge), url).
// When remote lookup fails but a valid override is set, the override wins and error is nil.
func DetectHostFromRepo(repoRoot, gitCmd string) (Host, string, error) {
	override := os.Getenv(EnvForge)

	remoteURL, remoteErr := getOriginURL(repoRoot, gitCmd)
	host, err := ResolveHost(override, remoteURL)
	if err != nil {
		return HostUnknown, remoteURL, err
	}

	if remoteErr != nil {
		if host == HostGitHub || host == HostGitLab {
			return host, "", nil
		}
		return HostUnknown, "", remoteErr
	}
	return host, remoteURL, nil
}

func getOriginURL(repoRoot, gitCmd string) (string, error) {
	if gitCmd == "" {
		gitCmd = "git"
	}
	cmd := exec.Command(gitCmd, "remote", "get-url", "origin")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git remote get-url origin: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func extractHostname(remoteURL string) string {
	raw := strings.TrimSpace(remoteURL)
	if raw == "" {
		return ""
	}

	// SSH form: git@host:path
	if strings.HasPrefix(raw, "git@") {
		rest := strings.TrimPrefix(raw, "git@")
		host, _, ok := strings.Cut(rest, ":")
		if !ok {
			return ""
		}
		return host
	}

	// scp-like without scheme is uncommon; prefer URL parse for http(s) and ssh://
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if u.Host != "" {
		// strip port if present
		return u.Hostname()
	}
	return ""
}
