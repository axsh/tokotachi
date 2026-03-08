package github

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveToken(t *testing.T) {
	tests := []struct {
		name     string
		envToken string
		wantEnv  bool // if true, expect the env value to be returned
	}{
		{
			name:     "GITHUB_TOKEN set returns env value",
			envToken: "ghp_test123",
			wantEnv:  true,
		},
		{
			name:     "GITHUB_TOKEN empty falls through without panic",
			envToken: "",
			wantEnv:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_TOKEN", tt.envToken)
			if tt.wantEnv {
				assert.Equal(t, tt.envToken, resolveToken())
			} else {
				assert.NotPanics(t, func() { resolveToken() })
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		owner   string
		repo    string
		wantErr bool
	}{
		{
			name:  "valid HTTPS URL",
			url:   "https://github.com/axsh/tokotachi-scaffolds",
			owner: "axsh",
			repo:  "tokotachi-scaffolds",
		},
		{
			name:  "URL with trailing slash",
			url:   "https://github.com/axsh/tokotachi-scaffolds/",
			owner: "axsh",
			repo:  "tokotachi-scaffolds",
		},
		{
			name:    "invalid URL with only one path segment",
			url:     "https://github.com/axsh",
			wantErr: true,
		},
		{
			name:  "empty URL for PR-only client",
			url:   "",
			owner: "",
			repo:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set GITHUB_TOKEN to avoid gh command execution during tests
			t.Setenv("GITHUB_TOKEN", "test-token")
			c, err := NewClient(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.owner, c.Owner)
			assert.Equal(t, tt.repo, c.Repo)
		})
	}
}

func TestClient_FetchFile(t *testing.T) {
	expectedContent := "version: 1.0.0\nscaffolds: []"
	encoded := base64.StdEncoding.EncodeToString([]byte(expectedContent))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/axsh/tokotachi-scaffolds/contents/catalog.yaml", r.URL.Path)
		resp := map[string]any{
			"type":     "file",
			"encoding": "base64",
			"content":  encoded,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	t.Setenv("GITHUB_TOKEN", "test-token")
	c, err := NewClient("https://github.com/axsh/tokotachi-scaffolds",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
	)
	require.NoError(t, err)

	content, err := c.FetchFile("catalog.yaml")
	require.NoError(t, err)
	assert.Equal(t, expectedContent, string(content))
}

func TestClient_FetchFile_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	t.Setenv("GITHUB_TOKEN", "test-token")
	c, err := NewClient("https://github.com/axsh/tokotachi-scaffolds",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
	)
	require.NoError(t, err)

	_, err = c.FetchFile("nonexistent.yaml")
	assert.Error(t, err)
}

func TestClient_FetchDirectory(t *testing.T) {
	fileContent := "# README\nHello"
	encoded := base64.StdEncoding.EncodeToString([]byte(fileContent))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/repos/axsh/tokotachi-scaffolds/contents/templates/project-default/base":
			// Directory listing
			resp := []map[string]any{
				{"name": "README.md", "type": "file", "path": "templates/project-default/base/README.md"},
				{"name": "subdir", "type": "dir", "path": "templates/project-default/base/subdir"},
			}
			json.NewEncoder(w).Encode(resp)

		case "/repos/axsh/tokotachi-scaffolds/contents/templates/project-default/base/subdir":
			// Subdirectory listing
			resp := []map[string]any{
				{"name": "file.txt", "type": "file", "path": "templates/project-default/base/subdir/file.txt"},
			}
			json.NewEncoder(w).Encode(resp)

		case "/repos/axsh/tokotachi-scaffolds/contents/templates/project-default/base/README.md",
			"/repos/axsh/tokotachi-scaffolds/contents/templates/project-default/base/subdir/file.txt":
			// File content
			resp := map[string]any{
				"type":     "file",
				"encoding": "base64",
				"content":  encoded,
			}
			json.NewEncoder(w).Encode(resp)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Setenv("GITHUB_TOKEN", "test-token")
	c, err := NewClient("https://github.com/axsh/tokotachi-scaffolds",
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
	)
	require.NoError(t, err)

	files, err := c.FetchDirectory("templates/project-default/base")
	require.NoError(t, err)
	require.Len(t, files, 2)

	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.RelativePath
	}
	assert.Contains(t, paths, "README.md")
	assert.Contains(t, paths, "subdir/file.txt")
}
